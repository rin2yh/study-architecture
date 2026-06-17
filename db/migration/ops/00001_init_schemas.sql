-- +goose Up
-- +goose StatementBegin
-- 運用系 (db-ops) のドメイン schema を作成 ([[ADR 0012]])。
CREATE SCHEMA IF NOT EXISTS product;
CREATE SCHEMA IF NOT EXISTS shipping;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP SCHEMA IF EXISTS shipping CASCADE;
DROP SCHEMA IF EXISTS product CASCADE;
-- +goose StatementEnd
