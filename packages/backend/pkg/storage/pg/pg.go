// Package pg provides the Postgres storage component: a bun.DB connection
// pool that lives with the application lifecycle. Run verifies connectivity
// and starts a background health check; callers consult Healthy to fail fast
// with ErrUnavailable instead of queueing behind dial timeouts while the
// database is down.
package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/zixyos/goloader/service"
)

// ErrUnavailable is the error callers should return when Healthy reports the
// database as down, so requests fail fast instead of holding a pool slot for
// the length of a dial timeout.
var ErrUnavailable = errors.New("pg: storage unavailable")

var _ serviceloader.Service = (*Storage)(nil)

// Storage owns the Postgres connection pool and its health state. The zero
// value is not usable; build one with New.
type Storage struct {
	logger *slog.Logger
	db     *bun.DB

	healthy atomic.Bool

	mu        sync.RWMutex
	cancel    context.CancelFunc
	serviceID serviceloader.UUID
	cfg       *Config

	wg sync.WaitGroup
}

// Option configures a Storage during construction. It is an interface rather
// than a bare function type so that options stay comparable in tests and can
// grow behaviour later without breaking callers.
type Option interface {
	apply(s *Storage)
}

type optionFunc func(*Storage)

func (f optionFunc) apply(s *Storage) { f(s) }

// WithLogger sets the structured logger used by the storage. It is required:
// New fails without it.
func WithLogger(logger *slog.Logger) Option {
	return optionFunc(func(s *Storage) {
		s.logger = logger
	})
}

// WithConfig sets the storage configuration (URL, timeouts, pool sizing). It
// is required: New fails without it or without cfg.URL.
func WithConfig(cfg *Config) Option {
	return optionFunc(func(s *Storage) {
		s.cfg = cfg
	})
}

// New builds a Storage from opts. It validates required options and applies
// defaults to a copy of the config so the caller's struct is never mutated.
func New(opts ...Option) (*Storage, error) {
	s := &Storage{}
	for _, opt := range opts {
		opt.apply(s)
	}

	if s.logger == nil {
		return nil, errors.New("pg: logger is required")
	}

	if s.cfg == nil {
		return nil, errors.New("pg: config is required")
	}

	if s.cfg.URL == "" {
		return nil, errors.New("pg: database URL is required")
	}

	cfg := *s.cfg
	if cfg.ServiceName == "" {
		cfg.ServiceName = _defaultServiceName
	}
	if cfg.MaxOpenConns == 0 {
		cfg.MaxOpenConns = _defaultMaxOpenConns
	}
	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = _defaultMaxIdleConns
	}
	if cfg.ConnMaxLifetime == 0 {
		cfg.ConnMaxLifetime = _defaultConnMaxLifetime
	}
	if cfg.ConnMaxIdleTime == 0 {
		cfg.ConnMaxIdleTime = _defaultConnMaxIdleTime
	}
	if cfg.DialTimeout == 0 {
		cfg.DialTimeout = _defaultDialTimeout
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = _defaultReadTimeout
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = _defaultWriteTimeout
	}
	if cfg.PingInterval == 0 {
		cfg.PingInterval = _defaultPingInterval
	}

	s.cfg = &cfg

	connector := pgdriver.NewConnector(
		pgdriver.WithDSN(cfg.URL),
		pgdriver.WithApplicationName(cfg.ServiceName),
		pgdriver.WithDialTimeout(cfg.DialTimeout),
		pgdriver.WithReadTimeout(cfg.ReadTimeout),
		pgdriver.WithWriteTimeout(cfg.WriteTimeout),
	)
	sqlDB := sql.OpenDB(connector)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	s.db = bun.NewDB(sqlDB, pgdialect.New())

	return s, nil
}

// Run pings the database once so a misconfigured DSN fails at startup, then
// starts the background health check. It returns once that check is running,
// and it returns the ping error while still starting the health loop, so a
// database that is down at boot is degraded rather than fatal and recovers on
// its own.
func (s *Storage) Run(ctx context.Context) error {
	loopCtx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancel = cancel
	s.mu.Unlock()

	s.wg.Add(1)
	go s.healthLoop(loopCtx)

	pingCtx, pingCancel := context.WithTimeout(ctx, s.cfg.DialTimeout)
	err := s.db.PingContext(pingCtx)
	pingCancel()

	s.healthy.Store(err == nil)

	if err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	s.logger.InfoContext(ctx, "postgres storage connected")
	return nil
}

// Stop stops the health check, waits for it, and closes the pool. It is safe
// to call even if Run was never called or failed.
func (s *Storage) Stop(ctx context.Context) error {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Unlock()

	s.wg.Wait()

	s.healthy.Store(false)

	if err := s.db.Close(); err != nil {
		return fmt.Errorf("close postgres: %w", err)
	}

	return nil
}

// DB exposes the underlying pool so repositories and the transactor can be
// wired; the handle is valid for the life of the Storage.
func (s *Storage) DB() *bun.DB {
	return s.db
}

// Healthy reports the last health-check result; callers should fail fast with
// ErrUnavailable when it is false.
func (s *Storage) Healthy() bool {
	return s.healthy.Load()
}

// Name returns the component name used in logs.
func (s *Storage) Name() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.cfg.ServiceName
}

// SetServiceID sets the service identifier used by the application loader.
func (s *Storage) SetServiceID(serviceID serviceloader.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.serviceID = serviceID
}

func (s *Storage) healthLoop(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.cfg.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingCtx, pingCancel := context.WithTimeout(ctx, s.cfg.DialTimeout)
			err := s.db.PingContext(pingCtx)
			pingCancel()

			was := s.healthy.Swap(err == nil)
			now := err == nil

			// Log transitions only.
			if was && !now {
				s.logger.WarnContext(ctx, "postgres storage unhealthy",
					"error", err,
				)
			} else if !was && now {
				s.logger.InfoContext(ctx, "postgres storage recovered")
			}
		}
	}
}
