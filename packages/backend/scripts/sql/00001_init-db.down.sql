-- V0 initial schema, reversed.
--
-- Structure of this file, the up file read backwards:
--   1. tables
--   2. enum types
--   3. shared functions
--
-- Sections 3, 4 and 5 of the up file (indexes, foreign keys, updated_at
-- triggers) have no counterpart here: DROP TABLE takes all three with it.
--
-- Every statement carries IF EXISTS, so this file is re-runnable and needs
-- none of the pg_catalog guards the up file uses. DROP has the IF EXISTS that
-- CREATE TYPE, ADD CONSTRAINT and CREATE TRIGGER lack.
--
-- This destroys data. It exists for local development and for tests that want
-- a clean database, not for production rollback.

-- ---------------------------------------------------------------------------
-- 1. Tables
--
-- One statement, so the order inside the list does not matter: PostgreSQL
-- resolves the dependencies itself. CASCADE covers anything created outside
-- this schema that still points at these tables.
-- ---------------------------------------------------------------------------

DROP TABLE IF EXISTS
  app_volumes,
  app_env_vars,
  deployments,
  apps,
  registry_credentials,
  hosts,
  auth_events,
  setup_claims,
  api_tokens,
  sessions,
  identities,
  users
CASCADE;

-- ---------------------------------------------------------------------------
-- 2. Enum types
--
-- After the tables, because a type cannot be dropped while a column still uses
-- it. No CASCADE: if a column outside this schema still depends on one of
-- these, the drop should fail loudly rather than silently take the column.
-- ---------------------------------------------------------------------------

DROP TYPE IF EXISTS
  deployment_trigger,
  deployment_status,
  app_desired_state,
  host_status,
  host_connection_type;

-- ---------------------------------------------------------------------------
-- 3. Functions
--
-- The triggers went with their tables, so nothing references this any more.
-- ---------------------------------------------------------------------------

DROP FUNCTION IF EXISTS set_updated_at();
