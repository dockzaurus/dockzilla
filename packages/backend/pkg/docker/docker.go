package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
)

// Storage is a wrapper around the Docker client
// that provides methods to interact with the Docker daemon.
type Storage struct {
	clientSDK *client.Client
}

// NewStorage creates a new instance of Storage with the provided configuration.
func NewStorage(cfg Config) (*Storage, error) {
	opts := []client.Opt{
		client.WithAPIVersionNegotiation(),
	}

	if cfg.Host != "" {
		opts = append(opts, client.WithHost(cfg.Host))
	} else {
		opts = append(opts, client.FromEnv)
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &Storage{
		clientSDK: cli,
	}, nil
}

// Client returns the underlying Docker client.
func (s *Storage) Client() *client.Client {
	return s.clientSDK
}

// Run pings the Docker daemon to ensure it's reachable.
func (s *Storage) Run(ctx context.Context) error {
	_, err := s.clientSDK.Ping(ctx)
	if err != nil {
		return fmt.Errorf("failed to ping Docker daemon: %w", err)
	}
	return nil
}

// Stop closes the connection to the Docker daemon.
func (s *Storage) Stop(ctx context.Context) error {
	if err := s.clientSDK.Close(); err != nil {
		return fmt.Errorf("failed to close docker client: %w", err)
	}
	return nil
}

// Name returns the name of the storage implementation.
func (s *Storage) Name() string {
	return "Docker Storage"
}
