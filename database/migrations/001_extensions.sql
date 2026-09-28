-- 001_extensions.sql
-- Enable required PostgreSQL extensions

CREATE EXTENSION IF NOT EXISTS postgis;
-- CREATE EXTENSION IF NOT EXISTS timescaledb;  -- Requires TimescaleDB installation
-- CREATE EXTENSION IF NOT EXISTS pgrouting;    -- Requires pgRouting installation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Verify extensions
SELECT extname, extversion FROM pg_extension WHERE extname IN ('postgis', 'uuid-ossp', 'pgcrypto');