package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"dockzilla/internal/core/jobs/schemas"
	"dockzilla/pkg/domain"
	errs "dockzilla/pkg/domain/errors"
	"github.com/gin-gonic/gin"
)

const (
	// _schemaMediaType is the media type registered for JSON Schema. The
	// retrieve endpoints serve the document itself rather than a wrapper, so a
	// caller can hand the response body straight to its own validator.
	_schemaMediaType = "application/schema+json"

	// _schemaVersionHeader reports which version was served. The path does not
	// carry it when the latest was asked for.
	_schemaVersionHeader = "X-Schema-Version"

	// _maxSchemaBody caps a submitted document. Job payloads may be large;
	// the schema describing one has no reason to be.
	_maxSchemaBody = 1 << 20
)

// Schemas serves the job payload schema registry. The zero value is not
// usable; build one with NewSchemas.
type Schemas struct {
	service schemas.Handler
	logger  *slog.Logger
}

// SchemasOption configures a Schemas during construction. It is an interface
// rather than a bare function type so that options stay comparable in tests
// and can grow behaviour later without breaking callers.
type SchemasOption interface {
	apply(s *Schemas)
}

type schemasOptionFunc func(*Schemas)

func (f schemasOptionFunc) apply(s *Schemas) { f(s) }

// SchemasWithHandler sets the use case the handler delegates to. It is
// required: NewSchemas fails when no service is provided.
func SchemasWithHandler(service schemas.Handler) SchemasOption {
	return schemasOptionFunc(func(s *Schemas) {
		s.service = service
	})
}

// SchemasWithLogger sets the structured logger. It is required: NewSchemas
// fails when no logger is provided.
func SchemasWithLogger(logger *slog.Logger) SchemasOption {
	return schemasOptionFunc(func(s *Schemas) {
		s.logger = logger
	})
}

// NewSchemas builds a Schemas from opts. It returns an error when a required
// option is missing, so a caller never receives a partially initialised
// Schemas.
func NewSchemas(opts ...SchemasOption) (*Schemas, error) {
	s := &Schemas{}
	for _, opt := range opts {
		opt.apply(s)
	}

	if s.service == nil {
		return nil, errors.New("schemas handler: service is required")
	}

	if s.logger == nil {
		return nil, errors.New("schemas handler: logger is required")
	}

	return s, nil
}

// List writes the registered schemas, newest first, optionally narrowed to one
// kind by the "kind" query parameter. Documents are omitted: a caller that
// wants one fetches it by reference.
func (s *Schemas) List(c *gin.Context) {
	listed, err := s.service.List(c.Request.Context(), domain.Kind(c.Query("kind")))
	if err != nil {
		s.fail(c, "list", err)

		return
	}

	summaries := make([]schemaSummary, 0, len(listed))
	for _, schema := range listed {
		summaries = append(summaries, summarise(schema))
	}

	c.JSON(http.StatusOK, gin.H{"schemas": summaries})
}

// Register publishes a schema. It answers 409 when the version already exists
// with a different document, because a published version is frozen and the
// caller has to pick a new one.
func (s *Schemas) Register(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, _maxSchemaBody)

	var request registerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		s.logger.DebugContext(c.Request.Context(), "rejected schema registration",
			"error", err,
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})

		return
	}

	ref := domain.SchemaRef{
		Kind:    domain.Kind(request.Kind),
		Version: domain.SchemaVersion(request.Version),
	}

	stored, err := s.service.Register(c.Request.Context(), ref, request.Schema)
	if err != nil {
		s.fail(c, "register", err)

		return
	}

	c.Header(_schemaVersionHeader, string(stored.Ref.Version))
	c.JSON(http.StatusCreated, summarise(stored))
}

// RetrieveLatest writes the document of the most recently registered version
// of the kind in the path.
func (s *Schemas) RetrieveLatest(c *gin.Context) {
	stored, err := s.service.Latest(c.Request.Context(), domain.Kind(c.Param("kind")))
	if err != nil {
		s.fail(c, "retrieve_latest", err)

		return
	}

	s.writeDocument(c, stored)
}

// Retrieve writes the document of the version named in the path.
func (s *Schemas) Retrieve(c *gin.Context) {
	ref := domain.SchemaRef{
		Kind:    domain.Kind(c.Param("kind")),
		Version: domain.SchemaVersion(c.Param("version")),
	}

	stored, err := s.service.Retrieve(c.Request.Context(), ref)
	if err != nil {
		s.fail(c, "retrieve", err)

		return
	}

	s.writeDocument(c, stored)
}

func (s *Schemas) writeDocument(c *gin.Context, schema domain.Schema) {
	c.Header(_schemaVersionHeader, string(schema.Ref.Version))
	c.Data(http.StatusOK, _schemaMediaType, schema.Document)
}

// fail maps a registry error onto a status code. Anything unrecognised is a
// 500 and is logged at error level; the rest are the caller's fault and are
// logged at debug so a client looping on a bad request cannot fill the logs.
func (s *Schemas) fail(c *gin.Context, action string, err error) {
	ctx := c.Request.Context()
	status, message := classify(err)

	if status == http.StatusInternalServerError {
		s.logger.ErrorContext(ctx, "schema registry request failed",
			"action", action,
			"error", err,
		)
	} else {
		s.logger.DebugContext(ctx, "schema registry request rejected",
			"action", action,
			"status", status,
			"error", err,
		)
	}

	c.JSON(status, gin.H{"error": message})
}

// registerRequest is the body of a publish. The document is kept raw so it
// reaches the registry exactly as it was sent.
type registerRequest struct {
	Kind    string          `json:"kind" binding:"required"`
	Version string          `json:"version" binding:"required"`
	Schema  json.RawMessage `json:"schema" binding:"required"`
}

// schemaSummary is a schema without its document, which is what the list and
// register responses carry.
type schemaSummary struct {
	Identifier string    `json:"identifier,omitempty"`
	Kind       string    `json:"kind"`
	Version    string    `json:"version"`
	CreatedAt  time.Time `json:"created_at,omitzero"`
}

func summarise(schema domain.Schema) schemaSummary {
	return schemaSummary{
		Identifier: schema.Identifier,
		Kind:       string(schema.Ref.Kind),
		Version:    string(schema.Ref.Version),
		CreatedAt:  schema.CreatedAt,
	}
}

func classify(err error) (status int, message string) {
	switch {
	case errors.Is(err, errs.ErrSchemaNotFound):
		return http.StatusNotFound, "schema not found"

	case errors.Is(err, errs.ErrInvalidSchemaRef):
		return http.StatusBadRequest, "kind and version are required"

	case errors.Is(err, errs.ErrSchemaInvalid):
		return http.StatusBadRequest, "document is not a valid json schema"

	case errors.Is(err, errs.ErrSchemaImmutable):
		return http.StatusConflict, "version already registered with a different document"

	default:
		return http.StatusInternalServerError, "internal server error"
	}
}
