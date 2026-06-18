-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS member;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP SCHEMA IF EXISTS member CASCADE;
-- +goose StatementEnd
