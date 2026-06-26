-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS inventory;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP SCHEMA IF EXISTS inventory CASCADE;
-- +goose StatementEnd
