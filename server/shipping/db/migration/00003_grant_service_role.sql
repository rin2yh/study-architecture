-- +goose Up
-- +goose StatementBegin
-- initdb (infra/db/ops/initdb) を通らない経路ではロール未作成となるため (ADR-[[202606231000]])。
DO $$
BEGIN
  IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'shipping_svc') THEN
    GRANT USAGE ON SCHEMA shipping TO shipping_svc;
    GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA shipping TO shipping_svc;
    GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA shipping TO shipping_svc;
    ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA shipping
      GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO shipping_svc;
    ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA shipping
      GRANT USAGE, SELECT ON SEQUENCES TO shipping_svc;
  END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'shipping_svc') THEN
    ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA shipping
      REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM shipping_svc;
    ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA shipping
      REVOKE USAGE, SELECT ON SEQUENCES FROM shipping_svc;
    REVOKE ALL ON ALL SEQUENCES IN SCHEMA shipping FROM shipping_svc;
    REVOKE ALL ON ALL TABLES IN SCHEMA shipping FROM shipping_svc;
    REVOKE USAGE ON SCHEMA shipping FROM shipping_svc;
  END IF;
END $$;
-- +goose StatementEnd
