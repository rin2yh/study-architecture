-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS shipping;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP SCHEMA IF EXISTS shipping CASCADE;
-- +goose StatementEnd
