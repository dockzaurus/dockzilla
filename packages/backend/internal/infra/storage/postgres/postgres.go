// Package postgres exposes the Postgres storage of the backend. It is a thin
// adapter around the pg storage component that satisfies domain.Service, so the
// application core can start and stop it like any other handler.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"dockzilla/pkg/domain"
	"dockzilla/pkg/storage/pg"
	"github.com/uptrace/bun"
)

// Verify at compile time that a Storage is a handler the application can run.
var _ domain.Service = (*Storage)(nil)

// Storage adapts the pg storage component to the domain.Service contract. The
// zero value is not usable; build one with NewStorage.
type Storage struct {
	logger *slog.Logger
	store  *pg.Storage
	cfg    *pg.Config
}

// Option configures a Storage during construction. It is an interface rather
// than a bare function type so that options stay comparable in tests and can
// grow behaviour later without breaking callers.
type Option interface {
	apply(s *Storage)
}

// optionFunc adapts a plain function to the Option interface.
type optionFunc func(*Storage)

func (f optionFunc) apply(s *Storage) { f(s) }

// WithLogger sets the structured logger used by the storage. It is required:
// NewStorage fails when no logger is provided.
func WithLogger(logger *slog.Logger) Option {
	return optionFunc(func(s *Storage) {
		s.logger = logger
	})
}

// WithConfig sets the storage configuration (database URL, pool sizing,
// timeouts). It is required: NewStorage fails when no configuration is provided.
func WithConfig(cfg *pg.Config) Option {
	return optionFunc(func(s *Storage) {
		s.cfg = cfg
	})
}

// NewStorage builds a Storage from opts. It returns an error when a required
// option is missing or when the underlying pg storage component cannot be
// created, so a caller never receives a partially initialised Storage.
func NewStorage(opts ...Option) (*Storage, error) {
	s := &Storage{}
	for _, opt := range opts {
		opt.apply(s)
	}

	if s.logger == nil {
		return nil, errors.New("postgres storage: logger is required")
	}

	if s.cfg == nil {
		return nil, errors.New("postgres storage: config is required")
	}

	store, err := pg.New(pg.WithLogger(s.logger), pg.WithConfig(s.cfg))
	if err != nil {
		return nil, fmt.Errorf("postgres storage: %w", err)
	}

	s.store = store

	return s, nil
}

// Run starts the postgres storage. It returns once the initial connectivity
// check and background health check are started.
func (s *Storage) Run(ctx context.Context) error {
	s.logger.InfoContext(ctx, "starting postgres storage")

	if err := s.store.Run(ctx); err != nil {
		return fmt.Errorf("run postgres storage: %w", err)
	}

	return nil
}

// Stop gracefully shuts the postgres storage down, honouring ctx's deadline.
func (s *Storage) Stop(ctx context.Context) error {
	if err := s.store.Stop(ctx); err != nil {
		return fmt.Errorf("stop postgres storage: %w", err)
	}

	return nil
}

// Name returns the service name used by the application core and the logs.
func (s *Storage) Name() string {
	return s.store.Name()
}

// DB exposes the underlying bun handle so repositories and the transactor can
// be wired.
func (s *Storage) DB() *bun.DB {
	return s.store.DB()
}

// Healthy reports whether the last database health check succeeded.
func (s *Storage) Healthy() bool {
	return s.store.Healthy()
}
