-- +goose Up
-- +goose StatementBegin
-- ドメインごとに Postgres schema を分けて区画する（Step 0: 共有DB / schema 分離）。
-- "order" は SQL 予約語のため、schema 名としては常に二重引用符が必要。
CREATE SCHEMA IF NOT EXISTS product;
CREATE SCHEMA IF NOT EXISTS "order";
CREATE SCHEMA IF NOT EXISTS payment;
CREATE SCHEMA IF NOT EXISTS member;
CREATE SCHEMA IF NOT EXISTS shipping;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP SCHEMA IF EXISTS shipping CASCADE;
DROP SCHEMA IF EXISTS member CASCADE;
DROP SCHEMA IF EXISTS payment CASCADE;
DROP SCHEMA IF EXISTS "order" CASCADE;
DROP SCHEMA IF EXISTS product CASCADE;
-- +goose StatementEnd
