-- FinHelper — migration 0005: fix partial schema from async/race migrations
-- Target: PostgreSQL 16
--
-- PROBLEM (Vercel cold start, DO NOT REMOVE):
--   0001_init.sql was applied asynchronously with a ~30s timeout. On the first
--    cold start, only some statements executed before the connection was dropped.
--    Subsequent cold starts SKIP 0001 because its check
--    ('SELECT 1 FROM pg_tables WHERE tablename = 'users'') is too weak — the
--    'users' table exists even when many columns, triggers, and the
--    touch_updated_at() function are MISSING.
--
-- This migration uses ONLY idempotent constructs (CREATE OR REPLACE / IF NOT EXISTS)
-- so it is safe to apply any number of times.
--
-- Convention: all monetary columns are NUMERIC(28,2) — NEVER float/double.
-- Convention: all rows have updated_at with BEFORE UPDATE trigger.
-- See: progress.md §"Vercel partial schema (2026-07)"

-- ============================================================================
-- 1. touch_updated_at() — the core trigger function
-- ============================================================================
-- Used by EVERY table in the schema. If this is missing, ALL BEFORE UPDATE
-- triggers fail and registration/any write operation returns 500.
-- Source: 0001_init.sql §"updated_at trigger"
CREATE OR REPLACE FUNCTION touch_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- 2. Tables with missing columns
-- ============================================================================

-- 2a. categories — may lack deleted_at if 0001 applied partially
-- Source: 0001_init.sql lines 87-97
ALTER TABLE categories ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- 2b. operations — may lack deleted_at, income_subtype, category_confidence
ALTER TABLE operations ADD COLUMN IF NOT EXISTS deleted_at    TIMESTAMPTZ;
ALTER TABLE operations ADD COLUMN IF NOT EXISTS income_subtype income_subtype;
ALTER TABLE operations ADD COLUMN IF NOT EXISTS category_confidence NUMERIC(4, 3)
    CHECK (category_confidence IS NULL OR (category_confidence >= 0 AND category_confidence <= 1));

-- 2c. accounts — may lack deleted_at
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- 2d. budgets — may lack deleted_at
ALTER TABLE budgets ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- 2e. goals — may lack deleted_at
ALTER TABLE goals ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- 2f. refresh_tokens — may lack revoked_at
ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMPTZ;

-- 2g. categorization_rules — may lack deleted_at, updated_at, is_enabled
ALTER TABLE categorization_rules ADD COLUMN IF NOT EXISTS deleted_at  TIMESTAMPTZ;
ALTER TABLE categorization_rules ADD COLUMN IF NOT EXISTS updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE categorization_rules ADD COLUMN IF NOT EXISTS is_enabled  BOOLEAN NOT NULL DEFAULT TRUE;

-- 2h. counterparty_overrides — may lack updated_at
ALTER TABLE counterparty_overrides ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- 2i. goal_contributions — may lack
ALTER TABLE goal_contributions ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- ============================================================================
-- 3. Missing triggers (function must exist first — guaranteed by step 1)
-- ============================================================================
-- All tables use BEFORE UPDATE triggers named {tablename}_touch.
-- These are idempotent via DROP...CREATE pattern.

-- Users
DROP TRIGGER IF EXISTS users_touch ON users;
CREATE TRIGGER users_touch BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- Accounts
DROP TRIGGER IF EXISTS accounts_touch ON accounts;
CREATE TRIGGER accounts_touch BEFORE UPDATE ON accounts FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- Categories
DROP TRIGGER IF EXISTS categories_touch ON categories;
CREATE TRIGGER categories_touch BEFORE UPDATE ON categories FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- Operations
DROP TRIGGER IF EXISTS operations_touch ON operations;
CREATE TRIGGER operations_touch BEFORE UPDATE ON operations FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- Goals
DROP TRIGGER IF EXISTS goals_touch ON goals;
CREATE TRIGGER goals_touch BEFORE UPDATE ON goals FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- Budgets
DROP TRIGGER IF EXISTS budgets_touch ON budgets;
CREATE TRIGGER budgets_touch BEFORE UPDATE ON budgets FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- Categorization rules (from 0002)
DROP TRIGGER IF EXISTS categorization_rules_touch ON categorization_rules;
CREATE TRIGGER categorization_rules_touch BEFORE UPDATE ON categorization_rules
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- Counterparty overrides (from 0002)
DROP TRIGGER IF EXISTS counterparty_overrides_touch ON counterparty_overrides;
CREATE TRIGGER counterparty_overrides_touch BEFORE UPDATE ON counterparty_overrides
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- Goal contributions (from 0003)
DROP TRIGGER IF EXISTS goal_contributions_touch ON goal_contributions;
CREATE TRIGGER goal_contributions_touch BEFORE UPDATE ON goal_contributions
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- ============================================================================
-- 4. Missing indexes
-- ============================================================================
-- These are "IF NOT EXISTS" so they don't error on re-apply.

-- Categories
CREATE INDEX IF NOT EXISTS categories_user_idx ON categories (user_id) WHERE deleted_at IS NULL;

-- Operations indexes
CREATE INDEX IF NOT EXISTS operations_user_date_idx ON operations (user_id, operation_date DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS operations_user_type_idx ON operations (user_id, operation_type) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS operations_account_idx ON operations (account_id);
CREATE INDEX IF NOT EXISTS operations_category_idx ON operations (category_id);

-- Accounts
CREATE INDEX IF NOT EXISTS accounts_user_idx ON accounts (user_id) WHERE deleted_at IS NULL;

-- Goals
CREATE INDEX IF NOT EXISTS goals_user_idx ON goals (user_id) WHERE deleted_at IS NULL;

-- Budgets
CREATE INDEX IF NOT EXISTS budgets_user_idx ON budgets (user_id) WHERE deleted_at IS NULL;

-- Refresh tokens
CREATE INDEX IF NOT EXISTS refresh_tokens_user_idx ON refresh_tokens (user_id);
CREATE INDEX IF NOT EXISTS refresh_tokens_expires_idx ON refresh_tokens (expires_at) WHERE revoked_at IS NULL;

-- Users
CREATE INDEX IF NOT EXISTS users_user_hash_idx ON users (user_hash) WHERE deleted_at IS NULL;

-- ============================================================================
-- 5. Tighten the 0001 check function — validate full schema
-- ============================================================================
-- Helper function: returns TRUE if the full 0001 schema is complete.
-- Used by migrate.go to detect partial application.
CREATE OR REPLACE FUNCTION fn_full_0001_schema() RETURNS BOOLEAN AS $$
BEGIN
    -- Check all 6 core tables exist
    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'users') THEN RETURN FALSE; END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'accounts') THEN RETURN FALSE; END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'categories') THEN RETURN FALSE; END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'operations') THEN RETURN FALSE; END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'budgets') THEN RETURN FALSE; END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'goals') THEN RETURN FALSE; END IF;

    -- Check the trigger function exists (the most common partial-schema failure)
    IF NOT EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'touch_updated_at') THEN RETURN FALSE; END IF;

    -- Check key columns that are often missing
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
        WHERE table_name = 'categories' AND column_name = 'deleted_at') THEN RETURN FALSE; END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
        WHERE table_name = 'operations' AND column_name = 'deleted_at') THEN RETURN FALSE; END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
        WHERE table_name = 'accounts' AND column_name = 'deleted_at') THEN RETURN FALSE; END IF;

    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;
