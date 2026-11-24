-- The first migration. It creates no table: the domain is not defined yet.
-- It proves that the migration pipeline runs on a fresh database.

-- +goose Up
COMMENT ON SCHEMA public IS 'lora platform';

-- +goose Down
COMMENT ON SCHEMA public IS 'standard public schema';
