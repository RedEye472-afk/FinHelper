-- FinHelper — migration 0002: auto-categorization (BUSINESS_LOGIC.md ф.2)
--
-- Scope-lock (Plane from GLM.md): rules-only, NO ML. Two mechanisms:
--   1. categorization_rules  — keyword substring → category (Level 2, ~70%)
--   2. counterparty_overrides — exact counterparty → category with a
--      confirmation counter. At >= LEARN_THRESHOLD (3) corrections the row is
--      treated as a learned authoritative override (simplified Level 3 — a
--      counter stands in for the ML model; scope-lock keeps it deterministic).
--
-- MCC codes (Level 1, ~90%) are intentionally absent: MVP has no bank
-- integration, so there are no MCC values to match on.
--
-- Privacy note (PRIVACY_RULES.md): counterparty here is the already-PII-masked
-- value written by service/operations.Create. We never store raw names/phones.

-- ============================================================================
-- categorization_rules — keyword → category
-- ============================================================================
CREATE TABLE categorization_rules (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- Lowercased substring to match against operations.counterparty /
    -- operations.description (both PII-masked at write time).
    keyword         TEXT NOT NULL,
    category_id     BIGINT NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
    -- 'system' = seeded default at registration; 'user' = created/edited by user.
    source          TEXT NOT NULL DEFAULT 'system' CHECK (source IN ('system', 'user')),
    -- Higher priority is checked first. User rules outrank system rules so a
    -- manual correction wins over the default mapping.
    priority        INT NOT NULL DEFAULT 0,
    is_enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Soft delete, same pattern as categories/accounts for consistency.
    deleted_at      TIMESTAMPTZ,
    -- One keyword per user (case folded before insert).
    UNIQUE (user_id, keyword)
);

CREATE INDEX categorization_rules_user_idx
    ON categorization_rules (user_id, is_enabled, priority DESC)
    WHERE deleted_at IS NULL;

CREATE TRIGGER categorization_rules_touch
    BEFORE UPDATE ON categorization_rules
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- ============================================================================
-- counterparty_overrides — exact counterparty → category with confirmations
-- ============================================================================
CREATE TABLE counterparty_overrides (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- Normalized counterparty (lowercased + trimmed). Sourced from the
    -- PII-masked operations.counterparty, so no raw PII is ever stored here.
    counterparty    TEXT NOT NULL,
    category_id     BIGINT NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
    -- How many times the user confirmed this mapping. Starts at 1 on first
    -- manual correction; the service promotes the row to "learned" authority
    -- at >= LEARN_THRESHOLD (3). Pure counter, not ML.
    confirmations   INT NOT NULL DEFAULT 1 CHECK (confirmations >= 0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, counterparty)
);

CREATE INDEX counterparty_overrides_user_idx ON counterparty_overrides (user_id);

CREATE TRIGGER counterparty_overrides_touch
    BEFORE UPDATE ON counterparty_overrides
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
