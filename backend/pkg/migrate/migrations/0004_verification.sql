-- FinHelper — email verification & password reset (migration 0004)
-- Target: PostgreSQL 16
-- Adds columns for email verification (6-digit code) and password reset (UUID token)
-- to the users table. Both flows are optional — existing users without these
-- columns behave as before (verified = true by default for backfill).

-- ============================================================================
-- users — email verification columns
-- ============================================================================
ALTER TABLE users ADD COLUMN IF NOT EXISTS verified              BOOLEAN     NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS verification_code     TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS verification_expires  TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_reset_token  TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_reset_expires TIMESTAMPTZ;

-- Index for verification code lookups (relatively rare — full scan is fine,
-- but an index keeps the query fast when users do retry verification).
CREATE INDEX IF NOT EXISTS users_verification_code_idx ON users (verification_code)
    WHERE verification_code IS NOT NULL AND verified = FALSE;
CREATE INDEX IF NOT EXISTS users_password_reset_token_idx ON users (password_reset_token)
    WHERE password_reset_token IS NOT NULL;
