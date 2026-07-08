-- FinHelper — migration 0003: goal contributions journal (BUSINESS_LOGIC.md ф.5)
--
-- Journal of ad-hoc top-ups to a savings goal. The goals table itself (target,
-- baseline current_amount, monthly_contribution, target_date, expected_yield)
-- already exists in 0001 — this migration adds only the event log.
--
-- Idempotency (BUSINESS_LOGIC ф.1 style): contribution_id is client-generated
-- and UNIQUE per (user_id, goal_id). A replayed POST returns the original row
-- instead of duplicating it. This mirrors operations.calc_id exactly.
--
-- Hybrid current model (design spec §3.2): the effective amount saved is
--   goals.current_amount (baseline) + Σ goal_contributions.amount
-- Read at projection time; never cached — self-healing on every recompute.

CREATE TABLE goal_contributions (
    id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id           BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    goal_id           BIGINT NOT NULL REFERENCES goals (id) ON DELETE CASCADE,
    -- Client-generated; the idempotency key (style of operations.calc_id).
    contribution_id   TEXT NOT NULL,
    amount            NUMERIC(28, 2) NOT NULL CHECK (amount > 0 AND amount = ROUND(amount, 2)),
    contribution_date DATE NOT NULL,
    comment           TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, goal_id, contribution_id)
);

CREATE INDEX goal_contributions_goal_idx ON goal_contributions (goal_id);

CREATE TRIGGER goal_contributions_touch BEFORE UPDATE ON goal_contributions
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
