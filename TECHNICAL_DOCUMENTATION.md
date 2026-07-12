# FinHelper — Technical Documentation

## 1. Architecture Overview

**FinHelper** is a full-stack personal finance application with a **Go backend** and **React/TypeScript frontend**, deployed on **Vercel** (serverless Go) and **PostgreSQL**.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           FINHELPER ARCHITECTURE                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────┐      HTTPS/REST/JSON       ┌──────────────────────────┐   │
│  │   FRONTEND   │ ◄─────────────────────────► │        BACKEND           │   │
│  │  (React 19)  │   /api/v1/* endpoints       │   (Go 1.26, Chi router)  │   │
│  │              │   JWT Access/Refresh        │                          │   │
│  │  Vite + TS   │   TanStack Query (v5)       │  pgx/v5 + database/sql   │   │
│  │  Tailwind 4  │   React Router v7           │  mathcore (decimal math) │   │
│  └──────────────┘                             └──────────────┬─────────────┘   │
│                                                              │                │
│                        ┌─────────────────────────────────────┘                │
│                        ▼                                                      │
│               ┌─────────────────────┐                                        │
│               │   POSTGRESQL 16     │                                        │
│               │   (Supabase / Neon) │                                        │
│               └─────────────────────┘                                        │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Technology Stack

### Frontend (`/frontend`)

| Layer | Technology | Version | Purpose |
|-------|------------|---------|---------|
| Framework | React | 19.2.7 | UI library |
| Language | TypeScript | 6.0.3 | Type-safe development |
| Build | Vite | 8.1.3 | Fast dev/build |
| Styling | Tailwind CSS | 4.3.1 | Utility-first CSS |
| Routing | React Router | 7.18.0 | SPA routing |
| State/Server | TanStack Query | 5.101.2 | Server state, caching |
| Forms | React Hook Form | 7.81.0 | Form validation |
| Validation | Zod (via @hookform/resolvers) | 5.4.0 | Schema validation |
| Charts | Recharts | 3.9.0 | Data visualization |
| Animations | Framer Motion | 12.42.2 | Page transitions |
|| Math | decimal.js | 10.6.0 | Arbitrary-precision decimal |
|| PDF | pdfjs-dist | 6.1.200 | Bank statement parsing |
|| Icons | Lucide React | 1.21.0 | Icon system |
|| Notifications | Sonner | 2.0.7 | Toast notifications |
|| Tables | @tanstack/react-table | 8.21.3 | Sorting, filtering, pagination |
|| Virtualization | @tanstack/react-virtual | 3.14.6 | Virtual scrolling for large lists |
|| Forms | React Hook Form | 7.81.0 | Form validation |
|| Validation | Zod (via @hookform/resolvers) | 5.4.0 | Schema validation |
|| Date utils | date-fns | 4.4.0 | Modern date formatting |

### Backend (`/backend`)

| Layer | Technology | Version | Purpose |
|-------|------------|---------|---------|
| Language | Go | 1.26.4 | Core language |
| Router | Chi | 5.3.0 | HTTP routing + middleware |
| DB Driver | pgx/v5 + database/sql | 5.10.0 | PostgreSQL driver |
| Auth | golang-jwt/v5 | 5.2.2 | JWT tokens |
| Crypto | golang.org/x/crypto | 0.53.0 | bcrypt, SHA-256 |
| Decimal | shopspring/decimal | 1.4.0 | Exact monetary math |
| Config | YAML (gopkg.in/yaml.v3) | 3.0.1 | Tax rules config |
| Testing | testify, sqlmock | - | Unit/integration tests |

### Infrastructure

| Component | Technology |
|-----------|------------|
| Database | PostgreSQL 16 (Supabase/Neon) |
| Hosting (FE) | Vercel (static + serverless) |
| Hosting (BE) | Vercel Go serverless functions |
| CI/CD | GitHub Actions (implied) |
| Container | Docker (multi-stage) |

---

## 3. Backend Architecture

### 3.1 Package Structure

```
backend/
├── cmd/
│   └── server/           # Main entry point (main.go)
├── internal/
│   └── handler/
│       └── pdf_parse.go  # PDF bank statement extraction endpoint
├── pkg/
│   ├── auth/             # JWT, password hashing, user_hash
│   ├── config/           # Environment configuration
│   ├── domain/           # Core business types (Money, Operation, Goal, Account)
│   ├── email/            # Multi-provider email sender (Resend/SendGrid/Brevo)
│   ├── log/              # Structured logging (zerolog-style)
│   ├── mathcore/         # Pure financial mathematics (no I/O)
│   │   ├── credit/       # Annuity, differentiated, PSK, early repayment
│   │   ├── deposit/      # TVM: compound/simple interest, effective rate
│   │   ├── goals/        # Sinking fund formulas (Kopnova Ch.3.3.3)
│   │   ├── tax/          # Russian deposit tax (ФЗ-382)
│   │   └── tvm/          # Time Value of Money primitives
│   ├── migrate/          # Migration runner
│   ├── pii/              # PII masking (PRIVACY_RULES.md)
│   ├── ratelimit/        # Token bucket rate limiter
│   ├── service/          # Business logic layer (stateless services)
│   │   ├── budget/       # Бюджеты с rollover (ф.4)
│   │   ├── categorization/ # Auto-categorization (ф.2)
│   │   ├── credit/       # Credit calculator (ф.7)
│   │   ├── dashboard/    # Dashboard aggregates (ф.3)
│   │   ├── deposit/      # Deposit calculator (ф.6)
│   │   ├── goals/        # Savings goals (ф.5)
│   │   └── operations/   # Operations + balance recompute (ф.1)
│   ├── storage/          # Database access layer (sqlc-style)
│   │   ├── accounts.go
│   │   ├── operations.go
│   │   ├── budgets.go
│   │   ├── goals.go
│   │   ├── categorization.go
│   │   └── ...
│   └── transport/
│       └── http/         # HTTP handlers, middleware, router
└── queries/              # SQL queries (if using sqlc)
```

### 3.2 Service Layer Pattern

Each feature follows the **Service Layer** pattern:

```
┌─────────────────────────────────────────────────────────────┐
│                    HTTP HANDLER (transport/http)             │
│  - Decodes JSON → validates → calls Service                  │
│  - Maps service errors → HTTP status codes                   │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                      SERVICE (pkg/service/*)                 │
│  - Pure business logic, no HTTP concerns                     │
│  - Depends on REPO interfaces (testable with fakes)          │
│  - Owns validation, idempotency, derived calculations        │
│  - Stateless: construct once at boot, share across requests  │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                      REPO (pkg/storage)                      │
│  - Thin SQL wrapper, maps rows → domain types                │
│  - Single source of truth for queries                        │
│  - Satisfies service REPO interfaces                         │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                    POSTGRESQL (via pgx pool)                 │
└─────────────────────────────────────────────────────────────┘
```

**Key principle**: Services are stateless singletons created at startup in `main.go`, injected with repository implementations (`*storage.Pool`). This enables unit testing with in-memory fakes.

### 3.3 Domain Types (`pkg/domain`)

All monetary values use **`domain.Money`** — a wrapper around `shopspring/decimal.Decimal` with **fixed scale = 2** (kopecks). **Never** use `float64` for money.

```go
type Money struct { v decimal.Decimal }  // scale=2, ROUND_HALF_AWAY_FROM_ZERO

// Constructors
func NewMoney(d decimal.Decimal) Money
func ParseMoney(s string) (Money, error)
func FromDecimal(d decimal.Decimal) Money  // for DB scans

// Arithmetic (returns new Money, scale preserved)
func (m Money) Add(b Money) Money
func (m Money) Sub(b Money) Money
func (m Money) Mul(factor decimal.Decimal) Money

// Comparisons
func (m Money) Cmp(b Money) int
func (m Money) IsPositive() bool
func (m Money) IsZero() bool
```

**Operation Types** (BUSINESS_LOGIC.md §1):
- `income` / `expense` — affect cashflow
- `transfer` / `currency_exchange` / `refund` — **excluded from cashflow**, only move balances

---

## 4. API Contract

### 4.1 Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/auth/register` | Register (returns message, requires email verification) |
| POST | `/api/v1/auth/login` | Login (returns tokens if verified, else `requires_verification=true`) |
| POST | `/api/v1/auth/verify-email` | Verify 6-digit code → returns tokens |
| POST | `/api/v1/auth/send-code` | Resend verification code |
| POST | `/api/v1/auth/forgot-password` | Send reset link |
| POST | `/api/v1/auth/reset-password` | Consume reset token |
| POST | `/api/v1/auth/refresh` | Rotate refresh token → new access+refresh |
| POST | `/api/v1/auth/logout` | Revoke refresh token |
| GET | `/api/v1/me` | Current user profile |
| POST | `/api/v1/auth/demo-login` | Demo mode (MVP) |

**Token Design** (`pkg/auth/jwt.go`):
- Access: 15 min default, HS256, claims: `uid`, `uhsh`, `kind="access"`
- Refresh: 30 days default, HS256, stored as **SHA-256 hash** in DB
- Rotation on refresh: old refresh revoked, new pair issued

### 4.2 Core Endpoints

| Feature | Endpoints |
|---------|-----------|
| **Accounts** | `GET/POST /api/v1/accounts`, `PATCH/DELETE /api/v1/accounts/:id` |
| **Operations** | `GET/POST /api/v1/operations`, `DELETE /api/v1/operations/:id`, `PATCH /api/v1/operations/:id/category` |
| **Categories** | `GET /api/v1/categories` |
| **Dashboard** | `GET /api/v1/dashboard?period=month` |
| **Budgets** | `GET/POST /api/v1/budgets`, `PATCH/DELETE /api/v1/budgets/:id`, `GET /api/v1/budgets/:id/status` |
| **Goals** | `GET/POST /api/v1/goals`, `PATCH/DELETE /api/v1/goals/:id`, `POST /api/v1/goals/:id/contributions`, `GET /api/v1/goals/:id/projection`, `POST /api/v1/goals/:id/simulate` |
| **Calculators** | `POST /api/v1/calc/credit`, `POST /api/v1/calc/deposit`, `POST /api/v1/calc/affordability` |

### 4.3 Response Format

**Success**: Direct object or `{ items: T[], more: boolean }` for lists.

**Error** (RFC 7807 Problem Details):
```json
{
  "type": "about:blank",
  "title": "Bad Request",
  "status": 400,
  "detail": "amount must be positive"
}
```

---

## 5. Database Schema (Migration 0001)

```sql
-- ENUMs
operation_type: income | expense | transfer | currency_exchange | refund
income_subtype: salary | fee | gift | investment | loan_repayment
account_type: cash | bank | savings | investment | crypto | debt

-- TABLES
users (id, email, password_hash, user_hash, created_at, updated_at, deleted_at)
refresh_tokens (id, user_id, token_hash, expires_at, revoked_at, created_at)
accounts (id, user_id, name, account_type, currency, balance, created_at, updated_at, deleted_at)
categories (id, user_id, name, parent_id, is_system, created_at, deleted_at)
operations (id, user_id, calc_id UNIQUE(user_id), operation_type, amount, amount_dst, currency, account_id, account_dst_id, category_id, income_subtype, counterparty, description, operation_date, is_planned, category_confidence, created_at, updated_at, deleted_at)
goals (id, user_id, name, target_amount, current_amount, monthly_contribution, target_date, expected_yield, created_at, updated_at, deleted_at)
goal_contributions (id, user_id, goal_id, contribution_id UNIQUE(user,goal), amount, contribution_date, comment, created_at)
budgets (id, user_id, category_id UNIQUE(user_id), limit_amount, rollover_policy, is_active, created_at, updated_at, deleted_at)
```

**Indexes**: Partial indexes on `deleted_at IS NULL` for all user-scoped tables.

**Money columns**: `NUMERIC(28,2)` with `CHECK (amount = ROUND(amount, 2))`.

---

## 6. Financial Mathematics (`pkg/mathcore`)

### 6.1 Time Value of Money (`tvm/`)
- `CompoundInterest(P, r, n, t)` — `P(1 + r/n)^(nt)`
- `SimpleInterest(P, r, t)` — `P(1 + rt)`
- `EffectiveRate(nominal, periodsPerYear)` — `(1 + r/n)^n - 1`
- `FisherRealRate(nominal, inflation)` — `(1+r)/(1+π) - 1`

### 6.2 Credit (`credit/`)
- **Annuity**: `P = S * r(1+r)^n / ((1+r)^n - 1)`
- **Differentiated**: Principal = S/n constant; Interest = remaining × r
- **PSK (ПСК)**: CBR 5750-У — IRR on cashflows (disbursement net of upfront fees, then negative payments including monthly fees). Solved via **Brent's method** on float64 (only float64 in codebase).
- **Early Repayment**: Re-amortize remaining balance at same rate.

### 6.3 Deposit (`deposit/` via `tvm/`)
- Capitalization: monthly / quarterly / annually / maturity
- Effective rate, real return (Fisher), tax per ФЗ-382

### 6.4 Goals (`goals/`)
- **Sinking fund** (Kopnova Ch. 3.3.3):
  - `FV = PV(1+i)^n + PMT * ((1+i)^n - 1)/i`
  - Solve for `PMT` (required monthly) or `n` (months to target)
- Hybrid current amount: `effective = current_amount + Σ contributions`

### 6.5 Tax (`tax/`)
- YAML rules per year (2024/2025/2026): `key_rate_jan1`, `non_taxable_limit = 1_000_000 * key_rate`
- Tax = max(0, interest - non_taxable) × 13%

---

## 7. Frontend Architecture

### 7.1 Project Structure

```
frontend/src/
├── api/              # TanStack Query hooks + API client
│   ├── client.ts     # fetch wrapper, auto-refresh, token storage
│   ├── auth.ts       # Auth endpoints
│   ├── queries.ts    # React Query hooks (useAccounts, useOperations, ...)
│   └── calculators.ts# Calculator mutations
├── components/
│   ├── layout/       # AppLayout, Header, MobileNav, *Menu
│   ├── ui/           # Primitive components (Button, Input, Card, Modal, ...)
│   └── shared/       # Cross-feature (AiExplanation, FormulaTooltip, Disclaimer)
├── context/
│   └── AuthContext.tsx   # User session, login/logout, demo fallback
├── hooks/
│   ├── useSettings.ts    # localStorage settings (theme, currency, hideBalance)
│   ├── useKeyboard.ts    # Global hotkeys (g=goals, o=operations, ...)
│   └── useBreakpoint.ts  # Responsive breakpoint hook
├── lib/
│   ├── money.ts      # decimal.js wrapper (formatMoney, toDecimal, M.*)
│   ├── import/       # Bank PDF/CSV parsers (Sberbank, Tinkoff, etc.)
│   └── utils/        # dates, forms, toast helpers
├── pages/            # Route components (Dashboard, Operations, Credit, Deposit, Goals, Budgets, ...)
├── store.tsx         # FinanceProvider → useDashboard() → KPI + categories
├── types.ts          # Shared TypeScript interfaces (matches Go API)
└── App.tsx           # Router, providers, lazy-loaded calculator pages
```

### 7.2 Money Handling (Frontend)

**Rule**: Money = **strings** in API ↔ **Decimal.js** in UI. Never `number`/`float`.

```typescript
// lib/money.ts
export function toDecimal(s: string | null | undefined): Decimal
export function formatMoney(d: Decimal | string | number): string  // "1 234,56 ₽"
export function formatCompact(d: Decimal): string                  // "1,2 млн ₽"
export function moneyToString(d: Decimal): string                  // "1234.56" for API

// Arithmetic helpers
export const M = {
  add: (a,b) => a.plus(b), sub: (a,b) => a.minus(b),
  mul: (a,b) => a.mul(b), div: (a,b) => a.div(b),
  zero: () => new Decimal(0),
  fromInput: (s: string) => safeParse(s),
  toInput: (d: Decimal) => d.toFixed(2),
}
```

### 7.3 Data Fetching (TanStack Query v5)

```typescript
// Pattern: query hooks in api/queries.ts
export function useOperations(limit = 50, before?: number) {
  return useQuery({
    queryKey: ['operations', limit, before],
    queryFn: () => operationsApi.listOperations(limit, before),
  })
}

// Mutations invalidate related queries
export function useCreateOperation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: operationsApi.createOperation,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['operations'] })
      qc.invalidateQueries({ queryKey: ['dashboard'] })
    },
  })
}
```

### 7.4 Routing & Layout

- **Protected routes** via `<ProtectedRoute>` wrapper (redirects to `/login` if no token)
- **Lazy-loaded** calculator pages (KaTeX heavy): `DepositPage`, `CreditPage`, `AffordabilityPage`, `MortgageVsRentPage`
- **Desktop**: Fixed sidebar (280px) + main content
- **Mobile**: Bottom nav + slide-up sheets

---

## 8. Key Business Features

| Feature | Spec Ref | Backend Service | Frontend Page |
|---------|----------|-----------------|---------------|
| **Operations (Ф.1)** | BUSINESS_LOGIC.md §1 | `service/operations` | `OperationsPage`, `OperationsNew` |
| **Auto-categorization (Ф.2)** | BUSINESS_LOGIC.md §2 | `service/categorization` | Integrated in create op |
| **Dashboard (Ф.3)** | BUSINESS_LOGIC.md §3 | `service/dashboard` | `DashboardPage` |
| **Budgets + Rollover (Ф.4)** | BUSINESS_LOGIC.md §4 | `service/budget` | `BudgetsPage` |
| **Goals (Ф.5)** | BUSINESS_LOGIC.md §5 | `service/goals` | `GoalsPage` |
| **Deposit Calc (Ф.6)** | BUSINESS_LOGIC.md §6 | `service/deposit` | `DepositPage` |
| **Credit Calc (Ф.7)** | BUSINESS_LOGIC.md §7 | `service/credit` | `CreditPage` |
| **Affordability (Ф.8)** | BUSINESS_LOGIC.md §8 | `service/credit` (shared) | `AffordabilityPage` |
| **Mortgage vs Rent (Ф.9)** | BUSINESS_LOGIC.md §9 | — (frontend only) | `MortgageVsRentPage` |

---

## 9. Security & Privacy

| Area | Implementation |
|------|----------------|
| **Auth** | bcrypt (cost 12), JWT HS256, separate access/refresh secrets |
| **PII** | Counterparty/description masked at write (`pkg/pii/mask.go`) — `[PERSON]`, `[CARD]`, `[PHONE]`, `[EMAIL]` |
| **CORS** | Configurable origins via `CORS_ALLOWED_ORIGINS` |
| **Rate Limit** | Token bucket (10 req/min/IP) on auth endpoints |
| **SQL** | Parameterized queries only (`pgx`/`database/sql`) |
| **Secrets** | Env vars only (`.env*` gitignored), no secrets in code |

---

## 10. Deployment

### 10.1 Docker (Multi-stage)

```dockerfile
# Build stage (Go)
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 go build -o server ./cmd/server

# Runtime
FROM alpine:3.20
COPY --from=builder /app/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
```

### 10.2 Vercel Configuration

```json
// vercel.json (root)
{
  "buildCommand": "cd frontend && npm run build",
  "outputDirectory": "frontend/dist",
  "framework": "vite",
  "functions": {
    "backend/cmd/server/main.go": { "maxDuration": 30 }
  }
}
```

### 10.3 Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | Postgres connection string |
| `JWT_ACCESS_SECRET` | Yes | ≥32 chars |
| `JWT_REFRESH_SECRET` | Yes | ≥32 chars |
| `JWT_ACCESS_TTL` | No | Default `15m` |
| `JWT_REFRESH_TTL` | No | Default `720h` |
| `USER_HASH_SALT` | Yes | Salt for anonymous `user_hash` |
| `CORS_ALLOWED_ORIGINS` | No | Comma-separated, default `http://localhost:5173` |
| `LOG_LEVEL` | No | `debug`/`info`/`warn`/`error` |
| `LOG_FORMAT` | No | `console`/`json` |
| Email providers | No | `RESEND_API_KEY`, `SENDGRID_API_KEY`, `BREVO_API_KEY` |

---

## 11. Testing Strategy

| Layer | Tool | Location |
|-------|------|----------|
| Unit (Go) | `testing` + `testify` + `sqlmock` | `*_test.go` next to source |
| Unit (Frontend) | Vitest + React Testing Library | `src/test/` |
| Integration | `go test -tags=integration` | Requires test DB |
| E2E | (Planned) Playwright | — |

**Run tests**:
```bash
# Backend
cd backend && go test ./...

# Frontend
cd frontend && npm test
```

---

## 12. Development Workflow

```bash
# 1. Start infrastructure
docker-compose up -d postgres

# 2. Backend (hot reload via air or manual)
cd backend && go run ./cmd/server

# 3. Frontend
cd frontend && npm run dev  # http://localhost:5173

# 4. Run migrations
cd backend && go run ./cmd/migrator
```

---

## 13. Code Conventions

| Aspect | Convention |
|--------|------------|
| Go packages | Lowercase, singular (`service/operations`, not `services`) |
| Go errors | Sentinel errors (`ErrNotFound`), wrap with `fmt.Errorf("%w: ...", err)` |
| Money | Always `domain.Money` (scale=2), never `float64` |
| Frontend money | Strings over API → `decimal.js` in UI |
| React components | PascalCase, `export default` for pages, named exports for UI |
| API paths | `/api/v1/<resource>` plural, RESTful |
| Idempotency | Client-generated `calc_id` (UUID) on all creates |

---

## 14. References

- `BUSINESS_LOGIC.md` — Functional specification (features Ф.1–Ф.13)
- `MATH_FORMULAS.md` — Mathematical formulas with Kopnova/CBR references
- `PRIVACY_RULES.md` — PII masking rules, data retention
- `CLAUDE.md` — Development guardrails (determinism, no float, etc.)
- `PROGRESS.md` — Task tracker (features Ф.1–Ф.13 status)
- `DESIGN_REFERENCES.md` — UI/UX references (T-Bank, premium dark neon)

*Generated from codebase analysis — FinHelper v1.0*