-- Drop secrets table and indexes
DROP INDEX IF EXISTS idx_secrets_user_id_name;
DROP INDEX IF EXISTS idx_secrets_user_id_deleted_at;
DROP INDEX IF EXISTS idx_secrets_user_id;
DROP TABLE IF EXISTS secrets;

