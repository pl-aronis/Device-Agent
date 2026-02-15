-- Migration 002: Drop unused tables
-- Removes tables that are not currently being used by the MDM server

-- Drop users table (no authentication implemented yet)
DROP TABLE IF EXISTS users;

-- Drop device_certificates table (SCEP certs not tracked in DB)
DROP TABLE IF EXISTS device_certificates;

-- Drop profiles table (profiles generated dynamically)
DROP TABLE IF EXISTS profiles;

-- Drop DEP/ABM related tables (not implemented)
DROP TABLE IF EXISTS dep_devices;

DROP TABLE IF EXISTS dep_tokens;

-- Drop audit_logs table (no audit logging implemented)
DROP TABLE IF EXISTS audit_logs;

-- Drop unused indexes
DROP INDEX IF EXISTS idx_dep_devices_serial;

DROP INDEX IF EXISTS idx_audit_logs_tenant;

DROP INDEX IF EXISTS idx_audit_logs_created;