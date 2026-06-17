-- +goose Up
-- +goose StatementBegin
-- "order" は SQL 予約語のため schema 名は常に二重引用符。
CREATE SCHEMA IF NOT EXISTS "order";
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP SCHEMA IF EXISTS "order" CASCADE;
-- +goose StatementEnd
