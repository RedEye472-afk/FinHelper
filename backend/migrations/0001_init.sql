-- FinHelper — initial schema (migration 0001)
-- Target: PostgreSQL 16
-- Convention: all monetary columns are NUMERIC(28,2) — NEVER float/double.
-- Identifiers: BIGINT IDENTITY (sorted-friendly, no UUID fragmentation).
-- Soft delete pattern: deleted_at TIMESTAMPTZ NULL.

-- ============================================================================
-- ENUMS
-- ============================================================================
CREATE TYPE operation_type AS ENUM (
    'income',          -- доход
    'expense',         -- расход
    'transfer',        -- внутренний перевод (между счетами)
    'currency_exchange', -- конвертация валюты
    'refund'           -- возврат
);

CREATE TYPE income_subtype AS ENUM (
    'salary',    -- зарплата
    'fee',       -- гонорар
    'gift',      -- подарок
    'investment',-- инвестиции
    'loan_repayment' -- возврат займа
);

CREATE TYPE account_type AS ENUM (
    'cash',        -- наличные
    'bank',        -- банковский счёт
    'savings',     -- накопительный
    'investment',  -- инвестиционный (брокерский)
    'crypto',      -- криптокошелёк
    'debt'         -- кредитный/долг
);

-- ============================================================================
-- users
-- ============================================================================
CREATE TABLE users (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email           TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,                       -- bcrypt hash
    user_hash       TEXT NOT NULL UNIQUE,                -- SHA-256(id || salt), for logs/analytics
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ                          -- soft delete
);

CREATE INDEX users_user_hash_idx ON users (user_hash) WHERE deleted_at IS NULL;

-- ============================================================================
-- refresh_tokens — for JWT refresh rotation + revocation
-- ============================================================================
CREATE TABLE refresh_tokens (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash      TEXT NOT NULL UNIQUE,                -- SHA-256 of refresh token
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ,                         -- set on logout/rotation
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX refresh_tokens_user_idx ON refresh_tokens (user_id);
CREATE INDEX refresh_tokens_expires_idx ON refresh_tokens (expires_at) WHERE revoked_at IS NULL;

-- ============================================================================
-- accounts — user's financial accounts (cash, bank, savings, ...)
-- ============================================================================
CREATE TABLE accounts (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    account_type    account_type NOT NULL DEFAULT 'cash',
    currency        CHAR(3) NOT NULL DEFAULT 'RUB',
    -- Balance is derived from operations, but cached for dashboard performance.
    -- Recomputed on every operation insert (trigger or service-layer).
    balance         NUMERIC(28, 2) NOT NULL DEFAULT 0 CHECK (balance = ROUND(balance, 2)),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX accounts_user_idx ON accounts (user_id) WHERE deleted_at IS NULL;

-- ============================================================================
-- categories — spending categories (food, transport, etc.)
-- ============================================================================
CREATE TABLE categories (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    parent_id       BIGINT REFERENCES categories (id) ON DELETE SET NULL,
    -- For user-typed categories vs system defaults
    is_system       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    UNIQUE (user_id, name)
);

CREATE INDEX categories_user_idx ON categories (user_id) WHERE deleted_at IS NULL;

-- ============================================================================
-- operations — financial transactions
-- ============================================================================
CREATE TABLE operations (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- Idempotency: client-generated calc_id prevents duplicate writes
    calc_id         TEXT NOT NULL,
    operation_type  operation_type NOT NULL,
    -- Amount is always positive; sign is derived from operation_type.
    -- For transfers/exchanges: this is the source-side amount.
    amount          NUMERIC(28, 2) NOT NULL CHECK (amount > 0 AND amount = ROUND(amount, 2)),
    -- For currency_exchange: destination amount
    amount_dst      NUMERIC(28, 2) CHECK (amount_dst IS NULL OR (amount_dst > 0 AND amount_dst = ROUND(amount_dst, 2))),
    currency        CHAR(3) NOT NULL DEFAULT 'RUB',
    -- Source account (always), destination account (for transfers/exchanges)
    account_id      BIGINT NOT NULL REFERENCES accounts (id) ON DELETE RESTRICT,
    account_dst_id  BIGINT REFERENCES accounts (id) ON DELETE RESTRICT,
    -- Categorisation
    category_id     BIGINT REFERENCES categories (id) ON DELETE SET NULL,
    -- For income operations, the subtype affects analytics/taxes
    income_subtype  income_subtype,
    -- Counterparty (merchant, payer). PII-masked at write (PRIVACY_RULES.md §"Маскирование PII").
    -- e.g. "Перевод от [PERSON]"
    counterparty    TEXT,
    description     TEXT,
    operation_date  DATE NOT NULL,
    -- Planned = dated in future, marked "плановая" (BUSINESS_LOGIC ф.1 edge-case)
    is_planned      BOOLEAN NOT NULL DEFAULT FALSE,
    -- Confidence of auto-categorisation (0..1). NULL if user-set manually.
    category_confidence NUMERIC(4, 3) CHECK (category_confidence IS NULL OR (category_confidence >= 0 AND category_confidence <= 1)),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    -- Idempotency: one (user_id, calc_id) pair → one row
    UNIQUE (user_id, calc_id)
);

CREATE INDEX operations_user_date_idx ON operations (user_id, operation_date DESC) WHERE deleted_at IS NULL;
CREATE INDEX operations_user_type_idx ON operations (user_id, operation_type) WHERE deleted_at IS NULL;
CREATE INDEX operations_account_idx ON operations (account_id);
CREATE INDEX operations_category_idx ON operations (category_id);

-- ============================================================================
-- goals — savings targets (ф.5)
-- ============================================================================
CREATE TABLE goals (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id             BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    target_amount       NUMERIC(28, 2) NOT NULL CHECK (target_amount > 0),
    current_amount      NUMERIC(28, 2) NOT NULL DEFAULT 0 CHECK (current_amount >= 0),
    -- Monthly contribution (annuity-style). NULL = no regular contribution.
    monthly_contribution NUMERIC(28, 2) CHECK (monthly_contribution IS NULL OR monthly_contribution >= 0),
    target_date         DATE,
    expected_yield      NUMERIC(8, 5) NOT NULL DEFAULT 0,  -- expected annual yield, e.g. 0.08
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
);

CREATE INDEX goals_user_idx ON goals (user_id) WHERE deleted_at IS NULL;

-- ============================================================================
-- budgets — per-category monthly limits with rollover (ф.4)
-- ============================================================================
CREATE TABLE budgets (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    category_id     BIGINT NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
    -- Monthly limit (always positive)
    limit_amount    NUMERIC(28, 2) NOT NULL CHECK (limit_amount > 0),
    -- Rollover policy: 'none' = unused amount expires, 'unlimited' = accumulate indefinitely,
    -- 'months_3' = accumulate up to 3 months
    rollover_policy TEXT NOT NULL DEFAULT 'none' CHECK (rollover_policy IN ('none', 'unlimited', 'months_3')),
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    UNIQUE (user_id, category_id)
);

CREATE INDEX budgets_user_idx ON budgets (user_id) WHERE deleted_at IS NULL;

-- ============================================================================
-- updated_at trigger — keeps updated_at in sync on any UPDATE
-- ============================================================================
CREATE OR REPLACE FUNCTION touch_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_touch    BEFORE UPDATE ON users    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER accounts_touch BEFORE UPDATE ON accounts FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER operations_touch BEFORE UPDATE ON operations FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER goals_touch   BEFORE UPDATE ON goals    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER budgets_touch BEFORE UPDATE ON budgets  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
