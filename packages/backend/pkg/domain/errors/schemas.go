package errors

import "errors"

var (
	// ErrInvalidSchemaRef is returned when a schema reference is not in the
	// "<kind>/<version>" form.
	ErrInvalidSchemaRef = errors.New("schema registry: reference must be <kind>/<version>")

	// ErrSchemaNotFound is returned when no schema is registered for a
	// reference. During dispatch it is terminal: a job whose contract nobody
	// published cannot become valid by being retried.
	ErrSchemaNotFound = errors.New("schema registry: schema not found")

	// ErrSchemaInvalid is returned when a submitted document is not a JSON
	// Schema the validator can compile.
	ErrSchemaInvalid = errors.New("schema registry: document is not a valid json schema")

	// ErrSchemaImmutable is returned when a registration targets a version that
	// already exists with a different document. Rewriting a published version
	// would change what it means for replicas that already validated against
	// it, so the registry refuses and the caller must publish a new version.
	ErrSchemaImmutable = errors.New("schema registry: version already registered")

	// ErrPayloadInvalid is returned when a payload does not satisfy the schema
	// its reference names.
	ErrPayloadInvalid = errors.New("schema registry: payload does not satisfy its schema")
)
