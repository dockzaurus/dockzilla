package domain

import (
	"context"
	"io"
)

// Image represents a Docker image with its metadata.
type Image struct {
	Digest    string
	ID        ImageID
	SizeBytes int64
}

// ImageID is a unique identifier for an image, represented as a UUID.
type ImageID = UUID

// DockerImageAPI defines the interface for interacting with Docker images.
type DockerImageAPI interface {
	Inventory(ctx context.Context) ([]Image, error)
	Remove(ctx context.Context, id ImageID) (int64, error)
	Resolve(ctx context.Context, ref string) (ImageID, bool, error)
	Load(ctx context.Context, reader io.Reader) (string, error)
}
