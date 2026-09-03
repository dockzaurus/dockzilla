package docker

import (
	"context"
	"fmt"
	"io"
	"sync"
	"dockzilla/pkg/domain"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

type ImageAdapter struct {
	client client.APIClient
	uuidGen domain.Generator
	mu sync.RWMutex
	digestToID map[string]domain.ImageID
	idToDigest map[domain.ImageID]string
}

func NewImageAdapter(gen domain.Generator) (domain.DockerImageAPI, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
        return nil, fmt.Errorf("failed to create docker client: %w", err)
    }
    return &ImageAdapter{
		client: cli,
		uuidGen: gen,
		digestToID: make(map[string]domain.ImageID),
		idToDigest: make(map[domain.ImageID]string),
	}, nil
}

func (a *ImageAdapter) Remove(ctx context.Context, id domain.ImageID) (int64, error) {
	a.mu.RLock()
	nativeID, exists := a.idToDigest[id]
	a.mu.RUnlock()

	if !exists {
		return 0, fmt.Errorf("image with ID %s not found", id)
	}

	_, err := a.client.ImageRemove(ctx, nativeID, image.RemoveOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to remove image %s: %w", id, err)
	}
	return 0, nil
}

func (a *ImageAdapter) Inventory(ctx context.Context) ([]domain.Image, error) {
	dockerImages, err := a.client.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	var result []domain.Image
	result = make([]domain.Image, 0, len(dockerImages))
	for _, img := range dockerImages {
		nativeID := img.ID
		id, exists := a.digestToID[nativeID]
		if !exists {
			id = domain.ImageID(a.uuidGen())
			a.digestToID[nativeID] = id
			a.idToDigest[id] = nativeID
		}

		result = append(result, domain.Image{
			Digest:    img.ID,
			ID:        id,
			SizeBytes: img.Size,
		})
	}

	return result, nil
}

func (a *ImageAdapter) Resolve(ctx context.Context, ref string) (domain.ImageID, bool,error) {
	inspectData, _, err := a.client.ImageInspectWithRaw(ctx, ref)
	if err != nil {
		return domain.ImageID{}, false, fmt.Errorf("failed to inspect image %s: %w", ref, err)
	}

	nativeID := inspectData.ID

	a.mu.Lock()
	defer a.mu.Unlock()

	id, exists := a.digestToID[nativeID]
	if !exists {
		id = domain.ImageID(a.uuidGen())
		a.digestToID[nativeID] = id
		a.idToDigest[id] = nativeID
	}

	return id, true, nil
}

func (a *ImageAdapter) Load(ctx context.Context, r io.Reader) (string, error) {
	resp, err := a.client.ImageLoad(ctx, r)
	if err != nil {
		return "", fmt.Errorf("failed to load image: %w", err)
	}
	defer resp.Body.Close()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
        return "", fmt.Errorf("failed to read load response: %w", err)
    }


	return "", nil
}