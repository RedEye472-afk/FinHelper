### Plan

# 📋 ПЛАН РЕАЛИЗАЦИИ MVP «ФинПомощник»

## 🎯 Зафиксированные решения (из ответов)

|Параметр|Решение|
|---|---|
|Стек|**Go 1.22+ бэкенд** + **React 18 + TypeScript** фронт|
|БД|**PostgreSQL 16**|
|MVP scope|**Фичи 1-9** (Level 1 + Level 2 BUSINESS_LOGIC)|
|Банки|❌ Только ручной ввод + CSV/Excel импорт|
|AI|❌ Без LLM, **fallback-шаблоны** из AI_GUARDRAILS.md|
|Валюты|**Только RUB**|
|Авторизация|**JWT**, без consent/audit (отложено в v1.0)|

---

## ⚠️ ВАЖНО: Перед кодом — корректировка документации

Прожарка выявила критические мат. ошибки в `MATH_FORMULAS.md`. Если их не исправить — golden-тесты будут с заведомо неверными эталонами. **Этап 0 обязательный**:

1. **ПСК** (![](./material-icons/markdown.svg)MATH_FORMULAS.md:208-231): переписать как численное решение уравнения `Σ CF_k / (1+i)^((d_k−d_0)/365) = 0` методом Ньютона (Указание ЦБ 5832-У). Пример 18.48% — пересчитать.
2. **Налог на вклады для 2026** (![](./material-icons/markdown.svg)MATH_FORMULAS.md:594-600): формула «1 млн × макс. ставка за год» устарела с 2026. Использовать ставку ЦБ на 1 января года расчёта (поправки декабря 2023).
3. **Валидатор чисел AI** (![](./material-icons/markdown.svg)AI_GUARDRAILS.md:135-148): починить сравнение (учесть нормализацию 0.185 → «18.5%»). Актуально для v1.0, в MVP использем fallback.

---

## 🏗️ Архитектура

### Технологии

- **Backend**: Go 1.22+, `chi` (router), `sqlc` (type-safe SQL генерация), `shopspring/decimal` (Decimal для денег), `golang-jwt/jwt/v5`, `golang-migrate`
- **Frontend**: React 18 + TypeScript + Vite, TanStack Query, TailwindCSS, `react-hook-form` + `zod`
- **Математика**: `shopspring/decimal` для денег; **исключение — XIRR**: bridge на `float64` только внутри numerical solver (BrentQ), результат сразу обратно в Decimal
- **Инфра**: Docker Compose (postgres + backend + frontend), на прод — Selectel/Yandex Cloud (соответствие 152-ФЗ)

### Структура проекта

![](./material-icons/document.svg)text

```
FinHelper/├── backend/│   ├── cmd/server/main.go│   ├── internal/│   │   ├── domain/         # типы домена (Money, Decimal, Transaction)│   │   ├── mathcore/       # детерминированные расчёты│   │   │   ├── tvm/        # TVM: simple, compound, fisher│   │   │   ├── credit/     # annuity, differentiated, PSK, early│   │   │   ├── investment/ # NPV, XIRR, MIRR, DPP, PI│   │   │   ├── daycount/   # ACT/365, 30/360, ACT/ACT│   │   │   └── tax/        # NPD, USN, NDFL, deposit (versioned)│   │   ├── storage/        # sqlc-generated код, migrations│   │   ├── service/        # бизнес-логика фич│   │   ├── transport/http/ # REST handlers, middleware│   │   └── ai/             # fallback-шаблон (stub для LLM в v1.0)│   ├── migrations/         # SQL миграции golang-migrate│   ├── configs/tax_rules_2024.yaml  # версионирование налогов│   ├── golden/             # golden-тесты (JSON из MATH_FORMULAS.md §6)│   └── go.mod├── frontend/│   ├── src/│   │   ├── api/            # REST клиент│   │   ├── components/     # переиспользуемые UI│   │   ├── features/       # фичи: operations, dashboard, credit, deposit...│   │   └── lib/decimal.ts  # обёртка над decimal.js для фронта│   └── package.json├── docker-compose.yml└── docs/                   # исправленные .md
```

### REST API (OpenAPI 3.0)

`POST /api/v1/auth/{register,login,refresh}` · `POST /api/v1/operations` · `GET /api/v1/dashboard?period=` · `POST /api/v1/calc/{deposit,credit,annuity,affordability,mortgage-vs-rent}` · `POST /api/v1/import/csv`

---

## 📅 ЭТАПЫ (12 недель ≈ 3 месяца, по roadmap из CLAUDE.md)

### Этап 0 — Корректировка документации (2-3 дня)

**Что:** починить мат. ошибки в `docs/`, обновить эталоны golden-тестов. **Файлы:** `FinHelper/MATH_FORMULAS.md`, `FinHelper/AI_GUARDRAILS.md`. **DoD:** все числовые примеры пересчитаны и сверены с эталонами (Excel/ЦБ калькулятор). Никаких противоречий формулы↔пример.

### Этап 1 — Фундамент backend (неделя 1-2)

**Что:** scaffold Go проекта, PostgreSQL схема, JWT auth, Decimal utilities, DayCount engine. **Файлы:** `go.mod`, `cmd/server/main.go`, `internal/domain/`, `internal/storage/`, `migrations/0001_init.sql`, `internal/transport/http/auth.go`, `internal/mathcore/daycount/`. **Зависимости:** chi, sqlc, shopspring/decimal, golang-jwt, golang-migrate, postgresql. **Тесты:** unit на DayCount (ACT/365, 30/360 edge-cases), integration на auth flow. **DoD:** пользователь регистрируется/логинится через REST, получает JWT, тесты зелёные.

### Этап 2 — Математическое ядро (неделя 3-4) — **САМЫЙ РИСКОВАННЫЙ**

**Что:** реализовать все формулы из `MATH_FORMULAS.md` §1-5 с golden-тестами. **Файлы:** `internal/mathcore/{tvm,credit,investment,daycount,tax}/`, `golden/*.json`, `configs/tax_rules_*.yaml`. **Зависимости:** нет новых (всё на shopspring/decimal + стандартная library). **Тесты:** golden-тесты с точностью ±1e-10, table-driven на edge-cases, coverage ≥ 90% по mathcore. **DoD:** все 4 golden-кейса (§6) зелёные + 50+ edge-case тестов. CI блокирует merge при падении.

### Этап 3 — Фичи 1-4: Level 1 (неделя 5-6)

**Что:** ручной ввод операций, авто-категоризация (rules-based, без ML), дашборд, бюджеты. **Файлы:** `internal/service/{operations,categorization,dashboard,budget}/`, REST handlers, React-фичи `features/{operations,dashboard,budgets}`. **Зависимости (фронт):** Vite, TanStack Query, Tailwind, decimal.js. **Тесты:** service-layer unit тесты + React Testing Library. **DoD:** пользователь вводит операцию → видит её на дашборде → создаёт бюджет → получает alert при перерасходе.

### Этап 4 — Фичи 5-9: Level 2 калькуляторы (неделя 7-9)

**Что:** трекер целей, калькулятор вкладов, кредитный калькулятор (с ПСК + досрочка), анализ доступности, ипотека vs аренда. **Файлы:** `internal/service/{goals,deposit,credit,affordability,mortgage}/`, React-фичи. **Зависимости:** нет новых. **Тесты:** scenario-тесты на каждый калькулятор (вход из BUSINESS_LOGIC → ожидаемый выход). **DoD:** все 5 калькуляторов работают, кнопка «Показать формулу» раскрывает математику (из docstrings).

### Этап 5 — AI stub + объяснения (неделя 10)

**Что:** AI service возвращает fallback-шаблон из ![](./material-icons/markdown.svg)AI_GUARDRAILS.md:197-216, индикатор «🤖 Объяснение от ИИ», кнопка «Показать формулу», обязательный дисклеймер. **Файлы:** `internal/ai/fallback.go`, React `components/AiExplanation.tsx`. **DoD:** каждый расчёт сопровождается fallback-объяснением + дисклеймером в 100% ответов.

### Этап 6 — Импорт и финализация (неделя 11-12)

**Что:** CSV/Excel импорт (`excelize`), OpenAPI документация, E2E тесты, README, docker-compose для разработки. **Файлы:** `internal/service/import/`, `api/openapi.yaml`, `e2e/`, `README.md`. **DoD:** `docker-compose up` поднимает полностью рабочее окружение. Импорт CSV создаёт операции. E2E покрывает critical path (register → add operation → see dashboard → calc credit).

---

## 🛡️ Риски и mitigation

|Риск|Вероятность|Влияние|Mitigation|
|---|---|---|---|
|**Неверная формула ПСК** (этап 2)|Высокая|Критическое|Реализовать через numerical solver сразу; сверить с эталоном ЦБ/Excel СТАВКА; golden-тест|
|**Decimal vs float для XIRR**|Средняя|Среднее|Bridge: solver на float64, ввод/вывод/округление через Decimal; documented exception в коде|
|**Сложность фичи 9 (ипотека vs аренда)**|Средняя|Среднее|Реализовать последней (этап 4), разбить на подзадачи; упростить MVP до базовых сценариев|
|**Превышение сроков**|Высокое|Среднее|Жёсткий scope-lock: ML-категоризация → rules-only, multi-currency → RUB only, AI → stub|
|**Ошибки в tax_rules конфигах**|Средняя|Высокое|Versioned YAML + unit-тесты на каждый год; эталон — калькулятор ФНС|

---

## 🎯 ПЕРВЫЕ 3 ЗАДАЧИ (для немедленного старта после Этапа 0)

### Задача 1: Backend scaffold + PostgreSQL схема

**Вход:** решения по стеку из этого плана. **Выход:** запускаемый Go-проект с подключением к PostgreSQL и применёнными миграциями. **Требования:**

- Go module `finhelper`, структура каталогов по плану
- `docker-compose.yml` с postgres:16
- Миграция `0001_init.sql`: таблицы `users`, `accounts`, `categories`, `operations`, `goals`, `budgets` (с PK, FK, индексы)
- Все money-колонки как `NUMERIC(28, 2)` (не float!)
- `sqlc.yaml` конфиг, генерация первого запроса `SELECT 1` для проверки
- `.env.example` с `DATABASE_URL`, `JWT_SECRET` **Тесты:** integration-тест `TestDBConnection`, `TestMigrationsApply`. **Verification:** `docker-compose up -d postgres && go run cmd/server/main.go` → в логе «DB connected».

### Задача 2: JWT auth (register/login/refresh)

**Вход:** таблица `users` из задачи 1. **Выход:** 3 REST endpoint + middleware. **Требования:**

- `POST /api/v1/auth/register` (email, password) → валидация, bcrypt, JWT access (15 мин) + refresh (30 дней)
- `POST /api/v1/auth/login` → проверка пароля, выдача токенов
- `POST /api/v1/auth/refresh` → ротация refresh
- Middleware `AuthMiddleware` — проверка JWT, прокидывание `user_id` в context
- Password policy: ≥ 8 символов
- В логах только `user_hash` (SHA-256 от user_id + salt), НЕ email — ![](./material-icons/markdown.svg)PRIVACY_RULES.md:32 **Тесты:** unit на token generation, integration на полный auth flow, edge-cases (неверный пароль, истёкший токен). **Verification:** `curl POST /register` → 201 + tokens; `curl GET /me` без токена → 401.

### Задача 3: DayCount engine + TVM базовые формулы

**Вход:** исправленные формулы из Этапа 0. **Выход:** `internal/mathcore/daycount/` + `internal/mathcore/tvm/` с тестами. **Требования:**

- `DayCountEngine.YearFraction(start, end, convention)` → Decimal; конвенции ACT/365, 30/360 ISDA, ACT/ACT ISMA
- `tvm.SimpleInterest(P, i, t)` → Decimal, edge-cases t=0, i=0, t<0 (error)
- `tvm.CompoundInterest(P, i, m, t)` → Decimal, edge-cases m=1, t=0
- `tvm.EffectiveRate(i_nom, m)` → Decimal
- `tvm.Fisher(r_nom, π)` → Decimal, edge-case π=-1 (panic/error)
- Все функции — на `shopspring/decimal.Decimal`, **запрет `float64`** в money-расчётах (lint rule)
- Docstring с источником (Копнова §) и формулой в каждой функции **Тесты:** golden-тесты `golden/tvm_001.json`...`golden/tvm_010.json` с эталонами из MATH_FORMULAS.md §1 и §6.1-6.2; edge-case таблицы. **Verification:** `go test ./internal/mathcore/... -v` → все тесты зелёные; `go vet` и `golangci-lint` без замечаний; CI green.

---

## ✅ Чек-лист перед стартом кодирования

После утверждения плана я приступлю строго по этапам:

1. Сначала **Этап 0** — правки документации (с твоего явного одобрения правок в .md).
2. Параллельно могу начать **Задачу 1** (scaffold), она не зависит от правок математики.
3. На каждом этапе — показываю прогресс и тесты, прошу подтвердить переход к следующему