package pg

import "time"

// Pool sizing defaults, applied to whichever fields the configuration leaves at
// zero. They are deliberately finite: a Postgres connection is a server-side
// process, so the sum of every instance's pool has to stay under the server's
// max_connections. Twenty per instance leaves room for a handful of replicas of
// this service against a default max_connections of 100, plus headroom for
// migrations and a human with psql.
const (
	_defaultMaxOpenConns = 20

	// database/sql defaults MaxIdleConns to 2, which means a service allowed 20
	// connections would close and reopen 18 of them on every burst — a TCP
	// handshake, a TLS handshake and a Postgres startup per query. Keeping idle
	// equal to open is what makes the pool actually pool.
	_defaultMaxIdleConns = _defaultMaxOpenConns

	// A finite lifetime is how the pool notices the world changed: after a
	// failover or a DNS switch, connections still pinned to the old primary are
	// retired instead of living forever, and a replica that just came back
	// starts taking traffic again.
	_defaultConnMaxLifetime = 30 * time.Minute

	// Idle connections go back to the server during quiet periods so an idle
	// instance does not sit on its whole budget.
	_defaultConnMaxIdleTime = 5 * time.Minute
)

// Per-connection timeout defaults. Without them a network black hole — a load
// balancer that dropped the connection without a FIN, a frozen primary — leaves a
// query blocked forever while holding a pool slot, and the pool drains one stuck
// query at a time until every request is queueing behind it.
const (
	_defaultDialTimeout  = 5 * time.Second
	_defaultReadTimeout  = 30 * time.Second
	_defaultWriteTimeout = 30 * time.Second
)

// _defaultServiceName names the component in logs when the configuration does
// not provide one.
const _defaultServiceName = "postgres-storage"

// _defaultPingInterval paces the background health check. Thirty seconds is
// frequent enough to close the gate soon after an outage without adding
// measurable load.
const _defaultPingInterval = 30 * time.Second
