-- +goose Up
-- +goose StatementBegin
-- initdb (infra/db/ops/initdb) を通らない経路ではロール未作成となるため (ADR-[[202606231000]])。
DO $$
BEGIN
  IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'product_svc') THEN
    GRANT USAGE ON SCHEMA product TO product_svc;
    GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA product TO product_svc;
    GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA product TO product_svc;
    ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA product
      GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO product_svc;
    ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA product
      GRANT USAGE, SELECT ON SEQUENCES TO product_svc;
  END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'product_svc') THEN
    ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA product
      REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM product_svc;
    ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA product
      REVOKE USAGE, SELECT ON SEQUENCES FROM product_svc;
    REVOKE ALL ON ALL SEQUENCES IN SCHEMA product FROM product_svc;
    REVOKE ALL ON ALL TABLES IN SCHEMA product FROM product_svc;
    REVOKE USAGE ON SCHEMA product FROM product_svc;
  END IF;
END $$;
-- +goose StatementEnd
