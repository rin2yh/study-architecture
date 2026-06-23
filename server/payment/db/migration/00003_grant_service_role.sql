-- +goose Up
-- +goose StatementBegin
-- initdb (infra/db/customer/initdb) を通らない経路ではロール未作成となるため (ADR-[[202606231000]])。
DO $$
BEGIN
  IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'payment_svc') THEN
    GRANT USAGE ON SCHEMA payment TO payment_svc;
    GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA payment TO payment_svc;
    GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA payment TO payment_svc;
    ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA payment
      GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO payment_svc;
    ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA payment
      GRANT USAGE, SELECT ON SEQUENCES TO payment_svc;
  END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'payment_svc') THEN
    ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA payment
      REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM payment_svc;
    ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA payment
      REVOKE USAGE, SELECT ON SEQUENCES FROM payment_svc;
    REVOKE ALL ON ALL SEQUENCES IN SCHEMA payment FROM payment_svc;
    REVOKE ALL ON ALL TABLES IN SCHEMA payment FROM payment_svc;
    REVOKE USAGE ON SCHEMA payment FROM payment_svc;
  END IF;
END $$;
-- +goose StatementEnd
