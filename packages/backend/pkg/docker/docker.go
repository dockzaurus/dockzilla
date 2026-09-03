package docker

import (
	"context"
	"github.com/docker/docker/client"
	"fmt"
)

type Storage struct {
	clientSDK client.APIClient
}

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

func (s *Storage) Client() client.APIClient {
	return s.clientSDK
}


func (s *Storage) Run(ctx context.Context) error {
	_, err := s.clientSDK.Ping(ctx)
	if err != nil {
		return fmt.Errorf("failed to ping Docker daemon: %w", err)
	}
	return nil
}

func (s *Storage) Stop(ctx context.Context) error {
	return s.clientSDK.Close()
}

func (s *Storage) Name() string {
	return "Docker Storage"
}
