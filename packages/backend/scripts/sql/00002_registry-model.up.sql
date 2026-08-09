CREATE TABLE IF NOT EXISTS schemas (
    id BIGINT GENERATED ALWAYS AS  IDENTITY PRIMARY KEY,
    identifier UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,

    kind VARCHAR(255) NOT NULL,
    version VARCHAR(127) NOT NULL,
    schema JSONB NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    archived_at TIMESTAMPTZ DEFAULT NULL,

    UNIQUE (kind, version)
);

CREATE INDEX jobs_schemas_kind_idx ON schemas (kind);
CREATE INDEX jobs_schema_uniques_idx ON schemas (kind, version);
CREATE INDEX jobs_schema_identifier_idx ON schemas (identifier);