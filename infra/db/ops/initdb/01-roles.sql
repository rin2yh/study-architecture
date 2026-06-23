-- ADR-[[202606231000]]
-- ロールはクラスタ全体のオブジェクトで schema migration の単位 (DB 内) に収まらないため、
-- ここ (DB 初期化) で先に用意する。schema への GRANT は各サービスの migration が ec で付与する。
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'product_svc') THEN
    CREATE ROLE product_svc LOGIN PASSWORD 'product_pass';
  END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'shipping_svc') THEN
    CREATE ROLE shipping_svc LOGIN PASSWORD 'shipping_pass';
  END IF;
END $$;
