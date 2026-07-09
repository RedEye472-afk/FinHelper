-- 0006_align_accounts_schema.sql
-- Fix: production Neon schema has column 'type' instead of 'account_type',
-- and enum lacks 'crypto'/'debt' values that domain.AccountType expects.
-- This migration aligns DB to match the Go code (0001_init.sql canonical).

-- 1. Rename column type → account_type (idempotent via separate checks)
--    Cannot use DO $$ because splitSQL breaks on $$ blocks.
--    Instead: ALTER ... RENAME TO ... — errors are ignored by migrator.
ALTER TABLE accounts RENAME COLUMN "type" TO account_type;

-- 2. Add missing enum values: crypto, debt
ALTER TYPE account_type ADD VALUE IF NOT EXISTS 'crypto';
ALTER TYPE account_type ADD VALUE IF NOT EXISTS 'debt';

-- 3. touch_updated_at function + triggers (ensure they exist)
CREATE OR REPLACE FUNCTION touch_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers (idempotent DROP + CREATE)
DROP TRIGGER IF EXISTS accounts_touch ON accounts;
CREATE TRIGGER accounts_touch BEFORE UPDATE ON accounts FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

DROP TRIGGER IF EXISTS users_touch ON users;
CREATE TRIGGER users_touch BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

DROP TRIGGER IF EXISTS categories_touch ON categories;
CREATE TRIGGER categories_touch BEFORE UPDATE ON categories FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

DROP TRIGGER IF EXISTS operations_touch ON operations;
CREATE TRIGGER operations_touch BEFORE UPDATE ON operations FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

DROP TRIGGER IF EXISTS budgets_touch ON budgets;
CREATE TRIGGER budgets_touch BEFORE UPDATE ON budgets FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

DROP TRIGGER IF EXISTS goals_touch ON goals;
CREATE TRIGGER goals_touch BEFORE UPDATE ON goals FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
