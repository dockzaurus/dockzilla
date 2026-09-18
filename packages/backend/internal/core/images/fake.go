// Package images provides the core domain types and store port.
// NOTE: The fake store in this package does NOT simulate Docker's real-world
// veto behaviour (refusing to remove an in-use image). Furthermore, Load's
// digest is a simple stream-hash, not a real Docker load digest.
package images

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"sync"
	"time"

	"dockzilla/internal/utils"
)

// FakeStore is a fake implementation of the ImageStore interface.
type FakeStore struct {
	lock       sync.RWMutex
	digestToID map[string]ImageID
	images     map[ImageID]Image
}

// Inventory returns a list of all images in the fake store.
func (f *FakeStore) Inventory(ctx context.Context) ([]Image, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	var images []Image
	images = make([]Image, 0, len(f.images))
	for _, image := range f.images {
		images = append(images, image)
	}

	return images, nil
}

// Load simulates loading an image from a tarball and returns a fake digest.
func (f *FakeStore) Load(ctx context.Context, r io.Reader) (digest string, err error) {
	hash := sha256.New()

	n, err := io.Copy(hash, r)
	if err != nil {
		return "", fmt.Errorf("failed to read image data: %w", err)
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
		Digest:    digest,
		ID:        id,
		SizeBytes: n,
		CreatedAt: time.Now(),
	}

	return digest, nil
}

// Resolve returns the ImageID associated with the given digest, if it exists.
func (f *FakeStore) Resolve(ctx context.Context, digest string) (
	id ImageID, exists bool, err error) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	id, exists = f.digestToID[digest]
	return id, exists, nil
}

// Remove simulates removing an image by its ImageID.
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
