// Package schemas implements the job payload schema registry: the component
// that decides what a job of a given kind and version is allowed to look like,
// and answers that question the same way on every replica.
//
// It exists because a payload is just bytes on the queue. During a rolling
// deploy the producer and the consumer of a job are different builds, and an
// external service enqueueing over HTTP is not our build at all, so nothing in
// the type system connects what was sent to what will be read. A registered
// schema is that connection, and it is durable rather than compiled in so that
// a replica can validate against a contract it does not itself know.
//
// Schemas reach the registry two ways and both end in the same table. The ones
// this binary ships with are generated from the Go argument structs in
// pkg/domain, embedded in the catalog subpackage, and registered at boot. The
// rest arrive over the HTTP API. Reads never consult the embedded copy: the
// database is the single answer, so a schema someone published at runtime is
// not a second-class citizen.
package schemas

import (
	"context"
	"encoding/json"

	"dockzilla/pkg/domain"
)

// Handler is the registry surface the transports call. It is the port the HTTP
// handler and, once payload validation is wired into the job engine, the
// enqueue and dispatch paths depend on.
type Handler interface {
	// Register publishes document as the schema for ref. Registering a
	// reference that already exists is a no-op when the document is the same
	// one and fails with ErrSchemaImmutable when it is not, so a published
	// version can never be rewritten under a replica already using it.
	Register(
		ctx context.Context,
		ref domain.SchemaRef,
		document json.RawMessage,
	) (domain.Schema, error)

	// Retrieve returns the schema registered for ref, or ErrSchemaNotFound.
	Retrieve(ctx context.Context, ref domain.SchemaRef) (domain.Schema, error)

	// Latest returns the most recently registered version of kind, or
	// ErrSchemaNotFound when the kind has no schema at all.
	Latest(ctx context.Context, kind domain.Kind) (domain.Schema, error)

	// List returns every registered schema, newest first. A zero kind lists
	// every kind. Documents are omitted: callers that want one ask for it by
	// reference.
	List(ctx context.Context, kind domain.Kind) ([]domain.Schema, error)

	// Validate reports whether payload satisfies the schema ref names. Both a
	// payload that fails validation and a reference nobody registered are
	// returned as terminal errors, because neither becomes true on a retry.
	Validate(ctx context.Context, ref domain.SchemaRef, payload domain.JobsPayload) error
}
