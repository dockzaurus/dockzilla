package main

import (
	"embed"

	"github.com/zixyos/giniservice/http"
	goconfig "github.com/zixyos/goloader/config"
)

//go:embed *.toml
var files embed.FS

// DatabaseConfig type represent the database configuration.
type DatabaseConfig struct {
	URL string `koanf:"url"`
}

// CacheConfig type represent the cache configuration.
type CacheConfig struct {
	URL string `koanf:"url"`
}

// StorageConfig type embed the storage providers configuration.
type StorageConfig struct {
	Database DatabaseConfig `koanf:"database"`
	Cache    CacheConfig    `koanf:"cache"`
}

// ServiceConfig type represent the service configuration.
type ServiceConfig struct {
	Name    string `koanf:"name"`
	Version string `koanf:"version"`
}

// Config type represent the Application configuration.
type Config struct {
	HTTP    http.Config    `koanf:"http"`
	Service ServiceConfig `koanf:"service"`
	Storage StorageConfig `koanf:"storage"`
}

// LoadConfig load to all service configuration.
func LoadConfig() (*Config, error) {
	var config *Config
	if err := goconfig.Load(&config, goconfig.WithFs(files)); err != nil {
		return nil, err
	}

	return config, nil
}
