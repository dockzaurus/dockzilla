package main

// LoadConfig reads the embedded TOML for the current APP_ENV, so these tests
// set that variable and cannot run in parallel with each other.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Setenv("APP_ENV", "local")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Equal(t, "dockzilla-back", cfg.Service.Name)
	require.Equal(t, "v1", cfg.Service.Version)

	require.Equal(t, "dockzilla-postgres", cfg.Storage.Database.ServiceName)
	require.NotEmpty(t, cfg.Storage.Database.URL)
	require.Equal(t, "dockzilla-redis", cfg.Storage.Cache.ServiceName)
	require.NotEmpty(t, cfg.Storage.Cache.URL)

	require.Equal(t, "dckz", cfg.Queue.Name)
	require.Equal(t, "dckz-worker", cfg.Queue.Consumer)
	require.Equal(t, 500*time.Millisecond, cfg.Queue.Tick)

	require.Equal(t, "dockzilla-http-api", cfg.HTTP.ServiceName)
	require.Equal(t, 8181, cfg.HTTP.HTTPServer.Port)
}

func TestLoadConfig_EnvironmentOverridesTheFile(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("SERVICE_NAME", "from-the-environment")

	cfg, err := LoadConfig()
	require.NoError(t, err)

	require.Equal(t, "from-the-environment", cfg.Service.Name)
}

func TestLoadConfig_UnknownEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", "does-not-exist")

	cfg, err := LoadConfig()

	require.Error(t, err)
	require.Nil(t, cfg)
	require.ErrorContains(t, err, "load config")
}
