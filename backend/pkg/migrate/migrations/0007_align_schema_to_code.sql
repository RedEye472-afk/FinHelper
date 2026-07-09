-- 0007_align_schema_to_code.sql
-- Fix: production Neon schema columns don't match Go code expectations.
-- The partial 0001 migration created simplified columns. This migration
-- renames/adds columns to match what the Go storage layer expects.

-- operations: rename type → operation_type
ALTER TABLE operations RENAME COLUMN "type" TO operation_type;

-- operations: add missing columns from 0001_init.sql
ALTER TABLE operations ADD COLUMN IF NOT EXISTS amount_dst NUMERIC(28,2) CHECK (amount_dst IS NULL OR (amount_dst > 0 AND amount_dst = ROUND(amount_dst, 2)));
ALTER TABLE operations ADD COLUMN IF NOT EXISTS account_dst_id BIGINT;
ALTER TABLE operations ADD COLUMN IF NOT EXISTS counterparty TEXT;
ALTER TABLE operations ADD COLUMN IF NOT EXISTS is_planned BOOLEAN NOT NULL DEFAULT FALSE;

-- budgets: rename rollover → rollover_policy
ALTER TABLE budgets RENAME COLUMN rollover TO rollover_policy;

-- budgets: add missing columns
ALTER TABLE budgets ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;

-- goals: rename annual_yield → expected_yield
ALTER TABLE goals RENAME COLUMN annual_yield TO expected_yield;

-- goal_contributions: rename note → comment
ALTER TABLE goal_contributions RENAME COLUMN note TO comment;

-- budgets: fix rollover_policy type (was BOOLEAN, should be TEXT enum)
ALTER TABLE budgets DROP COLUMN IF EXISTS rollover_policy;
ALTER TABLE budgets ADD COLUMN rollover_policy TEXT NOT NULL DEFAULT 'none'
    CHECK (rollover_policy IN ('none', 'unlimited', 'months_3'));

-- budgets: make extra columns nullable (Go code doesn't use these)
ALTER TABLE budgets ALTER COLUMN period DROP NOT NULL;
ALTER TABLE budgets ALTER COLUMN start_date DROP NOT NULL;
ALTER TABLE budgets ALTER COLUMN end_date DROP NOT NULL;
