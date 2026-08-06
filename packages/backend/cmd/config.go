package main

import (
	"embed"
	"fmt"

	"dockzilla/pkg/queue/pgqueue"
	"dockzilla/pkg/storage/pg"
	"dockzilla/pkg/storage/redis"
	"github.com/zixyos/giniservice/http"
	goconfig "github.com/zixyos/goloader/config"
)

// _configFiles holds the per-environment TOML files, compiled into the binary
// so the service has no runtime dependency on its working directory.
//
//go:embed *.toml
var _configFiles embed.FS

// StorageConfig type embed the storage providers configuration.
type StorageConfig struct {
	Database pg.Config    `koanf:"database"`
	Cache    redis.Config `koanf:"cache"`
}

// ServiceConfig type represent the service configuration.
type ServiceConfig struct {
	Name    string `koanf:"name"`
	Version string `koanf:"version"`
}

// Config type represent the Application configuration.
type Config struct {
	HTTP    http.Config    `koanf:"http"`
	Service ServiceConfig  `koanf:"service"`
	Storage StorageConfig  `koanf:"storage"`
	Queue   pgqueue.Config `koanf:"queue"`
}

// LoadConfig reads the configuration for the current APP_ENV from the embedded
// TOML files, with environment variables taking precedence.
func LoadConfig() (*Config, error) {
	var config Config
	if err := goconfig.Load(&config, goconfig.WithFs(_configFiles)); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	return &config, nil
}
