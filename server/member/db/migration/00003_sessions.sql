-- +goose Up
-- password_hash は既存行も持てるよう DEFAULT '' で追加する。空文字は bcrypt 比較が必ず
-- 失敗するため「ログイン不可の会員」を表し、移行時にダミーパスワードを撒かずに済む。
ALTER TABLE member.members ADD COLUMN password_hash text NOT NULL DEFAULT '';

-- id は不透明トークンの SHA-256 (hex)。生トークンは Cookie だけが持ち、DB 流出時にも
-- 既存セッションを乗っ取られないようハッシュで保管する (ADR 0009)。
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
