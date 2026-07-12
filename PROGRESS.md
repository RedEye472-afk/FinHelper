# FinHelper — PROGRESS.md

> Compact status for AI handoff. **Стек:** Go 1.26 + React 18/TS | PostgreSQL 16 (Neon) | JWT auth | RUB only | `shopspring/decimal` (float64 запрещён) | Vercel λ + SPA
>
> **Репозиторий:** `https://github.com/RedEye472-afk/FinHelper` (origin=main)
> **Production:** `https://finhelper-frontend.vercel.app`
> **Окружение:** Windows, `C:\Program Files\Go\bin\go.exe`, Git Bash, Node v24.17.0, npm based, Vite, Vercel CLI v54.21.1

---

## 🏛 Ключевые архитектурные решения (ADRs)

| # | Решение | Детали |
|---|---------|--------|
| 1 | **Деньги = только decimal** | `shopspring/decimal`, scale=2, `ROUND_HALF_AWAY_FROM_ZERO`. В JSON — строки. float64 запрещён. Domain: `domain.Money`. |
| 2 | **PII-маскирование до persistence** | `pii.Mask()` вызывается строго в service/operations.Create. Ни storage, ни handler не маскируют сами. В логах только `user_hash`. |
| 3 | **JWT: access в памяти, refresh rotation** | Access 15min (Zustand store), Refresh 30d (httpOnly Secure cookie). SHA-256 в БД. Single-use: `UPDATE...RETURNING`. |
| 4 | **Идемпотентность** | `calc_id` → UNIQUE(user_id,calc_id) для операций; `contribution_id` → UNIQUE(user_id,goal_id,contribution_id) для целей. |
| 5 | **Vercel Go λ** | `api/index.go` (package handler, `func Handler`). root `go.mod` с `replace backend => ./backend`. Chi-router, **не** bridge.Start(). |
| 6 | **Embed.FS + Windows** | Всегда `path.Join` (forward-slash), не `filepath.Join` — иначе panic на Windows. |
| 7 | **Миграции: синхронные 60s** | Асинхронные приводили к частичной схеме. `verifySchema` — log.Printf (не fatal). |
| 8 | **ПСК ≠ XIRR** | ПСК (кредиты) = **номинальная** по ЦБ 5750-У. XIRR (инвестиции) = **эффективная** годовая доходность. |
| 9 | **НДФЛ = marginal brackets** | 5 ступеней (13/15/18/20/22%), каждая к своему slice. Не «всё по верхней». |
| 10 | **Goals hybrid model** | `effective_current = goals.current_amount (baseline) + Σ goal_contributions.amount`. |
| 11 | **Float64 bridges** | Ровно 2: `credit.BrentQ` (ПСК) и `investment.XIRR`. Весь остальной код — строго decimal. |

---

## ✅ Статус фич

| Фича | Пакет | Тестов | Статус |
|------|-------|--------|--------|
| 0–2. Backend scaffold, auth, daycount+tvm | config/domain/auth/ + mathcore | 38 | ✅ |
| 3–5. Credit, investment, tax (mathcore) | `mathcore/credit`, `mathcore/investment`, `mathcore/tax` | 51 | ✅ |
| 6. Операции (CRUD + PII + idemp) | `service/operations` + storage/HTTP | 38 | ✅ |
| 7. Авто-категоризация | `service/categorization` | 28 | ✅ |
| 8. Дашборд + Бюджеты | `service/dashboard` + `service/budget` | 53 | ✅ |
| 9. Трекер целей (ф.5) | `service/goals` (18 задач, CRUD+proj+simulate) | 83 | ✅ |
| 10. Кредитный калькулятор (ф.7) | `service/credit` + mathcore | ~40 | ✅ |
| 11. **Калькулятор вкладов (ф.6)** | **`service/deposit` + mathcore + HTTP + frontend** | **36** | **✅ (09.07)** |
| 12. Backend + Frontend: деплой на Vercel | `frontend/` + `api/index.go` | — | ✅ (12.07) |
| 13. Импорт выписок Сбербанк PDF | `frontend/src/lib/import/`, `ImportPage`, `scripts/pdf_parse.py`, `backend/internal/handler/pdf_parse.go` | — | ✅ (12.07) |
| 14. OSS Integration — TanStack Table/Virtual, React Hook Form, date-fns, Sonner | `@tanstack/react-virtual`, `@tanstack/react-table`, `react-hook-form`, `date-fns`, `sonner` | — | ✅ (12.07) |

**Total Go test packages:** 22 (все зелёные, проверено 09.07). **Documented float64 bridges:** 2.

---

## 🧮 Mathcore — все пакеты

| Пакет | Формулы | Источник |
|-------|---------|----------|
| `daycount` | ACT/365, 30/360 ISDA, ACT/ACT ISMA | Копнова Гл. 1 |
| `tvm` | Simple, Compound, Effective, Fisher | Копнова Гл. 1-2 |
| `credit` | Annuity, Differentiated, PSK (ЦБ 5750-У), Early, **BrentQ** | Копнова Гл. 4, ЦБ 5750-У |
| `investment` | NPV, XIRR (BrentQ), MIRR, DPP, PI | Копнова Гл. 5-6 |
| `tax` | НДФЛ (5-tier marginal), НПД, УСН, DepositTax (ФЗ-382) | НК РФ Гл. 23/26.2, ФЗ-422/382/257 |
| `goals` | Sinking Fund: FV, Contribution, Term, InflateTarget | Копнова Гл. 3.3.3 |
| `deposit` | Compound/Simple interest, cap freq, effective rate, Fisher, projection | Копнова Гл. 1-2 |

**Shopspring gotchas:** `Pow(d2) Decimal` — без error; `Ln(precision int32) (Decimal, error)` — с error.

---

## 🚀 Vercel Deploy — текущее состояние (12.07)

### ✅ Работает
- Frontend SPA: `https://finhelper-frontend.vercel.app` (Vite build, 2281 modules)
- Go λ: `api/index.go` (chi-router), `/healthz` и `/readyz` отвечают
- Build: стабильно зелёный (~2 min). `go build/vet/test` — локально зелёные.
- **Premium Dark Neon** тема (#0B1020 bg, #6E56CF primary)
- **Banking-style UI**: bottom nav, сайдбар, страницы Операции/Цели/Бюджеты/Настройки
- **Импорт выписок**: PDF (через pdf.js + py-pdf-parser) и CSV, массовый импорт
- **Счета**: создание/удаление, типы счетов с иконками

### ❌ Что не работает
- **Neon DB connection таймаутит из Vercel λ.** Функция не отвечает (ни degraded mode, ничего).
  - Строка: `postgresql://neondb_owner:npg_3RGdJ8DlAWbT@ep-long-base-at9e3d0y-pooler.c-9.us-east-1.aws.neon.tech/neondb?sslmode=require`
  - Из локальной среды (pip pg8000): **соединение в 1.3с** — пароль верный, хост жив
  - Из Vercel: полный таймаут (>60s). Предыдущая строка (с неверным паролем) отвечала «degraded mode» за <10s
  - **Гипотезы:** (а) Vercel Hobby (10s init timeout) не хватает на холодный старт λ + Neon pooler; (б) pooler (`ep-long-base-at9e3d0y-pooler`) блокирует IP Vercel; (в) pgx v5.10.0 зависает на SCRAM-SHA-256 с этим конкретным pooler.
- Все эндпоинты, требующие БД, недоступны (auth, operations, goals, dashboard, budgets). Stateless `/calc/*` могли бы работать, но без БД не проходит init.

### Структура Vercel λ

```
api/index.go
├── config.Load() → cfg
├── storage.Open(cfg.Database.URL) → pool  ← 5s ping timeout, вешается
├── migrate.Run(ctx, pool)  ← 60s context, не доходит
├── auth.NewJWTIssuer(...)
├── service init (operations, categorization, dashboard, budget, goals, credit, deposit)
├── chi.NewRouter()
│   ├── /healthz → {"status":"ok"}
│   ├── /readyz  → {"status":"ready"} (или degraded)
│   ├── /migrate → trigger migration
│   └── /api/v1/* → apiRouter (transporthttp.NewRouter)
└── func Handler(w, r) { getHandler().ServeHTTP(w, r) }
```

После смены DATABASE_URL на рабочий пароль — функция не доходит даже до degraded mode. Проблема: `storage.Open()` с корректными кредами не успевает за 10s Vercel инициалайз либо pooler вешается.

---

## 📦 Последние коммиты

| Хеш | Время | Описание |
|-----|-------|----------|
| `df05c90` | 09.07 12:00 | feat(deposit): implement ф.6 deposit calculator — mathcore + service + HTTP |
| `b043d67` | 09.07 15:40 | feat(deposit): wire deposit service in Vercel API + rework deposit page |
| current | 09.07 16:30+ | +DATABASE_URL updates, multiple redeploys |

---

## 🔧 Команды (fast reference)

```bash
# Backend build + test
cd /c/Users/user/ZCodeProject/FinHelper/backend
"C:/Program Files/Go/bin/go.exe" build ./... && "C:/Program Files/Go/bin/go.exe" vet ./... && "C:/Program Files/Go/bin/go.exe" test -count=1 ./...

# Frontend build
cd /c/Users/user/ZCodeProject/FinHelper/frontend && npm run build

# Vercel deploy (из frontend/ — слинкован с finhelper-frontend)
cd /c/Users/user/ZCodeProject/FinHelper/frontend && npx vercel deploy --prod --force --yes

# Vercel env management
npx vercel env ls production
npx vercel env rm DATABASE_URL production --yes
echo -n "new_url" | npx vercel env add DATABASE_URL production --yes

# Проверить λ
curl -s --max-time 15 https://finhelper-frontend.vercel.app/api/v1/healthz
```

---

## 🧩 Эта сессия (09.07) — что сделано

### Задача: Синхронизировать фронтенд с новыми фичами и задеплоить

**Что сделано:**
1. **`api/index.go`** — добавлена инициализация `depSvc := deposit.NewService()` + `Deposit: depSvc` в Deps (было упущено, Go build проходил).
2. **`calculators.ts`** — добавлены типы `DepositRequest`, `DepositResponse`, `DepositProjectionRow` + функция `calculateDeposit`.
3. **`queries.ts`** — добавлен хук `useDepositCalc` (mutation).
4. **`DepositPage.tsx`** — переписан с client-side `decimal.js` на вызов `POST /api/v1/calc/deposit` через `useDepositCalc`. Добавлены: projection table, loading/error states, real return, tax, disclaimer. Стиль унифицирован с CreditPage.
5. **Коммит и пуш** `b043d67` → авто-деплой на Vercel.
6. **Несколько редеплоев** с разными `DATABASE_URL` (исходная с `channel_binding=require`, очищенная, с устаревшим паролем).

### Текущий блокер: Vercel λ не стартует с правильным DATABASE_URL
- Старая строка (неверный пароль) → degraded mode за <10s ✅
- Новая строка (правильный пароль) → полный таймаут >60s ❌
- Neon проверен из локального окружения — работает (1.3s, PostgreSQL 18.4)
- Возможные причины:
  a) Пул соединений pgx + Neon pooler зависает при SCRAM-SHA-256
  b) Vercel Hobby (10s init timeout) прерывает холодный старт
  c) Neon pooler endpoint блокирует Vercel IP-диапазоны
  d) `stdlib.OpenDB` + `PingContext(5s)` не хватает на полный handshake

### План следующей сессии
1. **Диагностировать** таймаут λ: попробовать `sslmode=disable` для Neon, увеличить ping timeout, или перейти на прямой хост (не pooler).
2. **Решить БД**: или фикс Neon connection, или замена на локальный Docker PG + ngrok/tunnel.
3. **Проверить** все эндпоинты после восстановления БД: auth register/login, operations CRUD, dashboard, budget, goals, калькуляторы.
4. **Добавить** E2E тесты для деплоя (если будет CI).
