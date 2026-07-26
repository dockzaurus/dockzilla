// Package redis provides the Redis cache component: a go-redis client whose
// connection pool lives with the application lifecycle. Run verifies
// connectivity and starts a background health check; callers consult Healthy
// to fail fast with ErrUnavailable instead of queueing behind dial timeouts
// while the cache is down.
package redis

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/zixyos/goloader/service"
)

// ErrUnavailable is the error callers should return when Healthy reports the
// cache as down, so requests fail fast instead of holding a pool slot for the
// length of a dial timeout.
var ErrUnavailable = errors.New("redis: cache unavailable")

var _ serviceloader.Service = (*Cache)(nil)

// Cache owns the Redis client and its health state. The zero value is not
// usable; build one with New.
type Cache struct {
	logger *slog.Logger
	client *goredis.Client

	healthy atomic.Bool

	mu        sync.RWMutex
	cancel    context.CancelFunc
	serviceID serviceloader.UUID
	cfg       *Config

	wg sync.WaitGroup
}

// Option configures a Cache during construction. It is an interface rather
// than a bare function type so that options stay comparable in tests and can
// grow behaviour later without breaking callers.
type Option interface {
	apply(c *Cache)
}

type optionFunc func(*Cache)

func (f optionFunc) apply(c *Cache) { f(c) }

// WithLogger sets the structured logger used by the cache. It is required:
// New fails without it.
func WithLogger(logger *slog.Logger) Option {
	return optionFunc(func(c *Cache) {
		c.logger = logger
	})
}

// WithConfig sets the cache configuration (URL, timeouts, pool sizing). It is
// required: New fails without it or without cfg.URL.
func WithConfig(cfg *Config) Option {
	return optionFunc(func(c *Cache) {
		c.cfg = cfg
	})
}

// New builds a Cache from opts. It validates required options and applies
// defaults to a copy of the config so the caller's struct is never mutated.
func New(opts ...Option) (*Cache, error) {
	c := &Cache{}
	for _, opt := range opts {
		opt.apply(c)
	}

	if c.logger == nil {
		return nil, errors.New("redis: logger is required")
	}

	if c.cfg == nil {
		return nil, errors.New("redis: config is required")
	}

	if c.cfg.URL == "" {
		return nil, errors.New("redis: cache URL is required")
	}

	cfg := *c.cfg
	if cfg.ServiceName == "" {
		cfg.ServiceName = _defaultServiceName
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
	if cfg.ConnMaxLifetime == 0 {
		cfg.ConnMaxLifetime = _defaultConnMaxLifetime
	}
	if cfg.ConnMaxIdleTime == 0 {
		cfg.ConnMaxIdleTime = _defaultConnMaxIdleTime
	}
	if cfg.PingInterval == 0 {
		cfg.PingInterval = _defaultPingInterval
	}

	c.cfg = &cfg

	options, err := goredis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	// Explicit configuration and package defaults deliberately win over URL
	// query parameters, matching the pg component's DSN behaviour.
	options.DialTimeout = cfg.DialTimeout
	options.ReadTimeout = cfg.ReadTimeout
	options.WriteTimeout = cfg.WriteTimeout
	options.ConnMaxLifetime = cfg.ConnMaxLifetime
	options.ConnMaxIdleTime = cfg.ConnMaxIdleTime
	if cfg.PoolSize != 0 {
		options.PoolSize = cfg.PoolSize
	}
	if cfg.MinIdleConns != 0 {
		options.MinIdleConns = cfg.MinIdleConns
	}
	if cfg.PoolTimeout != 0 {
		options.PoolTimeout = cfg.PoolTimeout
	}

	c.client = goredis.NewClient(options)

	return c, nil
}

// Run pings the cache once so a misconfigured URL fails at startup, then
// starts the background health check. It returns once that check is running,
// and it returns the ping error while still starting the health loop, so a
// cache that is down at boot is degraded rather than fatal and recovers on
// its own.
func (c *Cache) Run(ctx context.Context) error {
	loopCtx, cancel := context.WithCancel(ctx)
	c.mu.Lock()
	c.cancel = cancel
	c.mu.Unlock()

	c.wg.Add(1)
	go c.healthLoop(loopCtx)

	pingCtx, pingCancel := context.WithTimeout(ctx, c.cfg.DialTimeout)
	err := c.client.Ping(pingCtx).Err()
	pingCancel()

	c.healthy.Store(err == nil)

	if err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}

	c.logger.InfoContext(ctx, "redis cache connected")
	return nil
}

// Stop stops the health check, waits for it, and closes the client. It is safe
// to call even if Run was never called or failed.
func (c *Cache) Stop(ctx context.Context) error {
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
	}
	c.mu.Unlock()

	c.wg.Wait()

	c.healthy.Store(false)

	if err := c.client.Close(); err != nil {
		return fmt.Errorf("close redis: %w", err)
	}

	return nil
}

// Client exposes the underlying go-redis client so repositories can be wired;
// the handle is valid for the life of the Cache.
func (c *Cache) Client() *goredis.Client {
	return c.client
}

// Healthy reports the last health-check result; callers should fail fast with
// ErrUnavailable when it is false.
func (c *Cache) Healthy() bool {
	return c.healthy.Load()
}

// Name returns the component name used in logs.
func (c *Cache) Name() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.cfg.ServiceName
}

// SetServiceID sets the service identifier used by the application loader.
func (c *Cache) SetServiceID(serviceID serviceloader.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.serviceID = serviceID
}

func (c *Cache) healthLoop(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(c.cfg.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingCtx, pingCancel := context.WithTimeout(ctx, c.cfg.DialTimeout)
			err := c.client.Ping(pingCtx).Err()
			pingCancel()

			was := c.healthy.Swap(err == nil)
			now := err == nil

			// Log transitions only.
			if was && !now {
				c.logger.WarnContext(ctx, "redis cache unhealthy",
					"error", err,
				)
			} else if !was && now {
				c.logger.InfoContext(ctx, "redis cache recovered")
			}
		}
	}
}
