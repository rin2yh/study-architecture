-- +goose Up
-- +goose StatementBegin
-- initdb (infra/db/customer/initdb) を通らない経路ではロール未作成となるため (ADR-[[202606231000]])。
DO $$
BEGIN
  IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'member_svc') THEN
    GRANT USAGE ON SCHEMA member TO member_svc;
    GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA member TO member_svc;
    GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA member TO member_svc;
    ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA member
      GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO member_svc;
    ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA member
      GRANT USAGE, SELECT ON SEQUENCES TO member_svc;
  END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'member_svc') THEN
    ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA member
      REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM member_svc;
    ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA member
      REVOKE USAGE, SELECT ON SEQUENCES FROM member_svc;
    REVOKE ALL ON ALL SEQUENCES IN SCHEMA member FROM member_svc;
    REVOKE ALL ON ALL TABLES IN SCHEMA member FROM member_svc;
    REVOKE USAGE ON SCHEMA member FROM member_svc;
  END IF;
END $$;
-- +goose StatementEnd
