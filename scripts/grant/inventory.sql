-- ADR-[[202606231000]]
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'inventory_svc')   THEN CREATE ROLE inventory_svc   LOGIN PASSWORD 'inventory_pass';   END IF;
END $$;

GRANT USAGE ON SCHEMA inventory TO inventory_svc;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES    IN SCHEMA inventory TO inventory_svc;
GRANT USAGE, SELECT                  ON ALL SEQUENCES IN SCHEMA inventory TO inventory_svc;
ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA inventory GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES    TO inventory_svc;
ALTER DEFAULT PRIVILEGES FOR ROLE ec IN SCHEMA inventory GRANT USAGE, SELECT                  ON SEQUENCES TO inventory_svc;
