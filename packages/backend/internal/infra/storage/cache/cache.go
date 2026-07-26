// Package cache exposes the Redis cache of the backend. It is a thin adapter
// around the redis cache component that satisfies domain.Service, so the
// application core can start and stop it like any other handler.
package cache

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"dockzilla/pkg/domain"
	"dockzilla/pkg/storage/redis"
	goredis "github.com/redis/go-redis/v9"
)

// Verify at compile time that a Storage is a handler the application can run.
var _ domain.Service = (*Storage)(nil)

// Storage adapts the redis cache component to the domain.Service contract. The
// zero value is not usable; build one with NewStorage.
type Storage struct {
	logger *slog.Logger
	cache  *redis.Cache
	cfg    *redis.Config
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

// WithLogger sets the structured logger used by the cache. It is required:
// NewStorage fails when no logger is provided.
func WithLogger(logger *slog.Logger) Option {
	return optionFunc(func(s *Storage) {
		s.logger = logger
	})
}

// WithConfig sets the cache configuration (cache URL, pool sizing, timeouts).
// It is required: NewStorage fails when no configuration is provided.
func WithConfig(cfg *redis.Config) Option {
	return optionFunc(func(s *Storage) {
		s.cfg = cfg
	})
}

// NewStorage builds a Storage from opts. It returns an error when a required
// option is missing or when the underlying redis cache component cannot be
// created, so a caller never receives a partially initialised Storage.
func NewStorage(opts ...Option) (*Storage, error) {
	s := &Storage{}
	for _, opt := range opts {
		opt.apply(s)
	}

	if s.logger == nil {
		return nil, errors.New("redis cache: logger is required")
	}

	if s.cfg == nil {
		return nil, errors.New("redis cache: config is required")
	}

	cache, err := redis.New(redis.WithLogger(s.logger), redis.WithConfig(s.cfg))
	if err != nil {
		return nil, fmt.Errorf("redis cache: %w", err)
	}

	s.cache = cache

	return s, nil
}

// Run starts the redis cache. It returns once the initial connectivity check
// and background health check are started.
func (s *Storage) Run(ctx context.Context) error {
	s.logger.InfoContext(ctx, "starting redis cache")

	if err := s.cache.Run(ctx); err != nil {
		return fmt.Errorf("run redis cache: %w", err)
	}

	return nil
}

// Stop gracefully shuts the redis cache down, honouring ctx's deadline.
func (s *Storage) Stop(ctx context.Context) error {
	if err := s.cache.Stop(ctx); err != nil {
		return fmt.Errorf("stop redis cache: %w", err)
	}

	return nil
}

// Name returns the service name used by the application core and the logs.
func (s *Storage) Name() string {
	return s.cache.Name()
}

// Client exposes the underlying go-redis client so repositories can be wired.
func (s *Storage) Client() *goredis.Client {
	return s.cache.Client()
}

// Healthy reports whether the last cache health check succeeded.
func (s *Storage) Healthy() bool {
	return s.cache.Healthy()
}
