-- Run after the collections migration's down file, which drops the triggers
-- that use set_updated_at().
DROP TABLE IF EXISTS rate_limits;
DROP FUNCTION IF EXISTS doi_fold(text);
DROP FUNCTION IF EXISTS set_updated_at();
