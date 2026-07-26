package redis

import "time"

// Pool sizing is deliberately left to the client: when Options.PoolSize is
// zero, go-redis sizes the pool at ten connections per CPU. Unlike Postgres,
// where every connection is a server-side process that has to fit under
// max_connections, a Redis connection is a single cheap socket against a
// server that defaults to 10,000 clients, so the client's own scaling rule is
// the right default and the configuration only overrides it in unusual
// deployments. The same goes for PoolTimeout: when zero, go-redis derives it
// from the read timeout plus one second.

// Per-connection timeout defaults. They mirror go-redis's own defaults, made
// explicit here so the component owns the contract rather than whichever
// client version is vendored — and because the dial timeout doubles as the
// deadline for the health-check ping.
const (
	_defaultDialTimeout  = 5 * time.Second
	_defaultReadTimeout  = 3 * time.Second
	_defaultWriteTimeout = 3 * time.Second
)

// Connection recycling defaults.
const (
	// go-redis leaves connection lifetime unlimited by default. A finite
	// lifetime is how the pool notices the world changed: managed Redis
	// (ElastiCache, Memorystore) fails over by moving a hostname, and a
	// connection pinned to the old primary otherwise lives forever.
	_defaultConnMaxLifetime = 30 * time.Minute

	// Matches the go-redis default. Redis connections are cheap for the
	// server to hold, so idle ones are recycled far more lazily than their
	// Postgres counterparts.
	_defaultConnMaxIdleTime = 30 * time.Minute
)

// _defaultServiceName names the component in logs when the configuration does
// not provide one.
const _defaultServiceName = "redis-cache"

// _defaultPingInterval paces the background health check. Thirty seconds is
// frequent enough to close the gate soon after an outage without adding
// measurable load.
const _defaultPingInterval = 30 * time.Second
