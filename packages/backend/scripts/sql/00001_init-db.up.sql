CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

  email VARCHAR(255) NOT NULL,
  display_name VARCHAR(255),

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX users_email_lower_idx ON users (lower(email));

CREATE TABLE IF NOT EXISTS identities (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
  
  user_id UUID NOT NULL REFERENCES users(identifier) ON DELETE CASCADE,
  provider VARCHAR(255) NOT NULL,
  provider_subject VARCHAR(255) NOT NULL, 
  password_hash VARCHAR(255) NOT NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  UNIQUE (provider, provider_subject),

  CONSTRAINT local_identity_has_password check (
    (provider = 'local' AND password_hash IS NOT NULL)
    or
    (provider <> 'local' AND password_hash IS NULL)
  )
); 

CREATE TABLE IF NOT EXISTS sessions (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

  user_id UUID NOT NULL REFERENCES users(identifier) ON DELETE CASCADE,
  token_hash BYTEA NOT NULL UNIQUE,
  user_agent VARCHAR(255),
  client_ip INET,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ
);

CREATE INDEX sessions_user_active_idx ON sessions (user_id) WHERE revoked_at is NULL;
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

CREATE TABLE IF NOT EXISTS api_tokens (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

  user_id UUID NOT NULL REFERENCES users(identifier) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  token_hash BYTEA NOT NULL UNIQUE,
  token_prefix VARCHAR(255) NOT NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  last_used_at TIMESTAMPTZ,
  expires_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ
);

CREATE INDEX api_tokens_user_active_idx ON
  api_tokens (user_id) WHERE revoked_at is NULL;

CREATE TABLE IF NOT EXISTS setup_claims (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
  token_hash BYTEA NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,
  consumed_by_user_id UUID REFERENCES users(identifier) ON DELETE SET NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS auth_events (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

  user_id UUID REFERENCES users(identifier) ON DELETE SET NULL,
  kind VARCHAR(127) NOT NULL,
  client_ip INET,
  request_id VARCHAR(255),
  detail JSONB,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

create INDEX auth_events_user_id_idx ON auth_events (user_id, id desc);
CREATE INDEX auth_events_created_at_idx ON auth_events (created_at);
---

CREATE TABLE IF NOT EXISTS hosts (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS registry_credentials (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS apps (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS deployments (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS app_env_vars (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS app_volumes (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS jobs (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
