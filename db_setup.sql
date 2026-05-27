-- One-time database setup — run as superuser on sparkdb BEFORE first deploy.
-- This file is safe to commit; it contains no real passwords.
-- Replace <migrator_password> and <app_password> with strong random strings
-- (store them in Cove as MARQUEE_MIGRATOR_DATABASE_URL / MARQUEE_APP_DATABASE_URL).

CREATE ROLE marquee_migrator LOGIN PASSWORD '<migrator_password>';
CREATE ROLE marquee_app      LOGIN PASSWORD '<app_password>';

CREATE DATABASE marquee OWNER marquee_migrator;

\c marquee

CREATE SCHEMA marquee AUTHORIZATION marquee_migrator;

REVOKE ALL ON DATABASE marquee FROM PUBLIC;
REVOKE CREATE ON SCHEMA public FROM PUBLIC;

GRANT CONNECT ON DATABASE marquee TO marquee_app;
GRANT USAGE  ON SCHEMA marquee TO marquee_app;

-- Existing objects (none at first run, but safe to have)
GRANT SELECT, INSERT, UPDATE, DELETE
  ON ALL TABLES IN SCHEMA marquee TO marquee_app;
GRANT USAGE, SELECT
  ON ALL SEQUENCES IN SCHEMA marquee TO marquee_app;

-- Future objects created by marquee_migrator
ALTER DEFAULT PRIVILEGES FOR ROLE marquee_migrator IN SCHEMA marquee
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO marquee_app;
ALTER DEFAULT PRIVILEGES FOR ROLE marquee_migrator IN SCHEMA marquee
  GRANT USAGE, SELECT ON SEQUENCES TO marquee_app;
ALTER DEFAULT PRIVILEGES FOR ROLE marquee_migrator IN SCHEMA marquee
  GRANT EXECUTE ON FUNCTIONS TO marquee_app;
