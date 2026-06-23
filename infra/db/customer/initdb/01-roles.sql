-- ADR-[[202606231000]]
-- ロールはクラスタ全体のオブジェクトで schema migration の単位 (DB 内) に収まらないため、
-- ここ (DB 初期化) で先に用意する。schema への GRANT は各サービスの migration が ec で付与する。
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'order_svc') THEN
    CREATE ROLE order_svc LOGIN PASSWORD 'order_pass';
  END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'payment_svc') THEN
    CREATE ROLE payment_svc LOGIN PASSWORD 'payment_pass';
  END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'member_svc') THEN
    CREATE ROLE member_svc LOGIN PASSWORD 'member_pass';
  END IF;
END $$;
