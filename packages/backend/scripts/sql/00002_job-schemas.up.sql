-- ---------------------------------------------------------------------------
-- Job payload schema registry.
--
-- One row per published payload contract. A row is written once and never
-- updated: (kind, version) is unique, registration is ON CONFLICT DO NOTHING,
-- and a caller that submits a different document for a version that already
-- exists is rejected rather than allowed to rewrite it. Replicas running
-- different builds during a rolling deploy therefore agree on what any given
-- version means.
--
-- The table is named job_schemas rather than schemas: "schema" already means
-- a namespace in PostgreSQL, and "registry" already means an image registry in
-- this product (see registry_credentials).
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS job_schemas (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

  -- kind matches a domain.Kind, version a domain.SchemaVersion ("v1").
  kind VARCHAR(255) NOT NULL,
  version VARCHAR(127) NOT NULL,
  -- document is the JSON Schema itself, stored and served verbatim.
  document JSONB NOT NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  -- archived_at retires a version without deleting it, so a job still on the
  -- queue that references it can be explained after the fact.
  archived_at TIMESTAMPTZ DEFAULT NULL,

  UNIQUE (kind, version)
);

-- No further indexes on purpose. UNIQUE (kind, version) already backs both the
-- lookup by reference and the lookup by kind alone, because kind is the
-- leading column, and identifier is already indexed by its UNIQUE constraint.

DROP TRIGGER IF EXISTS job_schemas_set_updated_at ON job_schemas;

CREATE TRIGGER job_schemas_set_updated_at
  BEFORE UPDATE ON job_schemas
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
