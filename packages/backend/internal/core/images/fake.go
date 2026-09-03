// Package images provides the core domain types and store port.
// NOTE: The fake store in this package does NOT simulate Docker's real-world
// veto behaviour (refusing to remove an in-use image). Furthermore, Load's
// digest is a simple stream-hash, not a real Docker load digest.
package images

import (
	"context"
	"crypto/sha256"
	"dockzilla/internal/utils"
	"fmt"
	"io"
	"sync"
	"time"
)

type FakeStore struct {
	lock sync.RWMutex
	digestToID map[string]ImageID
	images map[ImageID]Image
}

func (f *FakeStore) Inventory(ctx context.Context) ([]Image, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	var images []Image
	for _, image := range f.images {
		images = append(images, image)
	}

	return images, nil
}



func (f *FakeStore) Load(ctx context.Context, r io.Reader) (digest string, err error) {
	hash := sha256.New()

	n, err := io.Copy(hash, r)
	if err != nil {
		return "", err
	}

	digest = fmt.Sprintf("sha256:%x", hash.Sum(nil))

	f.lock.Lock()
	defer f.lock.Unlock()

	id, exists := f.digestToID[digest]
	if !exists {
		id = utils.Generator()
		f.digestToID[digest] = id
	}

	f.images[id] = Image{
		Digest: digest,
		ID: id,
		SizeBytes: n,
		CreatedAt: time.Now(),
	}

	return digest, nil
}


func (f *FakeStore) Resolve(ctx context.Context, digest string) (ImageID, bool, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	id, exists := f.digestToID[digest]
	return id, exists, nil
}

func (f *FakeStore) Remove(ctx context.Context, id ImageID) (reclaimed int64, err error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	images, exists := f.images[id]
	if !exists {
		return 0, fmt.Errorf("image with ID %s not found", id)
	}

	delete(f.images, id)
	delete(f.digestToID, images.Digest)

	return images.SizeBytes, nil
}