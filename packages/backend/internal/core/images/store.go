package images

import (
    "context"
	"io"
)

type Store interface {
    Inventory(ctx context.Context) ([]Image, error)
    Resolve(ctx context.Context, digest string) (ImageID, bool, error)
    Remove(ctx context.Context, id ImageID) (reclaimed int64, err error)
    Load(ctx context.Context, r io.Reader) (digest string, err error)
}
