package domain

import (
	"context"
	"io"
)

type Image struct {
	Digest string
	ID ImageID
	SizeBytes int64
}

type ImageID = UUID

type DockerImageAPI interface {
	Inventory(ctx context.Context) ([]Image, error)
	Remove(ctx context.Context, id ImageID) (int64, error)
	Resolve(ctx context.Context, ref string) (ImageID, bool, error)
	Load(ctx context.Context, reader io.Reader) (string, error)
}