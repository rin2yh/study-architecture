-- +goose Up
-- 住所帳 (複数住所) は会員が所有する (ADR-[[202606261704]] / ADR-[[202606231000]])。
CREATE TABLE member.addresses (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    member_id   bigint NOT NULL REFERENCES member.members (id) ON DELETE CASCADE,
    recipient   text NOT NULL,
    postal_code text NOT NULL,
    prefecture  text NOT NULL,
    city        text NOT NULL,
    line1       text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX addresses_member_id_idx ON member.addresses (member_id);

-- +goose Down
DROP TABLE member.addresses;
