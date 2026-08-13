package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"dockzilla/internal/core/jobs/schemas"
	"dockzilla/internal/infra/storage/postgres"
	"dockzilla/internal/models"
	"dockzilla/pkg/domain"
	errs "dockzilla/pkg/domain/errors"
	"github.com/uptrace/bun"
)

var _ schemas.Repository = (*Schemas)(nil)

// Schemas adapts the job_schemas table to the schemas.Repository port. The
// zero value is not usable; build one with NewSchemas.
type Schemas struct {
	logger *slog.Logger
	db     bun.IDB
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

// SchemasWithDB sets the database handle used when no transaction is ambient.
// Required.
func SchemasWithDB(db bun.IDB) SchemasOption {
	return schemasOptionFunc(func(s *Schemas) {
		s.db = db
	})
}

// NewSchemas builds a Schemas from opts, returning an error when a required
// option is missing.
func NewSchemas(opts ...SchemasOption) (*Schemas, error) {
	r := new(Schemas)

	for _, opt := range opts {
		opt.apply(r)
	}

	if r.logger == nil {
		return nil, errors.New("schemas repository: logger is required")
	}

	if r.db == nil {
		return nil, errors.New("schemas repository: database is required")
	}

	return r, nil
}

// Register inserts schema and returns the row that ends up stored.
//
// The insert is ON CONFLICT DO NOTHING rather than an upsert: a published
// version is immutable, and the use case tells a re-registration of the same
// document from an attempted rewrite by comparing the returned row against
// what it sent. An upsert here would make that comparison always succeed and
// hide the drift it exists to catch.
func (s *Schemas) Register(ctx context.Context, schema domain.Schema) (domain.Schema, error) {
	row := &models.JobSchemas{
		Kind:     string(schema.Ref.Kind),
		Version:  string(schema.Ref.Version),
		Document: schema.Document,
	}

	_, err := postgres.IDB(ctx, s.db).NewInsert().
		Model(row).
		On("CONFLICT (kind, version) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return domain.Schema{}, fmt.Errorf("insert schema %s: %w", schema.Ref, err)
	}

	return s.Get(ctx, schema.Ref)
}

// Get returns the schema stored for ref.
func (s *Schemas) Get(ctx context.Context, ref domain.SchemaRef) (domain.Schema, error) {
	row := new(models.JobSchemas)

	err := postgres.IDB(ctx, s.db).NewSelect().
		Model(row).
		Where("kind = ?", string(ref.Kind)).
		Where("version = ?", string(ref.Version)).
		Where("archived_at IS NULL").
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Schema{}, fmt.Errorf("%w: %s", errs.ErrSchemaNotFound, ref)
	}

	if err != nil {
		return domain.Schema{}, fmt.Errorf("select schema %s: %w", ref, err)
	}

	return toSchema(row), nil
}

// Latest returns the most recently registered version of kind.
//
// "Latest" is insertion order, not a comparison of version strings: "v10"
// sorts before "v2" lexicographically, and the registry deliberately does not
// assume a version is a number with a prefix.
func (s *Schemas) Latest(ctx context.Context, kind domain.Kind) (domain.Schema, error) {
	row := new(models.JobSchemas)

	err := postgres.IDB(ctx, s.db).NewSelect().
		Model(row).
		Where("kind = ?", string(kind)).
		Where("archived_at IS NULL").
		Order("id DESC").
		Limit(1).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Schema{}, fmt.Errorf("%w: %s", errs.ErrSchemaNotFound, kind)
	}

	if err != nil {
		return domain.Schema{}, fmt.Errorf("select latest schema for %s: %w", kind, err)
	}

	return toSchema(row), nil
}

// List returns the stored schemas without their documents, newest first. A
// zero kind lists every kind.
func (s *Schemas) List(ctx context.Context, kind domain.Kind) ([]domain.Schema, error) {
	var rows []models.JobSchemas

	query := postgres.IDB(ctx, s.db).NewSelect().
		Model(&rows).
		Column("identifier", "kind", "version", "created_at").
		Where("archived_at IS NULL").
		Order("id DESC")

	if kind != "" {
		query = query.Where("kind = ?", string(kind))
	}

	if err := query.Scan(ctx); err != nil {
		return nil, fmt.Errorf("select schemas: %w", err)
	}

	listed := make([]domain.Schema, 0, len(rows))
	for i := range rows {
		listed = append(listed, toSchema(&rows[i]))
	}

	return listed, nil
}

func toSchema(row *models.JobSchemas) domain.Schema {
	return domain.Schema{
		Identifier: row.Identifier,
		Ref: domain.SchemaRef{
			Kind:    domain.Kind(row.Kind),
			Version: domain.SchemaVersion(row.Version),
		},
		Document:  row.Document,
		CreatedAt: row.CreatedAt,
	}
}
