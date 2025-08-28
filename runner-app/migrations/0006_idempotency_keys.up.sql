-- 0006_idempotency_keys.up.sql
-- Create table to map idempotency keys to jobspec IDs.
-- Ensures duplicate POSTs with the same key return the same job without re-enqueuing.

CREATE TABLE IF NOT EXISTS idempotency_keys (
    id SERIAL PRIMARY KEY,
    idem_key TEXT NOT NULL,
    jobspec_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Uniqueness on key ensures only one job per idempotency key
CREATE UNIQUE INDEX IF NOT EXISTS uq_idempotency_keys_key ON idempotency_keys (idem_key);

-- Helpful index to look up by jobspec_id (optional but useful for cleanup/audits)
CREATE INDEX IF NOT EXISTS idx_idempotency_keys_job ON idempotency_keys (jobspec_id);
