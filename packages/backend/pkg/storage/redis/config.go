package redis

import "time"

// Config configures the Redis cache component. Zero-valued fields fall back to
// the package defaults in constants.go where one exists, and to go-redis's own
// defaults otherwise (PoolSize, MinIdleConns, PoolTimeout); URL is the only
// required field, and explicit configuration takes precedence over query
// parameters in the URL.
type Config struct {
	ServiceName     string        `koanf:"service_name"`
	ServiceVersion  string        `koanf:"service_version"`
	URL             string        `koanf:"url"`
	PoolSize        int           `koanf:"pool_size"`
	MinIdleConns    int           `koanf:"min_idle_conns"`
	PoolTimeout     time.Duration `koanf:"pool_timeout"`
	ConnMaxLifetime time.Duration `koanf:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `koanf:"conn_max_idle_time"`
	DialTimeout     time.Duration `koanf:"dial_timeout"`
	ReadTimeout     time.Duration `koanf:"read_timeout"`
	WriteTimeout    time.Duration `koanf:"write_timeout"`
	PingInterval    time.Duration `koanf:"ping_interval"`
}
