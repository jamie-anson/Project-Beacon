-- Drop transparency log migration
DROP VIEW IF EXISTS public_transparency_log;
DROP TRIGGER IF EXISTS trigger_set_transparency_log_fields ON transparency_log;
DROP FUNCTION IF EXISTS set_transparency_log_fields();
DROP SEQUENCE IF EXISTS transparency_log_index_seq;
DROP TABLE IF EXISTS transparency_log;
