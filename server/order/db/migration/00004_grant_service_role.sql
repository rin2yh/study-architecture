-- +goose Up
-- +goose StatementBegin
-- initdb (infra/db/customer/initdb) を通らない経路ではロール未作成となるため (ADR-[[202606231000]])。
DO $$
BEGIN
  IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'order_svc') THEN
    GRANT USAGE ON SCHEMA "order" TO order_svc;
    GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA "order" TO order_svc;
    GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA "order" TO order_svc;
    ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA "order"
      GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO order_svc;
    ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA "order"
      GRANT USAGE, SELECT ON SEQUENCES TO order_svc;
  END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'order_svc') THEN
    ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA "order"
      REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM order_svc;
    ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA "order"
      REVOKE USAGE, SELECT ON SEQUENCES FROM order_svc;
    REVOKE ALL ON ALL SEQUENCES IN SCHEMA "order" FROM order_svc;
    REVOKE ALL ON ALL TABLES IN SCHEMA "order" FROM order_svc;
    REVOKE USAGE ON SCHEMA "order" FROM order_svc;
  END IF;
END $$;
-- +goose StatementEnd
