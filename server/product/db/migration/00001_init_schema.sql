-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS product;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP SCHEMA IF EXISTS product CASCADE;
-- +goose StatementEnd
