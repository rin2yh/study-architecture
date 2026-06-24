#!/usr/bin/env bash
# サービスごとの最小権限ロールを作り、自 schema にのみ DML できるよう GRANT する (ADR-[[202606231000]])。
# migrate (table 作成) 済みのスタックに対して流す。冪等なので何度流してもよい。
set -euo pipefail

cd "$(dirname "$0")/.."

# psql は host に無い場合があるので postgres コンテナ内のものを使う。
apply() { docker compose exec -T "$1" psql -U ec -d "$2" -v ON_ERROR_STOP=1; }

apply db-customer ec_customer <<'SQL'
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'order_svc')   THEN CREATE ROLE order_svc   LOGIN PASSWORD 'order_pass';   END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'payment_svc') THEN CREATE ROLE payment_svc LOGIN PASSWORD 'payment_pass'; END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'member_svc')  THEN CREATE ROLE member_svc  LOGIN PASSWORD 'member_pass';  END IF;
END $$;

GRANT USAGE ON SCHEMA "order" TO order_svc;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES    IN SCHEMA "order" TO order_svc;
GRANT USAGE, SELECT                  ON ALL SEQUENCES IN SCHEMA "order" TO order_svc;
ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA "order" GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES    TO order_svc;
ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA "order" GRANT USAGE, SELECT                  ON SEQUENCES TO order_svc;

GRANT USAGE ON SCHEMA payment TO payment_svc;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES    IN SCHEMA payment TO payment_svc;
GRANT USAGE, SELECT                  ON ALL SEQUENCES IN SCHEMA payment TO payment_svc;
ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA payment GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES    TO payment_svc;
ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA payment GRANT USAGE, SELECT                  ON SEQUENCES TO payment_svc;

GRANT USAGE ON SCHEMA member TO member_svc;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES    IN SCHEMA member TO member_svc;
GRANT USAGE, SELECT                  ON ALL SEQUENCES IN SCHEMA member TO member_svc;
ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA member GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES    TO member_svc;
ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA member GRANT USAGE, SELECT                  ON SEQUENCES TO member_svc;
SQL

apply db-ops ec_ops <<'SQL'
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'product_svc')  THEN CREATE ROLE product_svc  LOGIN PASSWORD 'product_pass';  END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'shipping_svc') THEN CREATE ROLE shipping_svc LOGIN PASSWORD 'shipping_pass'; END IF;
END $$;

GRANT USAGE ON SCHEMA product TO product_svc;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES    IN SCHEMA product TO product_svc;
GRANT USAGE, SELECT                  ON ALL SEQUENCES IN SCHEMA product TO product_svc;
ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA product GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES    TO product_svc;
ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA product GRANT USAGE, SELECT                  ON SEQUENCES TO product_svc;

GRANT USAGE ON SCHEMA shipping TO shipping_svc;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES    IN SCHEMA shipping TO shipping_svc;
GRANT USAGE, SELECT                  ON ALL SEQUENCES IN SCHEMA shipping TO shipping_svc;
ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA shipping GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES    TO shipping_svc;
ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA shipping GRANT USAGE, SELECT                  ON SEQUENCES TO shipping_svc;
SQL
