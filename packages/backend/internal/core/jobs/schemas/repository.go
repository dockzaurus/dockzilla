package schemas

import (
	"context"

	"dockzilla/pkg/domain"
)

// Repository is the durable store of registered schemas. It is the authority:
// every read the use case serves comes from here or from a cache in front of
// it, never from the binary's own embedded catalog.
type Repository interface {
	// Register inserts schema and returns the row that is actually stored,
	// which is the pre-existing one when the reference was already registered.
	// Implementations MUST NOT overwrite an existing row — the use case
	// compares what came back against what it sent to detect a rewrite, and an
	// implementation that upserts would hide exactly the drift being looked
	// for.
	Register(ctx context.Context, schema domain.Schema) (domain.Schema, error)

	// Get returns the schema stored for ref, or ErrSchemaNotFound.
	Get(ctx context.Context, ref domain.SchemaRef) (domain.Schema, error)

	// Latest returns the most recently registered version of kind, or
	// ErrSchemaNotFound.
	Latest(ctx context.Context, kind domain.Kind) (domain.Schema, error)

	// List returns the stored schemas without their documents, newest first. A
	// zero kind lists every kind.
	List(ctx context.Context, kind domain.Kind) ([]domain.Schema, error)
}

// CacheRepository is an optional read-through cache in front of Repository.
//
// It is safe for it to fail: the use case logs a cache error and falls through
// to the repository, so a cache outage costs latency rather than correctness.
// It holds documents rather than compiled schemas because it is shared between
// replicas and a compiled schema cannot be serialised; the compiled form is
// kept in process by the use case.
type CacheRepository interface {
	// Get returns the cached schema for ref, or ErrSchemaNotFound on a miss.
	Get(ctx context.Context, ref domain.SchemaRef) (domain.Schema, error)

	// Put stores schema. Entries never need invalidating: a published version
	// is immutable, so a cached document cannot go stale.
	Put(ctx context.Context, schema domain.Schema) error
}
