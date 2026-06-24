# Design — Фича 5: Трекер целей с учётом внеплановых пополнений

- **Дата:** 2026-06-25
- **Этап:** 4 / Фича 5 (BUSINESS_LOGIC.md §5)
- **Стек:** Go 1.26 + chi v5 + shopspring/decimal + PostgreSQL 16
- **Статус:** утверждён пользователем (полный охват, аналитика + инфляция, идемпотентность по contribution_id, гибридная модель current_amount)

---

## 1. Цель и охват

Реализовать Level-2 калькулятор «Трекер целей» (BUSINESS_LOGIC.md §5) — первый из
калькуляторов Этапа 4. Пользователь создаёт финансовую цель (сумма, срок,
регулярный взнос, ожидаемая доходность, опционально инфляция), приложение
считает: требуемый взнос / срок / будущую сумму формулой аннуитета (фонд
возмещения, Копнова Гл. 3.3.3), мгновенно пересчитывает при внеплановых
пополнениях и показывает прогресс + историю.

### Охват (все подсистемы за один шаг)

1. CRUD целей (`goals` таблица уже существует в миграции 0001)
2. Журнал внеплановых пополнений (новая таблица `goal_contributions`)
3. Проекция статуса цели (`GET /goals/{id}/projection`)
4. What-if симуляция (stateless): по сохранённой цели и без неё

### Зафиксированные решения

| Решение | Выбор | Обоснование |
|---|---|---|
| Охват | Всё сразу | запрос пользователя |
| Математика | Аналитика + инфляция | sinking fund решается аналитически; численный solver (BrentQ) — overkill для MVP, не добавляем 3-й float64-bridge |
| Идемпотентность пополнений | `contribution_id` (client-generated) + UNIQUE | консистентно с ф.1 (`calc_id`) |
| Модель `current_amount` | Гибрид: baseline + Σ contributions | чистая модель (стартовый капитал + события), UX «начать с nonzero», self-healing |
| Пакет для формул | `mathcore/goals/` (новый) | принцип проекта «детерминированные расчёты → mathcore»; симметрия с tvm/credit |

---

## 2. Архитектура и структура пакетов

Полная симметрия с эталонным пакетом `budget` (storage → service с Repo
interface → http handler → wiring в router/main).

```
internal/
├── mathcore/goals/              ← НОВОЕ
│   ├── doc.go                   package doc + sentinel errors
│   ├── sinkingfund.go           SolveTerm, SolveContribution, SolveFutureValue, InflateTarget
│   └── sinkingfund_test.go      golden-тесты + edge-cases
├── domain/
│   └── goal.go                  ← НОВОЕ: Goal, GoalContribution, GoalStatus, Validate
├── storage/
│   ├── goals.go                 ← НОВОЕ: CRUD goals + CRUD/Sum contributions
│   └── goals_test.go            ← НОВОЕ: sqlmock
├── service/goals/
│   ├── goals.go                 ← НОВОЕ: Service + Repo interface
│   └── goals_test.go            ← НОВОЕ: fakeRepo
└── transport/http/
    ├── goals.go                 ← НОВОЕ: handler + Register
    └── goals_test.go            ← НОВОЕ: httptest + fakeRepo

migrations/0003_goals_contributions.sql  ← НОВОЕ
```

### Точки интеграции с существующим кодом (минимальные правки)

- `storage/dashboard.go::GoalProgresses` — прибавить
  `COALESCE(SUM(contributions.amount), 0)` к `current_amount` (одна строка SQL)
  для гибридной модели.
- `transport/http/router.go::Deps` — поле `Goals *goals.Service`, монтаж в
  authenticated group (nil-safe, как budget).
- `cmd/server/main.go` — `goals.NewService(pool)` + запись в Deps.

### Что НЕ трогаем

- таблицу `goals` (уже в 0001 со всеми нужными колонками)
- `mathcore/{credit,tvm,investment,daycount,tax}` (новый независимый подпакет)
- другие фичи 1–4

---

## 3. Модель данных

### 3.1 Миграция 0003_goals_contributions.sql

Таблица `goals` уже определена в миграции 0001 со всеми нужными колонками
(`target_amount`, `current_amount`, `monthly_contribution`, `target_date`,
`expected_yield`). Добавляется только журнал пополнений:

```sql
CREATE TABLE goal_contributions (
    id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id          BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    goal_id          BIGINT NOT NULL REFERENCES goals (id) ON DELETE CASCADE,
    contribution_id  TEXT NOT NULL,                       -- client-generated (идемпотентность)
    amount           NUMERIC(28, 2) NOT NULL CHECK (amount > 0 AND amount = ROUND(amount, 2)),
    contribution_date DATE NOT NULL,
    comment          TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, goal_id, contribution_id)
);

CREATE INDEX goal_contributions_goal_idx ON goal_contributions (goal_id);
CREATE TRIGGER goal_contributions_touch BEFORE UPDATE ON goal_contributions
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
```

Конвенции соблюдены: `BIGINT IDENTITY`, `NUMERIC(28,2)`, мягкое удаление через
`goals.deleted_at` (каскад), `touch_updated_at`.

### 3.2 Эффективный текущий остаток (гибридная модель)

```
effective_current = goals.current_amount + Σ goal_contributions.amount
```

- `goals.current_amount` — baseline («стартовый капитал», PATCH'ится напрямую,
  задаётся при CREATE; UX «начать с 200 000₽»).
- `goal_contributions` — события внеплановых пополнений (insert/delete).
- `Projection` / `Simulate` всегда читают `effective_current`, не `current_amount`.
- INSERT/DELETE contribution автоматически меняет projection (self-healing по
  аналогии с `accounts.balance` через полный recompute).

### 3.3 domain/goal.go

```go
type Goal struct {
    ID                 int64
    UserID             int64
    Name               string
    TargetAmount       domain.Money
    CurrentAmount      domain.Money  // baseline
    MonthlyContribution *domain.Money // nil = регулярного взноса нет
    TargetDate         *time.Time    // nil = без дедлайна
    ExpectedYield      decimal.Decimal
    CreatedAt, UpdatedAt time.Time
}

type GoalContribution struct {
    ID, UserID, GoalID  int64
    ContributionID      string
    Amount              domain.Money
    ContributionDate    time.Time
    Comment             string
    CreatedAt           time.Time
}

type GoalStatus string
const (
    StatusOnTrack    GoalStatus = "on_track"     // хватает взноса/срока
    StatusAtRisk     GoalStatus = "at_risk"      // на грани (взноса едва хватает)
    StatusBehind     GoalStatus = "behind"       // не успеем накопить (ErrUnreachable или n>deadline)
    StatusAchieved   GoalStatus = "achieved"     // effective >= target
    StatusNoDeadline GoalStatus = "no_deadline"  // target_date не задан
)
```

---

## 4. Математика: mathcore/goals/

Strictly decimal (`decimal.Ln` / `Pow`), без float64, без BrentQ. Число
documented float64-bridges остаётся равным 2 (PSK + XIRR).

### 4.1 Формулы (Копнова Г.П. Гл. 3.3.3 «Фонд возмещения»)

Обозначения:
- `P` — начальный (текущий) капитал
- `S` — целевая сумма
- `A` — периодический (месячный) взнос
- `i` — ставка за период (`expected_yield / 12`)
- `n` — число периодов (мес)

| Функция | Формула | Что решаем |
|---|---|---|
| `SolveFutureValue(P, A, i, n)` | `S = P·(1+i)^n + A·((1+i)^n − 1)/i` | будущая сумма S |
| `SolveContribution(P, S, i, n)` | `A = (S − P·(1+i)^n) · i / ((1+i)^n − 1)` | требуемый взнос A |
| `SolveTerm(P, S, A, i)` | `n = ln((S·i + A)/(A + P·i)) / ln(1+i)` | срок n (мес) |
| `InflateTarget(S, π, n)` | `S_инфл = S · (1+π)^(n/12)` | цель, скорректированная на инфляцию |

### 4.2 Edge-cases (покрываются тестами)

- `i = 0` → fallback: `SolveFutureValue = P + A·n`, `SolveContribution = (S−P)/n`,
  `SolveTerm = error` (нужна ставка).
- `S ≤ P` (effective current ≥ target) → `achieved` (n=0, A=0).
- `A > 0`, но `A ≤ P·i` (взнос не покрывает рост процентов) → целевая сумма
  растёт быстрее накоплений → `ErrUnreachable`.
- `π = 0` → `InflateTarget` возвращает S без изменений.
- `π = -1` → `ErrDeflation100Percent` (переиспользуем семантику из tvm).
- `n = 0` → `SolveFutureValue = P`.

### 4.3 Golden-эталоны (для тестов)

Подбираются по аналитическим примерам Копнова и пересчитываются вручную
(принцип проекта: doc↔код не должен расходиться). Конкретные значения
фиксируются в `sinkingfund_test.go` на этапе реализации.

### 4.4 Применение в Projection

1. `effective = current_amount + Σ contributions`
2. `progress = effective / target` (0..1+)
3. если `target_date` задан:
   - `monthsLeft = max(0, месяцев до target_date)`
   - `S_corr = InflateTarget(target, π, monthsLeft)` при `π > 0`
   - `required = SolveContribution(effective, S_corr, i, monthsLeft)`
   - статус: `achieved` если `effective ≥ S_corr`; иначе `on_track` /
     `at_risk` / `behind` по соотношению required vs заданный monthly_contribution
4. если `target_date` НЕ задан, но есть `monthly_contribution`:
   - `n = SolveTerm(effective, target, monthly_contribution, i)` →
     `estimated_date = now + n месяцев`; статус `on_track`/`behind`
5. иначе `no_deadline`

---

## 5. REST API

Все эндпоинты — за AuthMiddleware, RUB-only (по scope-lock проекта), money —
строкой в JSON (без float64), как везде в проекте.

| Метод | Путь | Назначение |
|---|---|---|
| POST   | `/api/v1/goals` | создать цель |
| GET    | `/api/v1/goals` | список целей |
| GET    | `/api/v1/goals/{id}` | одна цель |
| PATCH  | `/api/v1/goals/{id}` | обновить (target/current baseline/contribution/yield/date) |
| DELETE | `/api/v1/goals/{id}` | soft-delete |
| GET    | `/api/v1/goals/{id}/projection` | прогноз + статус (аналог `/budgets/{id}/status`) |
| POST   | `/api/v1/goals/{id}/contributions` | внеплановое пополнение (идемпотентно) |
| GET    | `/api/v1/goals/{id}/contributions` | журнал пополнений |
| DELETE | `/api/v1/goals/{id}/contributions/{cid}` | удалить пополнение |
| POST   | `/api/v1/goals/{id}/simulate` | what-if по сохранённой цели |
| POST   | `/api/v1/calc/goal` | stateless what-if без сохранения |

### Handler-стиль

Полная симметрия с `transport/http/budgets.go`: `decodeJSON`/`writeJSON`/
`writeError`/`writeServiceError`, `MustUserID(ctx)`, `applog.Info` для
аудита (с `user_hash`, без PII), `chi.URLParam` для `{id}`/`{cid}`,
`strconv.ParseInt` для id, маппинг sentinel-ошибок → 400/404/409/500.

### Идемпотентность пополнений (стиль ф.1)

```
repo.CreateContribution → INSERT … RETURNING …
  при ошибке → translatePgError(err,
      "goal_contributions_user_id_goal_id_contribution_id_key",
      ErrContributionExists)
service.AddContribution при ErrContributionExists
  → GetContributionByClientID(user_id, goal_id, contribution_id)
  → вернуть 200 + оригинал (как operations.GetOperationByCalcID)
```

Не используется `ON CONFLICT DO NOTHING` + отдельный lookup (расходится с
принятым в ф.1 шаблоном `INSERT…RETURNING` + sentinel).

### Тело запроса what-if

`POST /calc/goal` и `POST /goals/{id}/simulate` принимают один и тот же
`SimulateInput` (stateless-функция mathcore переиспользуется):

```json
{
  "current_amount": "200000.00",
  "target_amount":  "1000000.00",
  "monthly_contribution": "50000.00",
  "expected_yield": "0.08",
  "inflation": "0.06",
  "target_date": "2028-06-01"
}
```

`/simulate` по сохранённой цели комбинирует сохранённые поля цели с полями
тела запроса (тело приоритетнее — даёт «что если изменю взнос на …»).

---

## 6. Тестирование

Повторяет эталон budget (3 уровня тестов).

| Пакет | Инструмент | Покрытие |
|---|---|---|
| `mathcore/goals/` | table-driven + golden | все 4 функции, edge-cases (i=0, S≤P, unreachable, π=0/−1, n=0) |
| `storage/goals_test.go` | sqlmock | CreateGoal success/duplicate, Get/List/Update/Delete, contributions CRUD, Sum, ErrContributionExists |
| `service/goals_test.go` | fakeRepo (in-memory Repo) | Create validation, идемпотентность contribution, projection (on_track/at_risk/behind/achieved/no_deadline), simulate delta |
| `transport/http/goals_test.go` | httptest + fakeRepo + chi router | Create success/invalid, Projection, AddContribution идемпотентность (повтор = тот же ответ), Simulate, unauthorized, `/calc/goal` stateless |

### Definition of Done

```bash
cd C:/Users/user/ZCodeProject/FinHelper/backend
"C:/Program Files/Go/bin/go.exe" build ./...
"C:/Program Files/Go/bin/go.exe" vet ./...
"C:/Program Files/Go/bin/go.exe" test ./...   # все пакеты зелёные
```

Smoke без БД (как в PROGRESS): `/healthz` → 200, `/api/v1/goals` → 404 (не
смонтирован без pool — graceful).

### Ключевые эталоны (проверяются в тестах)

- Идемпотентность: повторный POST contributions с тем же `contribution_id` →
  200 + идентичный оригинал, без дубля.
- Effective current: goal с `current_amount=200000` + 2 contributions по 50000
  → effective = 300000 в projection.
- Projection achieved: `effective ≥ target` → status `achieved`, n=0, A=0.
- Simulate delta: изменение `monthly_contribution` в теле → меняет `n`/`A` в
  ответе, не сохраняя.

---

## 7. Риски и компромиссы

| Риск | Mitigation |
|---|---|
| Аналитика не решает «взнос при сроке И сумме И инфляции одновременно» | `SolveContribution` после `InflateTarget` покрывает основной случай; численный solver отложен (YAGNI для прототипа) |
| `decimal.Ln` точность на длинных сроках (>10 лет) | как и в `tvm.CompoundInterest`, при необходимости повысим `decimal.DivisionPrecision`; отложено до real-world кейсов |
| Дублирование bracket-search-паттерна — не возникает | аналитика, не solver; documented float64-bridges остаются = 2 |
| E2E через реальный Postgres | отложен (нет docker в окружении) — как во всём проекте; SQL сверен со схемой вручную + sqlmock |
| Гибрид `current_amount` + Σ contributions может дать drift при ошибках | self-healing через recompute в `Projection` (как `accounts.balance`) |

---

## 8. Готовность к будущему

- Stateless `Simulate`-функция mathcore закладывает фундамент для фичи 12
  «Симулятор Что если» (Этап 4 продолжение).
- Паттерн what-if через `POST /calc/goal` — шаблон для фич 6–9 (calc-эндпоинты).
- При появлении фичи 6 (вклады с пополнением) — оценить вынос формул в общий
  `mathcore/sinkingfund/` (PROGRESS фиксирует принцип «вынести при 3-м
  потребителе»).
