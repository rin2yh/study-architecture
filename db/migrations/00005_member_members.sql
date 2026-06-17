-- +goose Up
CREATE TABLE member.members (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email        text NOT NULL UNIQUE,
    display_name text NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE member.members;
