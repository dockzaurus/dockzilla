// Package docker provides the Docker storage implementation.
package docker

// Config holds the configuration for the Docker storage.
type Config struct {
	Host string `toml:"host"`
}
