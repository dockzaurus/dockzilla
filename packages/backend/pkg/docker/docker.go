package docker

import (
	"context"
	"github.com/docker/docker/client"
	"fmt"
)

type Storage struct {
	clientSDK *client.Client
}

func NewStorage() (*Storage, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &Storage{
		clientSDK: cli,
	}, nil
}

func (s *Storage) Client() *client.Client {
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
