# FinHelper — Project Documentation

## 1. Project Overview

**FinHelper** is a personal finance management application designed for Russian-speaking users. It combines **transaction tracking**, **budgeting with rollover**, **savings goals**, and **financial calculators** (deposit, credit, affordability, mortgage vs rent) in a single privacy-first application.

### 1.1 Vision
> "A T-Bank quality personal finance app with bank-grade calculation accuracy, zero PII leakage, and a premium dark-neon UX."

### 1.2 Target Audience
- Russian residents managing personal/household finances
- Users who need accurate credit/deposit calculations (ПСК, effective rates, tax)
- Privacy-conscious users (no telemetry, PII masked at write)
- Power users wanting keyboard shortcuts, bulk import, what-if simulations

### 1.3 Key Differentiators
| Aspect | FinHelper Approach |
|--------|-------------------|
| **Math accuracy** | Pure `decimal.Decimal` (Go) + `decimal.js` (TS), zero `float64` for money |
| **Calculators** | CBR 5750-У compliant ПСК, Kopnova sinking fund, ФЗ-382 tax |
| **Privacy** | PII masked before persistence (`[PERSON]`, `[CARD]`, `[PHONE]`, `[EMAIL]`) |
| **Architecture** | Service-layer pattern, stateless services, interface-based repos |
| **UX** | Premium dark neon, Framer Motion animations, keyboard shortcuts |
| **Offline-first** | Demo mode without auth, localStorage settings |

---

## 2. Feature Scope (BUSINESS_LOGIC.md Features Ф.1–Ф.13)

| ID | Feature | Status | Description |
|----|---------|--------|-------------|
| **Ф.1** | Operations & Accounts | ✅ Done | Manual entry, transfers, currency exchange, refunds; idempotent creates; balance recompute |
| **Ф.2** | Auto-categorization | ✅ Done | Keyword/regex rules per user; confidence scores; PII-safe (sees masked text only) |
| **Ф.3** | Dashboard | ✅ Done | Net worth, month income/expense, savings rate, category breakdown, goals progress |
| **Ф.4** | Budgets with Rollover | ✅ Done | Per-category monthly limits; policies: `none` / `unlimited` / `months_3`; forecast status |
| **Ф.5** | Savings Goals | ✅ Done | Target + deadline + recurring contribution; hybrid current amount; projection + what-if |
| **Ф.6** | Deposit Calculator | ✅ Done | Compound/simple interest, capitalization freq, effective rate, real return (Fisher), ФЗ-382 tax |
| **Ф.7** | Credit Calculator | ✅ Done | Annuity/differentiated, ПСК (CBR 5750-У), commissions/insurance, early repayment scenarios |
| **Ф.8** | Affordability Calculator | ✅ Done | Max loan by income/DTI, stress test at rate+2%, ПСК comparison |
| **Ф.9** | Mortgage vs Rent | ✅ Done | Frontend-only NPV comparison: buy (mortgage + costs) vs rent + invest difference |
|| **Ф.10** | Bank Import (PDF/CSV) | 🟡 Partial | Sberbank PDF parser; CSV column mapping; dedup by `calc_id`; TanStack Virtual for 4000+ ops preview |
|| **Ф.11** | Recurring Operations | ⏳ Planned | Template → auto-create on date; salary, subscriptions, rent |
|| **Ф.12** | What-if Simulator | ⏳ Planned | Monte Carlo on goals/credit; parameter sweeps |
|| **Ф.13** | Family/Shared Budgets | ⏳ Planned | Multi-user households, shared categories, permissions |

---

## 3. Product Requirements

### 3.1 Functional Requirements

#### FR-1: Transaction Management
- Create income/expense/transfer/exchange/refund operations
- Idempotent creates via client-generated `calc_id` (UUID v4)
- Auto-categorization on create (optional, confidence 0–1)
- Soft delete with balance recomputation
- Pagination (cursor-based, 50 default, max 200)

#### FR-2: Accounts & Balances
- Account types: cash, bank, savings, investment, crypto, debt
- Cached balance on account row, recomputed from operation history on every write
- Multi-currency schema ready (MVP: RUB only)

#### FR-3: Categories
- User-defined + system defaults
- Hierarchical (parent_id)
- Color/icon metadata (frontend only)

#### FR-4: Dashboard
- Period selector: month / quarter / year / custom
- KPI cards: net worth, income, expense, savings rate
- Category breakdown (expenses only, cashflow types)
- Net worth: assets (cash/bank/savings/investment) − debts (debt/credit)
- Goals progress preview
- Recent operations (10)

#### FR-5: Budgets
- One budget per category per user
- Rollover policies: `none` (expire), `unlimited` (accumulate all), `months_3` (last 3 months)
- Status computation: `ok` / `at_risk` / `over` / `inactive`
- Forecast: project current spend rate to period end

#### FR-6: Goals
- Target amount, optional deadline, optional monthly contribution, expected yield
- Hybrid current amount: `baseline + Σ contributions`
- Projection: months left, required monthly, estimated months at current rate
- Status: `on_track` / `at_risk` / `behind` / `achieved` / `no_deadline`
- What-if simulation (override any parameter)

#### FR-7: Deposit Calculator
- Inputs: principal, annual rate, term (months), capitalization (monthly/quarterly/annually/maturity), inflation (optional), tax year (optional)
- Outputs: maturity amount, total interest, effective rate, real return, tax amount, month-by-month projection

#### FR-8: Credit Calculator
- Inputs: principal, annual rate, term (months), payment type (annuity/diff), upfront fees[], monthly fee, early repayment scenario
- Outputs: monthly payment (first for diff), ПСК, overpayment, full schedule, early repayment summary ("savings X, term reduced Y months")

#### FR-9: Affordability Calculator
- Inputs: monthly income, existing monthly payments, target DTI (default 50%), loan rate, term
- Outputs: max principal, monthly payment, ПСК, stress test at rate+2%

#### FR-10: Mortgage vs Rent
- Buy side: property price, down payment, mortgage rate, term, maintenance %, property tax, appreciation, transaction costs
- Rent side: monthly rent, rent growth, investment return on down payment + diff
- Output: NPV comparison, breakeven year, sensitivity sliders

### 3.2 Non-Functional Requirements

| Category | Requirement |
|----------|-------------|
| **Accuracy** | All monetary math in `decimal.Decimal` (Go) / `decimal.js` (TS); scale=2; ROUND_HALF_AWAY_FROM_ZERO |
| **Performance** | Dashboard < 300ms p95; calculator < 100ms; list pagination capped at 200 |
| **Availability** | Stateless services; `/healthz` + `/readyz`; DB optional for boot |
| **Security** | bcrypt cost 12; JWT HS256 separate secrets; refresh token hashed in DB; CORS allowlist; rate limit auth |
| **Privacy** | No analytics, no telemetry; PII masked at write; email only for auth; user_hash for logs |
| **Accessibility** | WCAG AA contrast; keyboard navigation; focus management; ARIA labels |
| **Internationalization** | RU primary; currency formatter locale-aware; date RU locale |
| **Browser Support** | Last 2 Chrome/Firefox/Safari/Edge; no IE |

---

## 4. User Flows

### 4.1 Onboarding (New User)
```
Landing → Register → Verify Email (6-digit code) → Set up first Account → Create first Operation → Dashboard
```
- Demo mode available without registration (localStorage only)

### 4.2 Daily Tracking
```
Dashboard → Operations → New Operation (type, amount, account, category, date) → Save → Dashboard updates
```
- Keyboard shortcuts: `n` = new operation, `o` = operations list

### 4.3 Budget Setup
```
Budgets → Create Budget (category, limit, rollover policy) → Dashboard shows status chips
```
- Monthly rollover computed automatically

### 4.4 Goal Planning
```
Goals → Create Goal (name, target, deadline, monthly, yield) → Add Contributions → Projection updates
```
- What-if: "What if I increase monthly to 50k?"

### 4.5 Credit Decision
```
Calculators → Credit → Fill params → See ПСК, schedule, early repayment scenarios
```
- Compare multiple banks by editing rate/fees

### 4.6 Bank Statement Import
```
Import → Select PDF/CSV → Map columns → Preview → Confirm → Dedup by calc_id → Operations created
```

---

## 5. UI/UX Specification

### 5.1 Design System (Premium Dark Neon)
| Token | Value | Usage |
|-------|-------|-------|
| `--bg-page` | `#0B1020` | Page background |
| `--bg-surface` | `#141A2D` | Cards, modals |
| `--bg-card` | `#1E293B` | Elevated surfaces |
| `--primary` | `#6E56CF` | Primary actions, accents |
| `--primary-glow` | `rgba(110,86,207,0.3)` | Focus rings, hover glows |
| `--success` | `#22C55E` | Income, positive |
| `--danger` | `#F43F5E` | Expense, negative |
| `--warning` | `#F59E0B` | Alerts, at_risk |
| `--text-primary` | `#F8FAFC` | Headings |
| `--text-secondary` | `#94A3B8` | Body |
| `--text-tertiary` | `#64748B` | Muted |
| `--border-default` | `rgba(255,255,255,0.06)` | Card borders |
| `--radius-lg` | `20px` | Cards, modals |
| `--radius-md` | `12px` | Buttons, inputs |
| `--font-mono` | `ui-monospace, SFMono, Menlo` | Money display |

### 5.2 Typography
- **Headings**: Inter, 600–700 weight
- **Body**: Inter, 400 weight, 14px base
- **Money**: `font-mono-money` (tabular nums), 2 decimal places, space thousands separator, comma decimal, `₽` suffix

### 5.3 Animation
- Page transitions: `framer-motion` `animate-page-in` (opacity + y: 12→0, 0.3s easeOut)
- List items: stagger 0.035s
- Sheets/sheets: spring (damping 28, stiffness 300)
- Hover/tap: scale 1.02, background glow

### 5.4 Responsive Breakpoints
| Breakpoint | Layout |
|------------|--------|
| `< 640px` | Mobile: bottom nav, full-width cards, slide-up sheets |
| `640–1024px` | Tablet: collapsible sidebar, 2-col grids |
| `> 1024px` | Desktop: fixed 280px sidebar, 3-col dashboard grids |

### 5.5 Key Components
- **AppLayout**: Sidebar + Header + Main + MobileNav
- **KPICard**: Icon + label + value + trend
- **VirtualTable**: `@tanstack/react-virtual` for 1000+ ops
- **DataTable**: `@tanstack/react-table` with sorting, filtering, pagination
- **FormulaTooltip**: KaTeX-rendered formula on hover
- **AiExplanation**: Plain-language explanation of the math
- **Disclaimer**: Legal footer on every calculator
- **Toast**: Sonner notifications
- **FormField**: React Hook Form + Zod validation

---

## 6. Data Model Summary

```
User (1) ─────< (N) Account
User (1) ─────< (N) Category
User (1) ─────< (N) Operation
User (1) ─────< (N) Goal
User (1) ─────< (N) Budget
User (1) ─────< (N) GoalContribution
Goal (1) ─────< (N) GoalContribution
Budget (1) ──< (1) Category
Operation (N) ─< (1) Account (source)
Operation (N) ─< (1) Account (dest, for transfer/exchange)
Operation (N) ─< (1) Category
```

All tables: `BIGINT IDENTITY` PK, `deleted_at` soft delete, `created_at`/`updated_at` with trigger.

---

## 7. API Design Principles

| Principle | Implementation |
|-----------|----------------|
| **RESTful** | `/api/v1/<resource>` plural nouns |
| **Versioning** | URL prefix `/v1/` |
| **Idempotency** | Client `calc_id` on all creates |
| **Pagination** | Cursor (`before_id`), `more` flag |
| **Errors** | RFC 7807 Problem Details |
| **Money** | Strings in JSON (`"1234.56"`), never numbers |
| **Dates** | ISO 8601 (`"2026-07-13"`, `"2026-07-13T15:04:05Z"`) |
| **Auth** | Bearer access token; refresh via `/auth/refresh` |

---

## 8. Infrastructure & DevOps

### 8.1 Environments
| Env | Purpose | Database | Frontend URL |
|-----|---------|----------|--------------|
| `local` | Developer machine | Docker Postgres | `http://localhost:5173` |
| `preview` | PR deployments | Neon branch | `https://finhelper-git-<branch>.vercel.app` |
| `production` | Live | Supabase/Neon main | `https://finhelper.ru` |

### 8.2 CI/CD (GitHub Actions)
```yaml
# .github/workflows/ci.yml (conceptual)
jobs:
  backend-test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env: { POSTGRES_PASSWORD: test }
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5 { with: { go-version: '1.26' } }
      - run: cd backend && go test ./...
  
  frontend-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4 { with: { node-version: '20' } }
      - run: cd frontend && npm ci && npm test
  
  deploy-preview:
    needs: [backend-test, frontend-test]
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-latest
    steps:
      - uses: amondnet/vercel-action@v25
        with: { vercel-token: ${{ secrets.VERCEL_TOKEN }}, vercel-args: '--scope=team' }
```

### 8.3 Monitoring
- **Health**: `/healthz` (liveness), `/readyz` (readiness + DB ping)
- **Logs**: Structured JSON (production) / console (dev) via `pkg/log`
- **Errors**: Sentry (planned) — no PII in events

---

## 9. Security Checklist

- [x] bcrypt cost 12 for passwords
- [x] JWT: separate access/refresh secrets, HS256, short access TTL
- [x] Refresh tokens stored as SHA-256 hash only
- [x] CORS allowlist from env
- [x] Rate limiting on auth endpoints (10 req/min/IP)
- [x] Parameterized SQL everywhere
- [x] PII masking at write (`pkg/pii`)
- [x] No secrets in repo (`.env*` gitignored)
- [x] Security headers via Vercel (`_headers`)
- [ ] CSP header (planned)
- [ ] Dependency scanning (Dependabot/Trivy)
- [ ] Penetration test (planned pre-launch)

---

## 10. Roadmap

### v1.1 (Q3 2026)
- [ ] Ф.10: Full bank import (Tinkoff, Alfa, Raiffeisen PDF/CSV)
- [ ] Ф.11: Recurring operations (templates + scheduler)
- [ ] Export: CSV/Excel/PDF reports
- [ ] Webhooks for external integrations

### v1.2 (Q4 2026)
- [ ] Ф.12: What-if simulator (Monte Carlo on goals/credit)
- [ ] Ф.13: Family/shared budgets (invite by email, roles)
- [ ] PWA: offline mode, install prompt
- [ ] Multi-currency accounts (FX rates from CBR)

### v2.0 (2027)
- [ ] Investment tracking (broker API sync, MOEX)
- [ ] Tax declarations helper (НДФЛ, investment deductions)
- [ ] AI insights (spending anomalies, subscription detection)
- [ ] Public API for third-party integrations

---

## 11. Team & Roles

| Role | Responsibility |
|------|----------------|
| **Backend Engineer** | Go services, mathcore, DB, auth, deployment |
| **Frontend Engineer** | React/TS, UI components, charts, calculators |
| **DevOps** | Vercel, Docker, CI/CD, monitoring |
| **QA** | Test plans, E2E (Playwright), regression |
| **Product** | Prioritization, specs, user research |

---

## 12. Glossary

| Term | Definition |
|------|------------|
| **calc_id** | Client-generated UUID for idempotent creates |
| **user_hash** | SHA-256(user_id \|\| salt) — anonymous ID for logs/analytics |
| **ПСК (Полная стоимость кредита)** | APR per CBR 5750-У, includes all fees/insurance |
| **Rollover** | Carry-over of unused budget to next period |
| **Sinking fund** | Regular contributions accumulating with interest to hit target |
| **Effective rate** | `(1 + nominal/n)^n - 1` — true annual yield with compounding |
| **Real return** | Fisher equation: `(1+eff)/(1+inf) - 1` |
| **DTI** | Debt-to-Income ratio (monthly payments / gross income) |
| **NPV** | Net Present Value — discounted cashflow comparison |

---

*Project Documentation v1.0 — FinHelper*