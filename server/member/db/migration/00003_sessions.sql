-- +goose Up
-- ADR 0009
ALTER TABLE member.members ADD COLUMN password_hash text NOT NULL DEFAULT '';

-- ADR 0009
CREATE TABLE member.sessions (
    id         text PRIMARY KEY,
    member_id  bigint NOT NULL REFERENCES member.members (id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX sessions_member_id_idx ON member.sessions (member_id);

-- +goose Down
DROP TABLE member.sessions;
ALTER TABLE member.members DROP COLUMN password_hash;
