package docker

import (
	"context"
	"fmt"
	"io"
	"sync"
	"dockzilla/pkg/domain"
	"dockzilla/internal/core/images"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

type ImageAdapter struct {
	client *client.Client
	uuidGen domain.Generator
	mu sync.RWMutex
	digestToID map[string]images.ImageID
	idToDigest map[images.ImageID]string
}

func NewImageAdapter(gen domain.Generator) (*ImageAdapter, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
        return nil, fmt.Errorf("failed to create docker client: %w", err)
    }
    return &ImageAdapter{
		client: cli,
		uuidGen: gen,
		digestToID: make(map[string]images.ImageID),
		idToDigest: make(map[images.ImageID]string),
	}, nil
}

func (a *ImageAdapter) Remove(ctx context.Context, id images.ImageID) error {
	a.mu.RLock()
	nativeID, exists := a.idToDigest[id]
	a.mu.RUnlock()

	if !exists {
		return fmt.Errorf("image with ID %s not found", id)
	}

	_, err := a.client.ImageRemove(ctx, nativeID, image.RemoveOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove image %s: %w", id, err)
	}
	return nil
}

func (a *ImageAdapter) Inventory(ctx context.Context) ([]images.Image, error) {
	dockerImages, err := a.client.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	var result []images.Image
	for _, img := range dockerImages {
		nativeID := img.ID
		id, exists := a.digestToID[nativeID]
		if !exists {
			id = images.ImageID(a.uuidGen())
			a.digestToID[nativeID] = id
			a.idToDigest[id] = nativeID
		}

		result = append(result, images.Image{
			Digest:    img.ID,
			ID:        id,
			SizeBytes: img.Size,
		})
	}

	return result, nil
}

func (a *ImageAdapter) Resolve(ctx context.Context, ref string) (images.Image, error) {
	inspectData, _, err := a.client.ImageInspectWithRaw(ctx, ref)
	if err != nil {
		return images.Image{}, fmt.Errorf("failed to inspect image %s: %w", ref, err)
	}

	nativeID := inspectData.ID

	a.mu.Lock()
	defer a.mu.Unlock()

	id, exists := a.digestToID[nativeID]
	if !exists {
		id = images.ImageID(a.uuidGen())
		a.digestToID[nativeID] = id
		a.idToDigest[id] = nativeID
	}

	return images.Image{Digest: nativeID, ID: id, SizeBytes: inspectData.Size }, nil
}

func (a *ImageAdapter) Load(ctx context.Context, r io.Reader) error {
	resp, err := a.client.ImageLoad(ctx, r)
	if err != nil {
		return fmt.Errorf("failed to load image: %w", err)
	}
	defer resp.Body.Close()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
        return fmt.Errorf("failed to read load response: %w", err)
    }


	return nil
}