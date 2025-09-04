-- 0006_idempotency_keys.down.sql
DROP INDEX IF EXISTS idx_idempotency_keys_job;
DROP INDEX IF EXISTS uq_idempotency_keys_key;
DROP TABLE IF EXISTS idempotency_keys;
