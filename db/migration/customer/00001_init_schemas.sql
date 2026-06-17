-- +goose Up
-- +goose StatementBegin
-- 顧客系 (db-customer) のドメイン schema を作成 ([[ADR 0012]])。
-- "order" は SQL 予約語のため schema 名としては常に二重引用符が必要。
CREATE SCHEMA IF NOT EXISTS "order";
CREATE SCHEMA IF NOT EXISTS payment;
CREATE SCHEMA IF NOT EXISTS member;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP SCHEMA IF EXISTS member CASCADE;
DROP SCHEMA IF EXISTS payment CASCADE;
DROP SCHEMA IF EXISTS "order" CASCADE;
-- +goose StatementEnd
