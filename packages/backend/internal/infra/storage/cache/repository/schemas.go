// Package repository holds the Redis-backed implementations of the cache ports
// declared by the application core. Every type in it is a cache and nothing
// more: a miss and a failure are the same answer to the caller, which falls
// through to the durable store.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"dockzilla/internal/core/jobs/schemas"
	"dockzilla/pkg/domain"
	errs "dockzilla/pkg/domain/errors"
	goredis "github.com/redis/go-redis/v9"
)

const (
	// _schemaKeyPrefix namespaces registry entries inside the shared cache.
	_schemaKeyPrefix = "job:schema:"

	// _defaultSchemaTTL bounds how long an entry occupies memory. A published
	// version is immutable, so the TTL is about eviction rather than staleness
	// and can be generous.
	_defaultSchemaTTL = 24 * time.Hour
)

var _ schemas.CacheRepository = (*Schemas)(nil)

// Schemas caches schema documents in Redis, in front of the Postgres registry.
// The zero value is not usable; build one with NewSchemas.
type Schemas struct {
	logger *slog.Logger
	client *goredis.Client
	ttl    time.Duration
}

// SchemasOption configures a Schemas during construction.
type SchemasOption interface {
	apply(s *Schemas)
}

type schemasOptionFunc func(*Schemas)

func (f schemasOptionFunc) apply(s *Schemas) { f(s) }

// SchemasWithLogger sets the structured logger. Required.
func SchemasWithLogger(logger *slog.Logger) SchemasOption {
	return schemasOptionFunc(func(s *Schemas) {
		s.logger = logger
	})
}

// SchemasWithClient sets the Redis client. Required.
func SchemasWithClient(client *goredis.Client) SchemasOption {
	return schemasOptionFunc(func(s *Schemas) {
		s.client = client
	})
}

// SchemasWithTTL overrides how long an entry is kept.
func SchemasWithTTL(ttl time.Duration) SchemasOption {
	return schemasOptionFunc(func(s *Schemas) {
		s.ttl = ttl
	})
}

// NewSchemas builds a Schemas from opts, returning an error when a required
// option is missing.
func NewSchemas(opts ...SchemasOption) (*Schemas, error) {
	r := &Schemas{ttl: _defaultSchemaTTL}

	for _, opt := range opts {
		opt.apply(r)
	}

	if r.logger == nil {
		return nil, errors.New("schemas cache: logger is required")
	}

	if r.client == nil {
		return nil, errors.New("schemas cache: client is required")
	}

	return r, nil
}

// Get returns the cached schema for ref, or ErrSchemaNotFound on a miss.
func (s *Schemas) Get(ctx context.Context, ref domain.SchemaRef) (domain.Schema, error) {
	raw, err := s.client.Get(ctx, key(ref)).Bytes()
	if errors.Is(err, goredis.Nil) {
		return domain.Schema{}, fmt.Errorf("%w: %s", errs.ErrSchemaNotFound, ref)
	}

	if err != nil {
		return domain.Schema{}, fmt.Errorf("read cached schema %s: %w", ref, err)
	}

	var cached entry
	if err := json.Unmarshal(raw, &cached); err != nil {
		// A value this process cannot read is indistinguishable from a miss,
		// and the read path recovers by going to the repository.
		s.logger.WarnContext(ctx, "discarding unreadable cached schema",
			"schema_ref", ref.String(),
			"error", err,
		)

		return domain.Schema{}, fmt.Errorf("%w: %s", errs.ErrSchemaNotFound, ref)
	}

	return domain.Schema{
		Identifier: cached.Identifier,
		Ref:        ref,
		Document:   cached.Document,
		CreatedAt:  cached.CreatedAt,
	}, nil
}

// Put stores schema for the configured TTL.
func (s *Schemas) Put(ctx context.Context, schema domain.Schema) error {
	raw, err := json.Marshal(entry{
		Identifier: schema.Identifier,
		Document:   schema.Document,
		CreatedAt:  schema.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("encode schema %s: %w", schema.Ref, err)
	}

	if err := s.client.Set(ctx, key(schema.Ref), raw, s.ttl).Err(); err != nil {
		return fmt.Errorf("cache schema %s: %w", schema.Ref, err)
	}

	return nil
}

// entry is the cached representation. The reference is the key, so it is not
// repeated in the value.
type entry struct {
	Identifier string          `json:"identifier"`
	Document   json.RawMessage `json:"document"`
	CreatedAt  time.Time       `json:"created_at"`
}

func key(ref domain.SchemaRef) string {
	return _schemaKeyPrefix + ref.String()
}
