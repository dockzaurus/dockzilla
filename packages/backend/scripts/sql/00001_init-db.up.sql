CREATE EXTENSION IF NOT EXISTS pg_cron;

CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ---------------------------------------------------------------------------
-- 1b. Enum types
--
-- The database owns every closed set, so the allowed values are declared once
-- rather than repeated in a CHECK constraint per table. CREATE TYPE has no
-- IF NOT EXISTS, hence the to_regtype guard.
--
-- To add a value later:
--   ALTER TYPE deployment_status ADD VALUE IF NOT EXISTS 'cancelled';
--
-- Growing an enum is cheap. Removing or renaming a value is not, because there
-- is no DROP VALUE: it means creating a new type, moving the column, and
-- dropping the old one. So only genuinely closed sets belong here.
--
-- Deliberately NOT enums:
--   identities.provider  values are 'oidc:<name>' where <name> is chosen by
--                        the operator, so the set is not knowable here
--   auth_events.kind     grows with every auditable action
-- ---------------------------------------------------------------------------

DO $$
DECLARE
  e RECORD;
BEGIN
  FOR e IN
    SELECT * FROM (VALUES
      ('host_connection_type', ARRAY['local_socket', 'ssh']),
      ('host_status',          ARRAY['unknown', 'reachable', 'unreachable']),
      ('app_desired_state',    ARRAY['running', 'stopped']),
      ('deployment_status',    ARRAY['queued', 'pulling', 'starting', 'running',
                                     'failed', 'superseded']),
      ('deployment_trigger',   ARRAY['api', 'cli', 'webhook', 'rollback'])
    ) AS src(type_name, labels)
  LOOP
    IF to_regtype(e.type_name) IS NULL THEN
      EXECUTE format(
        'CREATE TYPE %I AS ENUM (%s)',
        e.type_name,
        (SELECT string_agg(quote_literal(l), ', ') FROM unnest(e.labels) AS l)
      );

      RAISE NOTICE 'created enum type %', e.type_name;
    END IF;
  END LOOP;
END $$;

-- ---------------------------------------------------------------------------
-- 2. Tables: auth block
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS users (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

  email VARCHAR(255) NOT NULL,
  display_name VARCHAR(255),

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS identities (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

  user_id UUID NOT NULL,
  provider VARCHAR(255) NOT NULL,
  provider_subject VARCHAR(255) NOT NULL,

  -- Nullable on purpose. An argon2id PHC string runs about 100 characters.
  -- A local identity must have one and any other provider must not, which the
  -- check below enforces. Making this NOT NULL would make every non-local
  -- provider impossible to insert.
  password_hash VARCHAR(255),

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  UNIQUE (provider, provider_subject),
  UNIQUE (user_id, provider),

  CONSTRAINT local_identity_has_password CHECK (
    (provider = 'local' AND password_hash IS NOT NULL)
    OR
    (provider <> 'local' AND password_hash IS NULL)
  )
);

CREATE TABLE IF NOT EXISTS sessions (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

  user_id UUID NOT NULL,
  token_hash BYTEA NOT NULL UNIQUE,
  user_agent VARCHAR(255),
  client_ip INET,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS api_tokens (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

  user_id UUID NOT NULL,
  name VARCHAR(255) NOT NULL,
  token_hash BYTEA NOT NULL UNIQUE,
  token_prefix VARCHAR(255) NOT NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  last_used_at TIMESTAMPTZ,
  expires_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS setup_claims (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

  token_hash BYTEA NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,
  consumed_by_user_id UUID,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS auth_events (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

  -- Nullable: a failed login for an unknown address has no user to point at.
  user_id UUID,
  -- Open set, grows with every auditable action, so no enum type.
  kind VARCHAR(127) NOT NULL,
  client_ip INET,
  request_id VARCHAR(255),
  detail JSONB,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------------
-- 2. Tables: core block
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS hosts (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

  name VARCHAR(255) NOT NULL UNIQUE,
  connection_type host_connection_type NOT NULL,

  socket_path VARCHAR(1023),

  ssh_host VARCHAR(255),
  ssh_port INTEGER,
  ssh_user VARCHAR(255),
  ssh_key_ref VARCHAR(255),

  status host_status NOT NULL DEFAULT 'unknown',
  last_seen_at TIMESTAMPTZ,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT hosts_connection_fields CHECK (
    (connection_type = 'local_socket' AND socket_path IS NOT NULL)
    OR
    (connection_type = 'ssh' AND ssh_host IS NOT NULL AND ssh_user IS NOT NULL)
  )
);

CREATE TABLE IF NOT EXISTS registry_credentials (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

  name VARCHAR(255) NOT NULL UNIQUE,
  registry_host VARCHAR(255) NOT NULL,
  username VARCHAR(255) NOT NULL,

  password_encrypted BYTEA NOT NULL,
  encryption_key_version INTEGER NOT NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS apps (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

  name VARCHAR(63) NOT NULL,
  host_id UUID NOT NULL,
  image_ref VARCHAR(1023) NOT NULL,
  registry_credential_id UUID,
  desired_state app_desired_state NOT NULL DEFAULT 'running',

  archived_at TIMESTAMPTZ,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  -- DNS label rules: the name becomes a container name and a subdomain.
  CONSTRAINT apps_name_is_dns_label CHECK (
    name ~ '^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$'
  )
);

CREATE TABLE IF NOT EXISTS deployments (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

  app_id UUID NOT NULL,
  image_ref VARCHAR(1023) NOT NULL,
  -- The immutable digest resolved at pull time. This is what rollback targets:
  -- a tag may have moved since, a digest cannot.
  image_digest VARCHAR(255),
  status deployment_status NOT NULL DEFAULT 'queued',
  container_id VARCHAR(255),

  triggered_by deployment_trigger NOT NULL,
  triggered_by_user_id UUID,

  error_code VARCHAR(127),
  error_message TEXT,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS app_env_vars (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

  app_id UUID NOT NULL,
  key VARCHAR(255) NOT NULL,
  -- Ciphertext from the secret engine. Never plaintext, never text.
  value_encrypted BYTEA NOT NULL,
  encryption_key_version INTEGER NOT NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  UNIQUE (app_id, key),

  CONSTRAINT app_env_vars_key_valid CHECK (key ~ '^[A-Za-z_][A-Za-z0-9_]*$')
);

CREATE TABLE IF NOT EXISTS app_volumes (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

  app_id UUID NOT NULL,
  name VARCHAR(255) NOT NULL,
  mount_path VARCHAR(1023) NOT NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  UNIQUE (app_id, mount_path)
);

-- ---------------------------------------------------------------------------
-- 3. Indexes
--
-- PostgreSQL does not index foreign key columns automatically, and an
-- unindexed one makes every cascading delete a sequential scan of the child
-- table.
-- ---------------------------------------------------------------------------

CREATE UNIQUE INDEX IF NOT EXISTS users_email_lower_idx ON users (lower(email));

CREATE INDEX IF NOT EXISTS identities_user_id_idx ON identities (user_id);

CREATE INDEX IF NOT EXISTS sessions_user_id_idx ON sessions (user_id);
CREATE INDEX IF NOT EXISTS sessions_user_active_idx
  ON sessions (user_id) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS sessions_expires_at_idx ON sessions (expires_at);

CREATE INDEX IF NOT EXISTS api_tokens_user_id_idx ON api_tokens (user_id);
CREATE INDEX IF NOT EXISTS api_tokens_user_active_idx
  ON api_tokens (user_id) WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS setup_claims_consumed_by_user_id_idx
  ON setup_claims (consumed_by_user_id);

CREATE INDEX IF NOT EXISTS auth_events_user_id_idx ON auth_events (user_id, id DESC);
CREATE INDEX IF NOT EXISTS auth_events_created_at_idx ON auth_events (created_at);

-- Frees an archived app's name for reuse while keeping active names unique.
CREATE UNIQUE INDEX IF NOT EXISTS apps_name_active_idx
  ON apps (name) WHERE archived_at IS NULL;
CREATE INDEX IF NOT EXISTS apps_host_id_idx ON apps (host_id);
CREATE INDEX IF NOT EXISTS apps_registry_credential_id_idx
  ON apps (registry_credential_id);

-- The keyset pagination path for the deployment ledger.
CREATE INDEX IF NOT EXISTS deployments_app_id_id_idx ON deployments (app_id, id DESC);
CREATE INDEX IF NOT EXISTS deployments_triggered_by_user_id_idx
  ON deployments (triggered_by_user_id);

CREATE INDEX IF NOT EXISTS app_env_vars_app_id_idx ON app_env_vars (app_id);
CREATE INDEX IF NOT EXISTS app_volumes_app_id_idx ON app_volumes (app_id);

-- ---------------------------------------------------------------------------
-- 4. Foreign keys
--
-- Added after every table exists, and skipped when already present. Every one
-- references identifier rather than id, so the domain can build a complete
-- object graph in memory before anything is written.
--
-- ON DELETE choices:
--   CASCADE  the child cannot outlive its parent (credentials, sessions)
--   SET NULL the child records history that must survive the parent
--   RESTRICT deleting the parent is a mistake we want to block
-- ---------------------------------------------------------------------------

DO $$
DECLARE
  fk RECORD;
  fk_name TEXT;
BEGIN
  FOR fk IN
    SELECT * FROM (VALUES
      ('identities',   'user_id',                'users',                'CASCADE'),
      ('sessions',     'user_id',                'users',                'CASCADE'),
      ('api_tokens',   'user_id',                'users',                'CASCADE'),
      ('setup_claims', 'consumed_by_user_id',    'users',                'SET NULL'),
      ('auth_events',  'user_id',                'users',                'SET NULL'),
      ('apps',         'host_id',                'hosts',                'RESTRICT'),
      ('apps',         'registry_credential_id', 'registry_credentials', 'SET NULL'),
      ('deployments',  'app_id',                 'apps',                 'RESTRICT'),
      ('deployments',  'triggered_by_user_id',   'users',                'SET NULL'),
      ('app_env_vars', 'app_id',                 'apps',                 'CASCADE'),
      ('app_volumes',  'app_id',                 'apps',                 'CASCADE')
    ) AS t(child_table, child_column, parent_table, on_delete)
  LOOP
    fk_name := format('%s_%s_fkey', fk.child_table, fk.child_column);

    IF NOT EXISTS (
      SELECT 1
      FROM pg_constraint
      WHERE conname = fk_name
        AND conrelid = format('%I', fk.child_table)::regclass
    ) THEN
      EXECUTE format(
        'ALTER TABLE %I ADD CONSTRAINT %I
           FOREIGN KEY (%I) REFERENCES %I(identifier) ON DELETE %s',
        fk.child_table, fk_name, fk.child_column, fk.parent_table, fk.on_delete
      );

      RAISE NOTICE 'added foreign key %', fk_name;
    END IF;
  END LOOP;
END $$;

-- ---------------------------------------------------------------------------
-- 5. updated_at triggers
--
-- The trigger rather than application code, because the reconciler and the
-- health engine both write rows without passing through a handler.
--
-- DROP then CREATE, because CREATE OR REPLACE TRIGGER needs PostgreSQL 14 and
-- CREATE TRIGGER has no IF NOT EXISTS.
-- ---------------------------------------------------------------------------

DO $$
DECLARE
  tbl TEXT;
BEGIN
  FOR tbl IN
    SELECT unnest(ARRAY[
      'users',
      'identities',
      'sessions',
      'api_tokens',
      'hosts',
      'registry_credentials',
      'apps',
      'deployments',
      'app_env_vars',
      'app_volumes'
    ])
  LOOP
    EXECUTE format(
      'DROP TRIGGER IF EXISTS %I ON %I',
      tbl || '_set_updated_at', tbl
    );

    EXECUTE format(
      'CREATE TRIGGER %I BEFORE UPDATE ON %I
         FOR EACH ROW EXECUTE FUNCTION set_updated_at()',
      tbl || '_set_updated_at', tbl
    );
  END LOOP;
END $$;
