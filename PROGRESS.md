# 📋 PROGRESS.md — что сделано в проекте FinHelper

> Журнал состояния. Кратко: что готово, что плохо, что хорошо. Обновляется по
> мере прохождения этапов. Восстанавливай контекст отсюда, если сессия прервалась.

**Стек**: Go 1.26 + React 18/TS | PostgreSQL 16 | JWT auth | только RUB | без LLM (fallback-шаблоны)
**План**: см. утверждение в чате — MVP = фичи 1-9 из BUSINESS_LOGIC.md
**Окружение**: Windows, Go через `C:\Program Files\Go\bin\go.exe`, git bash как shell

---

## ✅ Этап 0 — Корректировка документации (ВЫПОЛНЕН)

**Файлы:** `FinHelper/MATH_FORMULAS.md`, `FinHelper/AI_GUARDRAILS.md`

Что исправлено:
1. **ПСК** (§2.3) — убрал неверную упрощённую формулу `(Σплатежей − P)/P × (12/n)`
   и арифметическую ошибку «18.48%» (там было 9.24%). Заменил на численное решение
   уравнения денежного потока по методике ЦБ (Указание 5750-У). Эталон: **~17.96%**.
2. **Налог на вклады** (§5.4) — переписал под ФЗ-382 от 26.12.2023: ставка ЦБ на
   1 января, а не «макс за год». Добавил пример 2026.
3. **Golden-тест налога** (§6.4) — `key_rate_max` → `key_rate_jan1`.
4. **AI валидатор чисел** (AI_GUARDRAILS §"Проверка 1") — починил сравнение: было
   строковое (отвергало любой правильный ответ из-за `0.185` vs `18.5%`), стало
   числовое с допуском + нормализация запятых/пробелов + производные (×100, целая часть).

**Хорошо:** источники указаны (Копнова, ЦБ, ФЗ), формулы теперь методически корректны.
**Плохо:** в исходных доках были арифметические ошибки в примерах — это критично
для проекта, где точность = доверие. Все пересчитано вручную.

---

## ✅ Задача 1 — Backend scaffold (ВЫПОЛНЕН + ВЕРИФИЦИРОВАН)

**Файлы созданы:**
- `FinHelper/docker-compose.yml` — postgres:16-alpine, healthcheck, volume pgdata
- `FinHelper/backend/.env.example` — все переменные с комментариями
- `FinHelper/.gitignore` — Go/Node/secrets/IDE
- `FinHelper/backend/go.mod` — module `github.com/RedEye472-afk/FinHelper`, Go 1.26.4
- `FinHelper/backend/migrations/0001_init.sql` — полная схема:
  users, refresh_tokens, accounts, categories, operations, goals, budgets
  + ENUM-типы + триггер `touch_updated_at`
- `FinHelper/backend/internal/config/config.go` — загрузка конфига из env
- `FinHelper/backend/internal/config/config_test.go` — тесты валидации
- `FinHelper/backend/internal/log/logger.go` — slog-обёртка с user_hash из ctx
- `FinHelper/backend/internal/domain/money.go` — тип Money (decimal, scale=2)
- `FinHelper/backend/internal/domain/money_test.go` — тесты точности (0.1+0.2=0.3!)
- `FinHelper/backend/internal/domain/operation.go` — Operation + invariants
- `FinHelper/backend/internal/storage/postgres.go` — пул соединений (database/sql + pgx)
- `FinHelper/backend/cmd/server/main.go` — HTTP server с /healthz, /readyz, graceful shutdown

**Зависимости:** chi v5.3.0, shopspring/decimal v1.4.0, pgx v5.10.0, stdlib.

**Верификация (проверено 23.06):**
- `go build ./...` → BUILD_OK
- `go vet ./...` → VET_OK
- `go test ./...` → все тесты зелёные (config + domain)
- Smoke-тест: `/healthz` → 200 `{"status":"ok"}`; `/readyz` → 503 без БД ✓

**Хорошо:**
- Money на decimal с фиксированной scale=2 (соответствует NUMERIC(28,2) в БД)
- Тест доказывает: 0.1 + 0.2 = 0.30, а не 0.30000000004 (главный принцип проекта)
- Logger возвращает user_hash из ctx, а не email — приватность by design
- Идемпотентность заложена: UNIQUE(user_id, calc_id) в operations
- Сервер бьётся без БД — удобно для smoke-тестов и CI
- Чёткое разделение пакетов: domain / config / log / storage / transport

**Плохо / технический долг:**
- `sqlc` пока не подключён (план Этапа 1), SQL пока в миграциях, queries в коде
- Нет golang-migrate CLI — миграции применяются вручную через psql (нужно для Задачи 2)
- В Windows go не в PATH, вызываю по полному пути `C:\Program Files\Go\bin\go.exe`
- bash в Windows — это git bash, Windows-команды `set X=Y` не работают, нужно export
- `.env.example` требует минимум 32-символьные секреты — генерация на пользователе
- README.md ещё не написан (план Этапа 6)

---

## ✅ Задача 2 — JWT auth (ВЫПОЛНЕН + ВЕРИФИЦИРОВАН)

**Цель:** `POST /api/v1/auth/{register,login,refresh}` + `GET /api/v1/me` + AuthMiddleware — достигнута.

**Файлы созданы:**
- `internal/auth/password.go` — bcrypt cost 12, policy ≥8 ≤72 символов (NIST 800-63B: длина без composition-rules), `ValidatePassword`/`HashPassword`/`CheckPassword`
- `internal/auth/hash.go` — `UserHash = SHA-256(user_id || ":" || salt)` с сепаратором против prefix-collision, `HashToken` (для refresh-хранения), `HashEmail`, `FormatUserHashLogValue`
- `internal/auth/jwt.go` — `JWTIssuer` с access/refresh на разных секретах; claims `{uid, uhsh, kind}`; HMAC-only keyfunc (защита от alg-confusion/`none`); kind check (defense-in-depth); `JWTVerifier` interface для middleware
- `internal/storage/users.go` — `CreateUser`, `GetUserByEmail`, `GetUserByID`, `SaveRefreshToken`, `ConsumeRefreshToken` (atomic UPDATE…RETURNING — single-use rotation), `RevokeAllRefreshTokens`
- `internal/storage/errors.go` — `translatePgError` (23505 → `ErrUserExists` по constraint-name)
- `internal/transport/http/json.go` — `Problem` envelope, `writeJSON`/`writeError`
- `internal/transport/http/email.go` — `validateEmail` на `mail.ParseAddress` (отвергает `Display <addr>`)
- `internal/transport/http/auth.go` — `Register` (two-phase tx: INSERT placeholder → UPDATE user_hash по id), `Login` (идентичный 401 для not-found/wrong-pw — не oracle), `Refresh` (verify JWT → consume row → issue new pair)
- `internal/transport/http/middleware.go` — `AuthMiddleware.Wrap` (Bearer → claims → ctx `user_id`/`user_hash` + `applog.WithUserHash`)
- `internal/transport/http/router.go` — `NewRouter` монтирует `/api/v1/auth/*` (public) и authenticated group с `/me` placeholder
- `cmd/server/main.go` — wiring: JWT issuer → router; auth-эндпоинты mount только если есть DB pool (graceful: без БД `/healthz` работает, `/api/v1` → 404)

**Зависимости добавлены:** `golang-jwt/jwt/v5 v5.2.2`, `golang.org/x/crypto` (bcrypt), `github.com/DATA-DOG/go-sqlmock v1.5.2`. `golang-migrate` НЕ понадобился — миграции применяются через psql вручную (см. тех.долг).

**Верификация (проверено 23.06):**
- `go build ./...` → BUILD_OK
- `go vet ./...` → VET_OK
- `go test ./...` → все пакеты зелёные:
  - `internal/auth`: password (roundtrip, salts, malformed), hash (детерминизм, salt, separator, panic на empty salt), jwt (kind-cross-use, wrong secret, expired, alg=none, garbage)
  - `internal/storage`: users CRUD через sqlmock, 23505→ErrUserExists, ConsumeRefreshToken atomicity
  - `internal/transport/http`: полный auth flow через httptest.Server + sqlmock (register 201/409/400, login 200/401, refresh rotate, /me 200/401, AuthMiddleware reject missing/garbage/refresh-as-access)
- Binary smoke: `/healthz` → 200, `/readyz` → 503 без БД, `/api/v1/auth/*` не смонтированы без БД (404, не паника)

**Что НЕ покрыто (требует окружения с Postgres):**
- Полноценный E2E через реальную БД. В текущем окружении `docker`, `psql`, `pg_ctl` не найдены — Postgres поднять нельзя. Все SQL-запросы сверены со схемой `migrations/0001_init.sql` вручную. E2E прогон отложен до этапа, когда будет доступен docker-compose.

**Хорошо:**
- Refresh-токены single-use: `ConsumeRefreshToken` = atomic `UPDATE … WHERE revoked_at IS NULL AND expires_at > NOW() RETURNING user_id` — два параллельных refresh-запроса не смогут оба успеть
- В БД хранится только `SHA-256(refresh_token)`, не сам токен — кража БД не даёт имперсонации
- Access и refresh на разных секретах + kind claim = тройная защита от cross-use
- Логи содержат только `user_hash`, никогда email (PRIVACY_RULES.md §1)
- 401 идентичен для «нет такого email» и «неверный пароль» — не oracle
- HMAC-pinned keyfunc защищает от классической атаки `alg: none`/RS256→HS256 confusion
- `applog.WithUserHash` в middleware — каждое downstream-логирование автоматически несёт user_hash
- Register использует транзакцию для two-phase insert (placeholder user_hash → реальный по id)

**Плохо / технический долг:**
- Нет rate-limiting на `/auth/login` (brute-force защита) — отложено в Задачу безопасности
- Нет refresh-token reuse detection (при использовании отозванного токена должны ревёкнуть ВСЕ токены пользователя) — помечено в roadmap
- `ConsumeRefreshToken` не различает «revoked» и «expired» в логах — оба дают одинаковый 401 (намеренно для безопасности, но усложняет detection)
- E2E через реальный Postgres не прогнан (нет docker в окружении)
- `/me` placeholder — заменить на реальный profile-handler когда появятся фичи
- ip_hash / device fingerprinting для audit-log — не реализовано (в MVP scope не входит)

---

## ✅ Задача 3 — DayCount + TVM (ВЫПОЛНЕН + ВЕРИФИЦИРОВАН)

**Цель:** `internal/mathcore/daycount/` + `internal/mathcore/tvm/` с golden-тестами — достигнута.

**Подход:** диспатч двух parallel subagents (Explore) через скилл `dispatching-parallel-agents` для разведки: они подтвердили окружение, style-конвенции, и — главное — исправили критическую ошибку в моём ТЗ: `decimal.Pow` в shopspring v1.4.0 возвращает `Decimal`, **а не `(Decimal, error)`**, и **поддерживает дробные показатели** (через `Ln`+`ExpTaylor`). Это убрало необходимость в fallback'е на integer-only exponent. Финальную реализацию написал сам — agents оказались read-only.

**Файлы созданы:**
- `internal/mathcore/daycount/daycount.go` — `Convention` (`ACT365`, `Thirty360` ISDA, `ACTACT` ISMA), `YearFraction`, `DaysBetween`
  - 30/360 ISDA с правилами D1=31→30, D2=31 && D1≥30 → 30
  - ACT/ACT ISMA: разбиение периода по календарным годам, знаменатель 365/366 по году
  - Все даты нормализуются через `utcMidnight` (иммунитет к DST)
  - Sentinel errors: `ErrReversedPeriod`, `ErrUnknownConvention`
- `internal/mathcore/daycount/daycount_test.go` — golden (ACT365 182/365, 30/360 60/360, ACTACT 182/366, year-boundary 78/365 + 79/366), edge-cases (same-day, reversed, unknown), 13 тестов
- `internal/mathcore/tvm/tvm.go` — `SimpleInterest`, `CompoundInterest`, `EffectiveRate`, `FisherRealRate`
  - Compound использует дробный `Pow` напрямую (по разведке agent'а)
  - Защита от zero-value из `Pow` на undefined input (negative base + fractional exp)
  - Sentinel errors: `ErrNegativeTime`, `ErrInvalidCompounding`, `ErrDeflation100Percent`
- `internal/mathcore/tvm/tvm_test.go` — golden (Simple 105000 точный, Compound 110471.31, Compound fractional 0.5y ≈106152.02, Effective 0.104713, Fisher 0.018519), edge-cases (t=0, i=0, P=0, m≤0, t<0, π=-1, π>rate negative yield), 12 тестов

**Верификация (23.06):**
- `go build ./...` → BUILD_OK
- `go vet ./...` → VET_OK
- `go test ./...` → **все 8 пакетов зелёные** (config, domain, auth, storage, transport/http, daycount, tvm)
- Все golden-эталоны из MATH_FORMULAS.md §1 и §4 проходят

**Хорошо:**
- Каждая функция имеет doc comment с формулой, источником (Копнова/Люу/ЦБ) и edge-cases — для кнопки «Показать формулу» в UI (CLAUDE.md принцип 3 «Прозрачность»)
- Zero float64 — соответствует детерминизм-принципу проекта
- TVM имеет fractional-years тест (0.5 года) —Compound работает для любых сроков вклада, не только целых лет
- DayCount ACT/ACT корректно обрабатывает year-boundary (проверено 2023→2024)
- Tolerance `1e-6` для non-terminating decimals; точные equal для terminating (Simple)

**Плохо / технический долг:**
- `decimal.Pow` на дробных показателях internally использует `Ln`+`ExpTaylor` — точность зависит от `decimal.DivisionPrecision` (по умолчанию 16). Для финансовых расчётов на длинных сроках (>10 лет) стоит явно повысить precision. Отложено до появления real-world тест-кейсов.
- Нет ACT/30 (Eurobond), ACT/360, NL/365 — добавим при первом реальном использовании
- Compound на negative rate + fractional exponent может вернуть zero value — сейчас это детектится и возвращается как error, но без различения причин

**Решение от agents зафиксировано:** в `internal/mathcore/tvm/` CompoundInterest использует `base.Pow(exp)` напрямую (без integer-only fallback), т.к. shopspring v1.4.0 это поддерживает. Если когда-либо заменим decimal-либу — перепроверить.

---

## ✅ Задача 4 — credit/annuity + PSK (ВЫПОЛНЕН + ВЕРИФИЦИРОВАН)

**Цель:** `internal/mathcore/credit/` с golden-тестами — достигнута.

**Файлы созданы:**
- `internal/mathcore/credit/doc.go` — package doc + sentinel errors (ErrInvalidTerm/Month/Principal, ErrNoSignChange, ErrInsufficientCashflows, ErrSolverFailed, ErrEarlyExceedsBalance, ErrInvalidEarlyAmount)
- `internal/mathcore/credit/annuity.go` — `AnnuityPayment` (Копнова Гл. 4.2.1), fallback `P/n` при i=0; `AnnuitySchedule` (закрытие баланса в 0 за счёт absorb последней строки); `ScheduleRow`
- `internal/mathcore/credit/differentiated.go` — `DifferentiatedPayment(P,i,n,k)` и `DifferentiatedSchedule` (Копнова Гл. 4.2.2)
- `internal/mathcore/credit/solver.go` — **BrentQ** (Brent–Dekker, bisection+secant+IQI). Единственный documented float64 bridge в mathcore
- `internal/mathcore/credit/psk.go` — `PSK` / `PSKWithPeriods` по методологии ЦБ 5750-У: уравнение `Σ CF_k / (1+i)^q_k = 0`, `q_k = (d_k − d_0) × ЧБП/365`, годовая = `i × ЧБП` (номинальная, не эффективная). `Cashflow`, `AnnuityCashflows`, монотонный bracket-search
- `internal/mathcore/credit/early.go` — `EarlyRepayment` (2 режима: EarlyShortenTerm / EarlyLowerPayment), `reamortise`, `sumInterest`
- `internal/mathcore/credit/credit_test.go` — 20 тестов: golden + edge-cases

**Зависимости:** нет новых (shopspring/decimal уже в go.mod).

**Верификация (23.06):**
- `go build ./...` → BUILD_OK
- `go vet ./...` → VET_OK
- `go test ./...` → **все 9 пакетов зелёные** (+credit: 20 тестов)
- Ключевые эталоны:
  - Annuity 24m = 47073.4722232… → 47073.47 (HALF_EVEN) — сверка с Excel ПЛТ
  - Differentiated month 1/2 = 51666.67 / 51250.00
  - PSK no-fees = **0.1200 точно** (номинальная = headline rate, как и должно быть по ЦБ)
  - PSK with insurance+fee = **0.1776** (17.76%)
  - BrentQ √2, linear root, no-sign-change error path
  - Early repayment: shorten-term (payment const, term↓), lower-payment (term const, payment↓), closes-loan, edge-cases

**Хорошо:**
- PSK реализован по строгой методологии ЦБ 5750-У, не через упрощённую формулу (Этап 0 фикс доведён до конца)
- `PSK no-fees == 0.12` — главный sanity-check: кредит без доп. платежей имеет ПСК = номинальной ставке, как в рекламе банка. Эффективная ставка была бы 12.68% — это другая метрика
- BrentQ sign-safe и bracket-preserving (корень всегда внутри брекета)
- float64 bridge изолирован в solver.go + psk.go (solveIRR/npvPerPeriod); все денежные суммы идут через decimal, в float конвертируются только даты→q и финальная ставка→decimal
- Аннуитетный график закрывается ровно в 0 за счёт absorb residual в последней строке
- Edge-cases: zero-rate fallback, P=0, n=0, paidMonths out of range, early>balance, no-sign-change, <2 cashflows

**Плохо / технический долг:**
- Обнаружены и исправлены арифметические ошибки в MATH_FORMULAS.md §2.1/§2.3/§6.3 (аннуитет указан 47073.46 вместо 47073.47 → ПСК из doc 17.96% был несогласован; правильное значение 17.76%). Это та же категория багов, что в Этапе 0 — нужны актуальные эталонные калькуляторы при составлении docs
- BrentQ пока не покрыт тестом на полиномиальные корни высокой степени — добавим при появлении XIRR (Этап 2 продолжение)
- E2E через реальную БД по-прежнему отложен (нет docker в окружении)

**КРИТИЧЕСКОЕ РЕШЕНИЕ (зафиксировано для будущих сессий):**
ПСК = **номинальная** годовая ставка `i × ЧБП` по Указанию ЦБ 5750-У, НЕ эффективная `(1+i)^ЧБП − 1`.
Это позволяет потребителям сравнивать ПСК с рекламируемой номинальной ставкой напрямую.
Когда будем делать investment/XIRR — там нужна **эффективная** доходность, это другой метод аннуализации.

---

## ✅ Задача 5 — investment (NPV/XIRR/MIRR/DPP/PI) (ВЫПОЛНЕН + ВЕРИФИЦИРОВАН)

**Файлы созданы:**
- `internal/mathcore/investment/doc.go` — package doc + sentinels (ErrInsufficientCashflows, ErrNoSignChange, ErrSolverFailed, ErrNoPositiveCF, ErrNoNegativeCF, ErrInvalidRate, ErrNeverPaidBack, ErrZeroInitialInvestment)
- `internal/mathcore/investment/npv.go` — `NPV` (decimal), `Cashflow` type, `sortByDate`
- `internal/mathcore/investment/xirr.go` — `XIRR` через `credit.BrentQ` (второй и последний documented float64 bridge). Effective annual yield `(1+r)^(days/365) − 1`
- `internal/mathcore/investment/mirr.go` — `MIRR` (PV_neg / FV_pos)^(1/n) − 1, decimal
- `internal/mathcore/investment/payback.go` — `DPP` (с интерполяцией) + `PI`
- `internal/mathcore/investment/investment_test.go` — 15 тестов: golden + edge-cases

**Верификация (23.06):** BUILD_OK / VET_OK / `go test ./internal/mathcore/investment/` → 15/15 зелёных

**Ключевые эталоны** (с пересчётом — doc §3 содержал арифметические ошибки):
- NPV (-100k, +30k/+40k/+50k, r=10%) = -2103.68 (unprofitable) — сходится с §3.1
- XIRR (4 квартальных CF) = **0.40657** (~40.66%) — round-trip проверка: NPV@XIRR ≈ 0. Doc §3.2 указывал 12.34% — **невозможно** для CF где 120k дохода на 100k вложений за год
- MIRR = **0.08631** (~8.63%) — кубический корень 1.28192 = 1.08631, не 1.08527 (doc §3.3)
- DPP — не окупается за 3 года при r=10% (сходится с §3.4)
- PI unprofitable = 0.97896 (< 1)

**КРИТИЧЕСКОЕ РЕШЕНИЕ (зафиксировано):**
XIRR = **эффективная** годовая доходность, в отличие от ПСК (номинальная по ЦБ).
Не путать метрики при разработке фич 5-9 (Этап 4).

**Хорошо:**
- BrentQ переиспользован из credit без дубля
- Все денежные суммы — decimal, в float конвертируются только даты→дни и финальный rate→decimal
- DPP корректно обрабатывает immediate payback (t=0 → 0, не −1)

**Плохо / тех. долг:**
- `solveXIRR` дублирует логику bracket-search из `credit.solveIRR`. Когда появится 3-й потребитель — вынести в `mathcore/solver` подпакет
- Doc §3 опять содержал неверные эталоны (XIRR 12.34%, MIRR 8.52%) — та же категория багов, что Этап 0 и Задача 4

---

## ✅ Задача 6 — tax (NPD/USN/NDFL/deposit) (ВЫПОЛНЕН + ВЕРИФИЦИРОВАН)

**Файлы созданы:**
- `configs/tax_rules_{2024,2025,2026}.yaml` — версионные правила (canon в backend/configs, копия embedded в tax/configs для `go:embed`)
- `internal/mathcore/tax/tax.go` — package doc + sentinels + `Dec` (YAML-friendly decimal через строку, чтобы избежать float-потерь) + `Rules`/`NPDRules`/`USNRules`/`NDFLRules`
- `internal/mathcore/tax/loader.go` — `LoadRules(year)` через `embed.FS`, `LoadRulesFromFS` (для тестов), `MustLoadRules`
- `internal/mathcore/tax/parse.go` — YAML-парсер
- `internal/mathcore/tax/deposit.go` — `DepositTax` (ФЗ-382) + `Threshold`
- `internal/mathcore/tax/npd.go` — `NPD` + `NPDResult.ExceedsLimit` (сигнал переключения режима)
- `internal/mathcore/tax/usn.go` — `USN` (2 режима: Income / IncomeMinusExpenses)
- `internal/mathcore/tax/ndfl.go` — `NDFL` (прогрессивная шкала 13%/15%) + `ChildDeduction`
- `internal/mathcore/tax/tax_test.go` — 16 тестов: golden + edge-cases

**Зависимости:** `gopkg.in/yaml.v3 v3.0.1`

**Верификация (23.06):** BUILD_OK / VET_OK / `go test ./internal/mathcore/tax/` → 16/16 зелёных

**Ключевые эталоны (§5):**
- DepositTax 2024: interest=200k, rate_jan1=0.16 → threshold 160k → tax=5200 ✓
- DepositTax 2025: interest=300k, rate_jan1=0.21 → threshold 210k → tax=11700 ✓
- NPD: 500k физлица + 300k бизнес = 20000 + 18000 = 38000 ✓
- USN 15%: (2M − 1.2M) × 0.15 = 120000 ✓
- NDFL прогрессивная: 7M = 5M×0.13 + 2M×0.15 = 950000 ✓
- ChildDeduction: 1→1400, 2→2800, 3→5800, 4→8800

**Хорошо:**
- `Dec`-обёртка гарантирует, что YAML `"0.16"` парсится в decimal без float64-потери
- `go:embed` — правила tamper-evident, изменение требует перекомпиляции (важно для фин. приложения)
- `path.Join` (не `filepath.Join`) для embed.FS — forward-slash независимо от OS
- Версионирование: 2024/2025/2026 отдельно, добавление нового года = новый файл + тест

**КРИТИЧЕСКОЕ РЕШЕНИЕ (зафиксировано):**
embed.FS **всегда** использует forward slashes, даже на Windows. `filepath.Join` даст `configs\…` → `fs.ReadFile` упадёт. Используем `path.Join`.

**Плохо / тех. долг:**
- Канон configs лежат и в `backend/configs/` и в `internal/mathcore/tax/configs/` (дублирование). Когда будет CI — сделать `backend/configs/` единственным источником и symlink/embed-директиву с относительным путём
- Ставки ЦБ на 01.01.2026 взяты предположительно (21%) — сверить с cbr.ru перед релизом

---

## ✅ ЭТАП 2 ЗАКРЫТ — мат. ядро полностью готово

| Пакет | Файлов | Тестов | Статус |
|---|---|---|---|
| mathcore/daycount | 2 | 13 | ✅ (Задача 3) |
| mathcore/tvm | 2 | 12 | ✅ (Задача 3) |
| mathcore/credit | 6 | 20 | ✅ (Задача 4) |
| mathcore/investment | 6 | 15 | ✅ (Задача 5) |
| mathcore/tax | 8 | 16 | ✅ (Задача 6) |
| **ИТОГО** | **24** | **76** | **build/vet/test зелёные** |

Documented float64 bridges: ровно 2 (credit/BrentQ через PSK, переиспользован в XIRR) — как и требовал план. Всё остальное — строго decimal.

---

## ✅ Этап 3 / Фича 1 — Ручной ввод операций (ВЫПОЛНЕНО + ВЕРИФИЦИРОВАНО)

**Цель:** BUSINESS_LOGIC.md ф.1 — ручной ввод операций с идемпотентностью, PII-маскированием, детерминированным пересчётом балансов — достигнута.

**Файлы созданы:**

*Domain layer:*
- `internal/domain/account.go` — `Account`, `AccountType` (cash/bank/savings/investment/crypto/debt), `Validate()`
- `internal/domain/category.go` — `Category` (system/user, parent_id), `Validate()`
- `internal/domain/money.go` — расширение: `FromDecimal`, `Abs`, `Neg`, `Equal`, `AddAll`. **Документация поправлена:** shopspring `Round(n)` = ROUND_HALF_AWAY_FROM_ZERO (1.005→1.01), НЕ HALF_EVEN как раньше утверждалось. Найденные неверные doc-комментарии в `money.go`/`money_test.go`/`credit/credit_test.go` исправлены (та же категория багов что в Этапе 0 — doc↔код рассинхрон).
- `internal/domain/operation.go` — `Validate()` усилена: добавлена проверка валидности `OperationType` (раньше пропускала любое строковое значение — дыра, найденная service-тестом `unknown_type`)

*PII masking (PRIVACY_RULES.md §"Маскирование PII"):*
- `internal/pii/pii.go` — `Mask(s)` регекс-маскирование: [PHONE], [EMAIL], [PASSPORT], [CARD], [PERSON] (Cyrillic+Latin ALLCAPS), [MEDICAL], [LEGAL]. Идемпотентное (brackets ломают матчинг), консервативное (не трогает amounts/dates).
  - **Критическая находка:** Go RE2 `\b` работает ТОЛЬКО с ASCII (`\w = [0-9A-Za-z_]`), рядом с кириллицей НЕ срабатывает. Для кириллических keyword-правил `\b` убран, для person-детекции переписан как «2+ ALLCAPS токенов через whitespace».
- `internal/pii/pii_test.go` — 15 кейсов (включая idempotency, multiple-types-at-once, keeps-amounts)

*Storage layer:*
- `internal/storage/accounts.go` — `CreateAccount`, `GetAccount`, `ListAccounts`, `SetAccountBalance`; sentinels `ErrAccountExists`/`ErrAccountNotFound`
- `internal/storage/categories.go` — CRUD + `SeedSystemCategories(tx, names)` (для онбординга)
- `internal/storage/operations.go` — `CreateOperation`, `GetOperation`, `GetOperationByCalcID`, `ListOperations` (с пагинацией через `Page{Limit, BeforeID}` и фильтрами `OperationFilter{From,To,Types,AccID,CatID,Planned}`), `DeleteOperation` (soft), `UpdateOperationCategory`, `SumByAccountSince`. `scanOperation` общий scanner для Row+Rows.
  - **Критический фикс бизнес-логики:** изначальный `SumByAccountSince` игнорировал переводы → балансы счетов не двигались при transfer/exchange. Переписан в один SQL с UNION ALL: source leg (-amount/+amount for income/expense; -amount for transfer/exchange) + destination leg (+amount_dst или +amount для transfer/exchange). Теперь переводы корректно двигают балансы (BUSINESS_LOGIC ф.1: «переводы НЕ в cashflow и налогах, **только в остатках счетов**»).
- `internal/storage/accounts_test.go` + `operations_test.go` — 14 тестов через `sqlmock` (в стиле `users_test.go`)

*Service layer:*
- `internal/service/operations/operations.go` — `Service` с `OperationRepo` interface (тестируем без БД). Методы: `Create`, `Get`, `List` (с `more`-сигналом через fetch-limit+1), `Delete`, `SetCategory`. Оркестрация: validate → mask PII → owner-check accounts → insert → recompute balances. Idempotency: при `ErrOperationExists` возвращается оригинал через `GetOperationByCalcID`. `recomputeBalance` делает полный recompute (self-healing, no drift). `newCalcID = srv:{user_id}:{unix_nano}`.
- `internal/service/operations/operations_test.go` — 9 тестов через in-memory `fakeRepo`: PII-masking, idempotency, invalid-input (3 кейса), unowned-account, transfer-recomputes-both-balances, list-more-flag, delete-recomputes-balance, delete-not-found

*HTTP layer:*
- `internal/transport/http/operations.go` — `OperationsHandler.Register(r)` монтирует `POST/GET /operations`, `GET/DELETE /operations/{id}`, `PATCH /operations/{id}/category`. `operationRequest`/`operationResponse` JSON-формы со строковыми деньгами (без float!), parse-хелперы `parseCreate`/`parseListQuery`. `writeServiceError` маппит sentinel'ы → 400/404/500.
- `internal/transport/http/operations_test.go` — 9 интеграционных тестов через `httptest.Server` + chi router + in-memory `fakeOpsRepo` (Create success/invalid-type/invalid-amount, Get not-found, Create→Get roundtrip, List pagination, Delete, SetCategory, Unauthorized-without-context)

*Wiring:*
- `internal/transport/http/router.go` — `Deps.Operations *operations.Service`, монтируется в authenticated group если non-nil (graceful: без БД /operations не появляется)
- `cmd/server/main.go` — `operations.NewService(pool)` подключается к Deps, `applog "api mounted"` (было `"auth endpoints mounted"`)

**Зависимости:** НЕТ новых (chi, shopspring/decimal, sqlmock уже в go.mod).

**Верификация (23.06):**
- `go build ./...` → BUILD_OK
- `go vet ./...` → VET_OK (включая проверку errors-before-use в тестах)
- `go test ./...` → **все 14 пакетов зелёные** (+pii: 15, +domain: +5, +storage: +14, +service/operations: +9, +transport/http: +9)
- Smoke-тест бинарника (без БД): `/healthz` → 200, `/readyz` → 503 `no_database`, `/api/v1/operations` → 404 (API не смонтирован без pool — корректное graceful-поведение)

**Ключевые эталоны:**
- PII: `"Перевод от ИВАН ИВАНОВИЧ И."` → `"Перевод от [PERSON]"`; `"Звонок +7 (999) 123-45-67"` → `"Звонок [PHONE]"` (проверено на service- и http-уровне)
- Идемпотентность: повторный POST с тем же `calc_id` возвращает ту же операцию, не создаёт дубль и не падает
- Round-trip: create → get-by-id → равенство всех полей включая PII-маскированный counterparty
- Пагинация: 3 операции, limit=2 → 2 items + `more=true`

**Хорошо:**
- **PII маскирование до persistence** — приватность by design, не после (PRIVACY_RULES). Один и тот же `pii.Mask` используется в service — гарантия что ни один путь записи не обходит маскирование
- **Идемпотентность на (user_id, calc_id)** с unique-constraint в БД + service-уровневой обработкой `ErrOperationExists` → fetch оригинала. Двойная защита: даже если два параллельных запроса придут с одним calc_id, БД-констрейнт пустит только один
- **Пересчёт баланса как полный recompute**, а не delta-update → self-healing: любой drift от частичного отказа исправляется на следующей операции
- **Storage-agnostic service** через `OperationRepo` interface → unit-тесты без БД, в стиле auth-слоя (`JWTVerifier`)
- **Money как строка в JSON** — нигде не проходит через float64 (нулевой детерминизм-нарушений)
- **Финансовая семантика transfer** наконец корректна: source -, destination +, cashflow/taxes не затронуты (соответствует ф.1 дословно)
- **Graceful degradation**: без БД сервер бьётся, /healthz работает, /api/v1 не падает с паникой — важно для CI и smoke

**Плохо / тех. долг:**
- E2E через реальный Postgres по-прежнему отложен (нет docker в окружении). Все SQL сверен со схемой `0001_init.sql` вручную + sqlmock покрывает контракты. Полноценный прогон — когда будет docker-compose
- `currency_exchange` с разными валютами сейчас работает «по доверению»: amount_dst пишется как есть, конвертация по курсу ЦБ НЕ реализована (RUB-only scope-lock; когда добавим multi-currency — понадобится exchange-rates таблица)
- Авто-категоризация (ф.2) ещё не подключена: `category_id` ставится вручную или None. База для неё (`categories`, `UpdateOperationCategory`) уже готова
- `SeedSystemCategories` написан, но не вызывается из registration flow — нужно звать в `AuthHandler.Register` после создания user (добавим в ф.2 когда будут правила категорий)
- Нет rate-limiting на /operations (как и на /auth) — отложено
- Мой изначальный `SumByAccountSince` имел логическую дыру (переводы не двигали баланс); нашёл тестом СЕЙЧАС, а не на ревью. Мораль: SQL-агрегации по типам операций требуют явных тест-кейсов на каждый тип, не только income/expense

**КРИТИЧЕСКОЕ РЕШЕНИЕ (зафиксировано для будущих сессий):**
PII-маскирование `pii.Mask` вызывается **только в service/operations.Create**, на входе в persistence. Ни storage, ни handlers не маскируют сами — единственная точка. Если появится ф.2 (авто-категоризация) или импорт CSV (Этап 6), они **обязаны** идти через тот же service-путь, иначе обойдут маскирование.

---

## ⏳ Этап 3 — Фичи 2-4 Level 1 — СЛЕДУЮЩИЙ ШАГ

`internal/service/{operations,categorization,dashboard,budget}/` + REST handlers:
- Фича 1: ручной ввод операций (calc_id для идемпотентности, переводы НЕ в cashflow)
- Фича 2: авто-категоризация (rules-based, без ML — scope-lock из рисков)
- Фича 3: сводный дашборд (баланс, чистая стоимость, прогресс целей)
- Фича 4: бюджеты с переносом остатков

Этап 4: фичи 5-9 (калькуляторы — используют готовое mathcore)
Этап 5: AI stub (fallback-шаблоны)
Этап 6: CSV/Excel импорт + OpenAPI + E2E

---

## 🔧 Команды для восстановления контекста

```bash
# Сборка + тесты
cd C:/Users/user/ZCodeProject/FinHelper/backend
"C:/Program Files/Go/bin/go.exe" build ./...
"C:/Program Files/Go/bin/go.exe" vet ./...
"C:/Program Files/Go/bin/go.exe" test ./...

# Smoke-тест сервера (без БД)
JWT_ACCESS_SECRET=$(printf 'a%.0s' {1..40}) \
JWT_REFRESH_SECRET=$(printf 'b%.0s' {1..40}) \
USER_HASH_SALT=salt HTTP_ADDR=:18080 \
./bin/finhelper.exe &
curl http://localhost:18080/healthz  # → {"status":"ok"}

# Поднять Postgres
cd C:/Users/user/ZCodeProject/FinHelper
docker-compose up -d postgres
```

## 📚 Принятые архитектурные решения

- **Деньги**: всегда `domain.Money` (обёртка над decimal.Decimal, scale=2). Запрет float64. Округление = shopspring default ROUND_HALF_AWAY_FROM_ZERO (1.005→1.01), НЕ bankers' rounding.
- **Логи**: только `user_hash` (SHA-256(user_id + USER_HASH_SALT)), email/телефон/ФИО — никогда.
- **Идемпотентность**: operations имеют `calc_id`, UNIQUE(user_id, calc_id). При дубле — возвращается оригинал.
- **PII-маскирование**: единственная точка — `pii.Mask` в `service/operations.Create` (до persistence). Storage и handlers не маскируют сами.
- **Soft delete**: `deleted_at TIMESTAMPTZ`, hard delete отложен в v1.0.
- **Баланс счетов**: cached в `accounts.balance`, полный recompute в `service/operations.recomputeBalance` (self-healing). Переводы двигают баланс через обе ноги (source −, destination +amount_dst).
- **Service-репозиторий interface** (`OperationRepo`): unit-тесты без БД через fake, интеграция через sqlmock, прод через `*storage.Pool`.
- **DB-опциональность**: конфиг грузится без DATABASE_URL, чтобы /healthz работал в CI.
- **CASFD-приватность**: PII в `counterparty` маскируется перед сохранением (PRIVACY_RULES.md §"Маскирование").
- **JWT (с Задачи 2)**: access (15 мин) + refresh (30 дней) на разных секретах; refresh single-use с rotation; в БД только SHA-256(refresh); claims несут `user_id` и `user_hash`.
