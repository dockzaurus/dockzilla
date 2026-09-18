package images

import (
	"context"
	"io"
)

// Store defines the interface for an image store, which manages Docker images.
type Store interface {
	Inventory(ctx context.Context) ([]Image, error)
	Resolve(ctx context.Context, digest string) (id ImageID, exists bool, err error)
	Remove(ctx context.Context, id ImageID) (reclaimed int64, err error)
	Load(ctx context.Context, r io.Reader) (digest string, err error)
}
