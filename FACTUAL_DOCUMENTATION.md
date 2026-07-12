# FinHelper — Factual Documentation (Reference Data)

## 1. Mathematical Constants & Formulas

### 1.1 Time Value of Money (TVM)
| Formula | Expression | Variables |
|---------|------------|-----------|
| Compound Interest | `A = P(1 + r/n)^(nt)` | P=principal, r=annual rate, n=compounding/year, t=years |
| Simple Interest | `A = P(1 + rt)` | P=principal, r=annual rate, t=years |
| Effective Annual Rate | `EAR = (1 + r/n)^n - 1` | r=nominal rate, n=periods/year |
| Real Rate (Fisher) | `r_real = (1 + r_nom)/(1 + π) - 1` | r_nom=effective rate, π=inflation |
| Annuity Payment | `PMT = P * r(1+r)^n / ((1+r)^n - 1)` | P=principal, r=period rate, n=periods |
| Future Value (Ordinary Annuity) | `FV = PMT * ((1+r)^n - 1) / r` | PMT=payment, r=period rate, n=periods |
| Present Value (Ordinary Annuity) | `PV = PMT * (1 - (1+r)^-n) / r` | PMT=payment, r=period rate, n=periods |

### 1.2 Credit Mathematics (CBR 5750-У / Kopnova Ch.4.2)
| Concept | Formula | Notes |
|---------|---------|-------|
| Annuity Payment | `A = S * i(1+i)^n / ((1+i)^n - 1)` | S=principal, i=monthly rate, n=months |
| Differentiated Principal | `P_k = S / n` | Constant principal per month |
| Differentiated Interest | `I_k = (S - (k-1)*P_k) * i` | On remaining balance |
| ПСК (PSK) | IRR of cashflows: `CF_0 = S - Σfees_upfront`, `CF_k = -(payment_k + fee_monthly)` | Solved via Brent's method (float64 only) |
| Early Repayment (shorten term) | Re-amortize: `n_new` from `PV = PMT * (1 - (1+i)^-n_new)/i` | Same payment, fewer periods |
| Early Repayment (lower payment) | `PMT_new = PV * i / (1 - (1+i)^-n_remaining)` | Same term, lower payment |

### 1.3 Deposit Mathematics (Kopnova Ch.1 / CBR)
| Capitalization | Formula |
|----------------|---------|
| Monthly | `A = P(1 + r/12)^(12t)` |
| Quarterly | `A = P(1 + r/4)^(4t)` |
| Annually | `A = P(1 + r)^t` |
| Maturity (Simple) | `A = P(1 + rt)` |

### 1.4 Goals / Sinking Fund (Kopnova Ch.3.3.3)
| Target | Formula |
|--------|---------|
| Required Monthly (given n) | `PMT = (FV - PV(1+i)^n) * i / ((1+i)^n - 1)` |
| Months to Target (given PMT) | `n = log(1 + (FV - PV(1+i)^n) * i / PMT) / log(1+i)` — solved iteratively |

### 1.5 Russian Deposit Tax (ФЗ-382 от 26.12.2023)
| Year | Key Rate (Jan 1) | Non-taxable Limit | Tax Rate |
|------|------------------|-------------------|----------|
| 2024 | 16% | 1,000,000 × 0.16 = 160,000 ₽ | 13% (residents) |
| 2025 | 18%* | 1,000,000 × 0.18 = 180,000 ₽ | 13% / 15% (progressive) |
| 2026 | TBD | 1,000,000 × key_rate | TBD |

*Projected — actual set by CBR annually.

**Tax Calculation**: `tax = max(0, total_interest - non_taxable_limit) × 0.13`

---

## 2. Russian Financial Regulations Reference

### 2.1 Key Regulatory Documents
| Document | Subject | Relevance |
|----------|---------|-----------|
| **ФЗ-353** (21.12.2013) | Consumer lending | ПСК calculation, disclosure |
| **Указание ЦБ 5750-У** (2020) | ПСК methodology | IRR cashflow definition |
| **ФЗ-382** (26.12.2023) | Deposit income tax | Non-taxable limit = 1M × key_rate |
| **НК РФ ст. 214.7** | Personal income tax on deposits | Tax base, rates, reporting |
| **ФЗ-102** (16.07.1998) | Mortgage (pledge) | Mortgage vs rent comparison basis |
| **Копнова С.Б.** | Financial Mathematics textbook | Formulas for annuity, sinking fund, TVM |

### 2.2 CBR Key Rate History (for tax calculations)
| Date | Key Rate |
|------|----------|
| 2023-01-01 | 7.5% |
| 2023-07-28 | 8.5% |
| 2023-09-15 | 12.0% |
| 2023-10-27 | 13.0% |
| 2023-12-15 | 16.0% |
| 2024-02-16 | 16.0% |
| 2024-04-26 | 16.0% |
| 2024-06-07 | 18.0%* |

*Current as of 2024 — verify at cbr.ru

---

## 3. Data Format Specifications

### 3.1 API Money Format
```json
// ALL monetary values as STRINGS with exactly 2 decimal places
{
  "amount": "123456.78",
  "balance": "0.00",
  "limit_amount": "50000.00"
}
```
**Never** send as JSON numbers — precision loss in transit.

### 3.2 Date/Time Formats
| Context | Format | Example |
|---------|--------|---------|
| Date only (operation_date, target_date) | `YYYY-MM-DD` | `2026-07-13` |
| Timestamp (created_at, updated_at) | `RFC3339` | `2026-07-13T15:04:05Z` |
| Period (dashboard) | `month` / `quarter` / `year` / `custom` | `month` |

### 3.3 Enumerated Values

**Operation Types** (`operation_type`):
```
income          // доход
expense         // расход
transfer        // внутренний перевод
currency_exchange // конвертация валюты
refund          // возврат
```

**Income Subtypes** (`income_subtype`):
```
salary          // зарплата
fee             // гонорар
gift            // подарок
investment      // инвестиции
loan_repayment  // возврат займа
```

**Account Types** (`account_type`):
```
cash            // наличные
bank            // банковский счёт
savings         // накопительный
investment      // инвестиционный (брокерский)
crypto          // криптокошелёк
debt            // кредитный/долг
```

**Budget Rollover Policies** (`rollover_policy`):
```
none            // непереносимый (сгорает)
unlimited       // неограниченный (накапливается бесконечно)
months_3        // до 3 месяцев
```

**Goal Statuses** (`status`):
```
on_track        // на пути
at_risk         // на грани (взнос >= 90% требуемого)
behind          // отстаём
achieved        // достигнута
no_deadline     // без дедлайна и без регулярного взноса
```

**Budget Statuses** (`status`):
```
ok              // в норме
at_risk         // под угрозой (прогноз > лимит)
over            // превышен
inactive        // бюджет отключен
```

**Capitalization Frequencies** (`capitalization`):
```
monthly         // ежемесячная
quarterly       // ежеквартальная
annually        // ежегодная
maturity        // в конце срока (простые %)
```

**Payment Types** (`payment_type`):
```
annuity         // аннуитетный
differentiated  // дифференцированный
```

**Early Repayment Modes** (`early.mode`):
```
shorten_term    // сократить срок
lower_payment   // уменьшить платёж
```

### 3.4 PII Masking Patterns (PRIVACY_RULES.md)
Applied at **write time** in `pkg/pii/mask.go`:

| Pattern | Replacement | Example |
|---------|-------------|---------|
| Russian phone | `[PHONE]` | `+7 999 123-45-67` → `[PHONE]` |
| Email | `[EMAIL]` | `user@example.com` → `[EMAIL]` |
| Bank card (16 digits) | `[CARD]` | `4276 1234 5678 9012` → `[CARD]` |
| Person name (2+ capitalized words) | `[PERSON]` | `Иван Петров` → `[PERSON]` |
| INN (10/12 digits) | `[INN]` | `7701234567` → `[INN]` |
| SNILS (11 digits) | `[SNILS]` | `123-456-789 00` → `[SNILS]` |
| Passport (series + number) | `[PASSPORT]` | `4510 123456` → `[PASSPORT]` |

**Rule**: Masking is **one-way** — original PII never stored.

---

## 4. Database Reference

### 4.1 Complete Schema (Migration 0001)
```sql
-- ENUMs
CREATE TYPE operation_type AS ENUM ('income','expense','transfer','currency_exchange','refund');
CREATE TYPE income_subtype AS ENUM ('salary','fee','gift','investment','loan_repayment');
CREATE TYPE account_type AS ENUM ('cash','bank','savings','investment','crypto','debt');

-- TABLES
users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    user_hash TEXT NOT NULL UNIQUE,  -- SHA-256(id || salt)
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,  -- SHA-256 of refresh token
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

accounts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    account_type account_type NOT NULL DEFAULT 'cash',
    currency CHAR(3) NOT NULL DEFAULT 'RUB',
    balance NUMERIC(28,2) NOT NULL DEFAULT 0 CHECK (balance = ROUND(balance,2)),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

categories (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    parent_id BIGINT REFERENCES categories(id) ON DELETE SET NULL,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (user_id, name)
);

operations (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    calc_id TEXT NOT NULL,  -- idempotency key
    operation_type operation_type NOT NULL,
    amount NUMERIC(28,2) NOT NULL CHECK (amount > 0 AND amount = ROUND(amount,2)),
    amount_dst NUMERIC(28,2) CHECK (amount_dst IS NULL OR (amount_dst > 0 AND amount_dst = ROUND(amount_dst,2))),
    currency CHAR(3) NOT NULL DEFAULT 'RUB',
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    account_dst_id BIGINT REFERENCES accounts(id) ON DELETE RESTRICT,
    category_id BIGINT REFERENCES categories(id) ON DELETE SET NULL,
    income_subtype income_subtype,
    counterparty TEXT,  -- PII-masked
    description TEXT,   -- PII-masked
    operation_date DATE NOT NULL,
    is_planned BOOLEAN NOT NULL DEFAULT FALSE,
    category_confidence NUMERIC(4,3) CHECK (category_confidence IS NULL OR (category_confidence >= 0 AND category_confidence <= 1)),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (user_id, calc_id)
);

goals (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    target_amount NUMERIC(28,2) NOT NULL CHECK (target_amount > 0),
    current_amount NUMERIC(28,2) NOT NULL DEFAULT 0 CHECK (current_amount >= 0),
    monthly_contribution NUMERIC(28,2) CHECK (monthly_contribution IS NULL OR monthly_contribution >= 0),
    target_date DATE,
    expected_yield NUMERIC(8,5) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

goal_contributions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    goal_id BIGINT REFERENCES goals(id) ON DELETE CASCADE,
    contribution_id TEXT NOT NULL,  -- client idempotency key
    amount NUMERIC(28,2) NOT NULL CHECK (amount > 0 AND amount = ROUND(amount,2)),
    contribution_date DATE NOT NULL,
    comment TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (user_id, goal_id, contribution_id)
);

budgets (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    category_id BIGINT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    limit_amount NUMERIC(28,2) NOT NULL CHECK (limit_amount > 0),
    rollover_policy TEXT NOT NULL DEFAULT 'none' CHECK (rollover_policy IN ('none','unlimited','months_3')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (user_id, category_id)
);

-- INDEXES (partial on deleted_at IS NULL)
CREATE INDEX users_user_hash_idx ON users (user_hash) WHERE deleted_at IS NULL;
CREATE INDEX refresh_tokens_user_idx ON refresh_tokens (user_id);
CREATE INDEX refresh_tokens_expires_idx ON refresh_tokens (expires_at) WHERE revoked_at IS NULL;
CREATE INDEX accounts_user_idx ON accounts (user_id) WHERE deleted_at IS NULL;
CREATE INDEX categories_user_idx ON categories (user_id) WHERE deleted_at IS NULL;
CREATE INDEX operations_user_date_idx ON operations (user_id, operation_date DESC) WHERE deleted_at IS NULL;
CREATE INDEX operations_user_type_idx ON operations (user_id, operation_type) WHERE deleted_at IS NULL;
CREATE INDEX operations_account_idx ON operations (account_id);
CREATE INDEX operations_category_idx ON operations (category_id);
CREATE INDEX goals_user_idx ON goals (user_id) WHERE deleted_at IS NULL;
CREATE INDEX budgets_user_idx ON budgets (user_id) WHERE deleted_at IS NULL;

-- TRIGGERS for updated_at
CREATE OR REPLACE FUNCTION touch_updated_at() RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_touch    BEFORE UPDATE ON users    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER accounts_touch BEFORE UPDATE ON accounts FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER operations_touch BEFORE UPDATE ON operations FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER goals_touch   BEFORE UPDATE ON goals    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER budgets_touch BEFORE UPDATE ON budgets  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
```

### 4.2 Common Queries

**Dashboard Aggregate (month)**:
```sql
SELECT
    SUM(CASE WHEN operation_type = 'income' THEN amount ELSE 0 END) AS income,
    SUM(CASE WHEN operation_type = 'expense' THEN amount ELSE 0 END) AS expense,
    SUM(CASE WHEN operation_type IN ('income','refund') THEN amount
             WHEN operation_type = 'expense' THEN -amount
             ELSE 0 END) AS net_cashflow
FROM operations
WHERE user_id = $1
  AND operation_date >= date_trunc('month', NOW())
  AND operation_date < date_trunc('month', NOW()) + INTERVAL '1 month'
  AND deleted_at IS NULL
  AND operation_type IN ('income','expense','refund');
```

**Budget Rollover (3 months)**:
```sql
WITH months AS (
    SELECT generate_series(
        date_trunc('month', NOW()) - INTERVAL '3 months',
        date_trunc('month', NOW()) - INTERVAL '1 month',
        INTERVAL '1 month'
    ) AS m_start
),
spent AS (
    SELECT
        m.m_start,
        COALESCE(SUM(o.amount), 0) AS spent
    FROM months m
    LEFT JOIN operations o
        ON o.user_id = $1
        AND o.category_id = $2
        AND o.operation_type = 'expense'
        AND o.operation_date >= m.m_start
        AND o.operation_date < m.m_start + INTERVAL '1 month'
        AND o.deleted_at IS NULL
    GROUP BY m.m_start
)
SELECT SUM(GREATEST(b.limit_amount - s.spent, 0)) AS rollover
FROM spent s
JOIN budgets b ON b.id = $3
WHERE s.spent < b.limit_amount;
```

---

## 5. Configuration Files Reference

### 5.1 Tax Rules YAML (`backend/configs/tax_rules_2024.yaml`)
```yaml
year: 2024
key_rate_jan1: 0.16          # 16%
non_taxable_base: 1000000    # 1M RUB
tax_rate: 0.13               # 13% for residents
progressive_threshold: 5000000  # 5M RUB (2025+)
progressive_rate: 0.15       # 15% above threshold (2025+)
```

### 5.2 Environment Variables (`.env.example`)
```bash
# Database
DATABASE_URL=postgres://user:pass@localhost:5432/finhelper?sslmode=disable

# JWT (generate with: openssl rand -base64 32)
JWT_ACCESS_SECRET=your-32-char-min-access-secret
JWT_REFRESH_SECRET=your-32-char-min-refresh-secret
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=720h

# User hashing
USER_HASH_SALT=random-salt-for-anonymous-user-hash

# HTTP
HTTP_ADDR=:8080
CORS_ALLOWED_ORIGINS=http://localhost:5173,https://finhelper.ru

# Logging
LOG_LEVEL=info
LOG_FORMAT=console

# Email (optional — at least one provider)
RESEND_API_KEY=
SENDGRID_API_KEY=
BREVO_API_KEY=
BREVO_SENDER_EMAIL=
FROM_EMAIL=onboarding@resend.dev
FROM_NAME=FinHelper
FRONTEND_URL=https://finhelper.ru
```

---

## 6. Error Code Reference

### 6.1 HTTP Status Mapping
| Service Error | HTTP Status | Problem `type` |
|---------------|-------------|----------------|
| `ErrInvalidArgument` | 400 | `validation_error` |
| `ErrNotFound` | 404 | `not_found` |
| `ErrUnauthorized` / JWT errors | 401 | `unauthorized` |
| `ErrForbidden` | 403 | `forbidden` |
| `ErrConflict` (idempotency) | 409 | `conflict` |
| `ErrRateLimited` | 429 | `rate_limited` |
| Internal / DB errors | 500 | `internal` |

### 6.2 Sentinel Errors (Go)
```go
// pkg/auth
var (
    ErrTokenExpired = errors.New("auth: token expired")
    ErrTokenInvalid = errors.New("auth: token invalid")
    ErrWrongKind    = errors.New("auth: token kind mismatch")
)

// pkg/service/operations
var (
    ErrInvalidArgument = errors.New("operations: invalid argument")
    ErrNotFound        = errors.New("operations: not found")
    ErrAccountMissing  = errors.New("operations: account not found")
)

// pkg/service/budget
var (
    ErrInvalidArgument = errors.New("budget: invalid argument")
    ErrNotFound        = errors.New("budget: not found")
)

// pkg/service/goals
var (
    ErrInvalidArgument = errors.New("goals: invalid argument")
    ErrNotFound        = errors.New("goals: not found")
)

// pkg/storage
var (
    ErrOperationExists    = errors.New("storage: operation already exists (calc_id)")
    ErrOperationNotFound  = errors.New("storage: operation not found")
    ErrAccountNotFound    = errors.New("storage: account not found")
    ErrCategoryNotFound   = errors.New("storage: category not found")
    ErrBudgetExists       = errors.New("storage: budget exists")
    ErrBudgetNotFound     = errors.New("storage: budget not found")
    ErrGoalNotFound       = errors.New("storage: goal not found")
    ErrContributionExists = errors.New("storage: contribution exists")
    ErrContributionNotFound = errors.New("storage: contribution not found")
    ErrUserNotFound       = errors.New("storage: user not found")
)
```

---

## 7. Keyboard Shortcuts (Frontend)

| Key | Action | Context |
|-----|--------|---------|
| `g` | Go to Goals | Global |
| `o` | Go to Operations | Global |
| `b` | Go to Budgets | Global |
| `d` | Go to Dashboard | Global |
| `n` | New Operation | Operations / Dashboard |
| `h` | Home (Dashboard) | Global |
| `s` | Settings | Global |
| `1` | Dashboard | Global |
| `2` | Operations | Global |
| `3` | Goals | Global |
| `4` | Budgets | Global |
| `?` | Show shortcuts help | Global |

---

## 8. Public Datasets (for testing/benchmarking)

> Use `@public-datasets` reference to access these.

| Dataset | Source | Relevance |
|---------|--------|-----------|
| **CBR Key Rate History** | cbr.ru | Tax non-taxable limit calculation |
| **MOEX Bond/Stock Data** | moex.com | Investment account tracking (future) |
| **Rosstat Inflation** | rosstat.gov.ru | Real return (Fisher) calculations |
| **Banki.ru Deposit Rates** | banki.ru | Deposit calculator benchmarks |
| **Banki.ru Credit Rates** | banki.ru | Credit calculator benchmarks |
| **Russian Household Budget Survey** | HSE/Rosstat | Category defaults, budget templates |

---

## 9. Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2026-07-13 | Initial release: Ф.1–Ф.9 complete, Ф.10 partial, Ф.11–Ф.13 planned |

---

## 10. Quick Reference Card

### 10.1 Money Handling Rules
```
✅ ALWAYS: domain.Money (Go) / decimal.js (TS)
✅ ALWAYS: Strings in JSON ("1234.56")
✅ ALWAYS: Scale = 2 (kopecks)
✅ ALWAYS: ROUND_HALF_AWAY_FROM_ZERO
❌ NEVER: float64 / number for money
❌ NEVER: JSON numbers for money
```

### 10.2 Idempotency Pattern
```typescript
// Frontend: generate UUID v4 for every create
const calc_id = crypto.randomUUID()
await api.createOperation({ ..., calc_id })

// Backend: UNIQUE(user_id, calc_id) → 409 on duplicate
// Service: catch 409 → fetch existing by calc_id → return it
```

### 10.3 PII Masking Checklist
- [ ] Counterparty masked before INSERT
- [ ] Description masked before INSERT
- [ ] Logs use `user_hash`, never email
- [ ] Exports exclude masked fields or keep masked

### 10.4 Calculator Precision Requirements
| Calculator | Precision | Method |
|------------|-----------|--------|
| Deposit | 2 decimals (kopecks) | `decimal.Decimal` throughout |
| Credit | 2 decimals + PSK (float64 IRR) | Decimal for schedule, Brent for PSK |
| Goals | 2 decimals | Decimal (sinking fund formulas) |
| Tax | 2 decimals | Decimal (ФЗ-382 formula) |

---

*Factual Documentation v1.0 — FinHelper*