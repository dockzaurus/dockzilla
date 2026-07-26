package pg

import "time"

// Config configures the Postgres storage component. Zero-valued fields fall
// back to the package defaults in constants.go; URL is the only required
// field.
type Config struct {
	ServiceName     string        `koanf:"service_name"`
	ServiceVersion  string        `koanf:"service_version"`
	URL             string        `koanf:"url"`
	MaxOpenConns    int           `koanf:"max_open_conns"`
	MaxIdleConns    int           `koanf:"max_idle_conns"`
	ConnMaxLifetime time.Duration `koanf:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `koanf:"conn_max_idle_time"`
	DialTimeout     time.Duration `koanf:"dial_timeout"`
	ReadTimeout     time.Duration `koanf:"read_timeout"`
	WriteTimeout    time.Duration `koanf:"write_timeout"`
	PingInterval    time.Duration `koanf:"ping_interval"`
}
