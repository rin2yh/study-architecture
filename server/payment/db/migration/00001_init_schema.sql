-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS payment;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP SCHEMA IF EXISTS payment CASCADE;
-- +goose StatementEnd
