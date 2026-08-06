package pgqueue

import "time"

// Config configures the pgque-backed job queue. Zero-valued Tick falls back to
// the package default; Name and Consumer are required.
type Config struct {
	ServiceName string        `koanf:"service_name"`
	Name        string        `koanf:"name"`
	Consumer    string        `koanf:"consumer"`
	Tick        time.Duration `koanf:"tick"`
}
