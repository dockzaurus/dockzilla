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

// ImageAdapter is an implementation of the domain.
// DockerImageAPI interface that interacts with the Docker API.
type ImageAdapter struct {
	client     client.APIClient
	uuidGen    domain.Generator
	mu         sync.RWMutex
	digestToID map[string]domain.ImageID
	idToDigest map[domain.ImageID]string
}

// NewImageAdapter creates a new instance of ImageAdapter
// with the provided Docker client and UUID generator.
func NewImageAdapter(gen domain.Generator) (*ImageAdapter, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	return &ImageAdapter{
		client:     cli,
		uuidGen:    gen,
		digestToID: make(map[string]domain.ImageID),
		idToDigest: make(map[domain.ImageID]string),
	}, nil
}

// Remove removes a Docker image by its ImageID and returns the reclaimed space in bytes.
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

// Inventory retrieves the list of Docker images and maps them to the domain.Image type.
func (a *ImageAdapter) Inventory(ctx context.Context) ([]domain.Image, error) {
	dockerImages, err := a.client.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	result := make([]domain.Image, 0, len(dockerImages))
	for _, img := range dockerImages {
		nativeID := img.ID
		id, exists := a.digestToID[nativeID]
		if !exists {
			id = a.uuidGen()
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

// Resolve resolves a Docker image reference to its corresponding ImageID.
func (a *ImageAdapter) Resolve(ctx context.Context, ref string) (
	id domain.ImageID,
	exists bool, err error) {
	inspectData, err := a.client.ImageInspect(ctx, ref)
	if err != nil {
		return domain.ImageID{}, false, fmt.Errorf("failed to inspect image %s: %w", ref, err)
	}

	nativeID := inspectData.ID

	a.mu.Lock()
	defer a.mu.Unlock()

	id, exists = a.digestToID[nativeID]
	if !exists {
		id = a.uuidGen()
		a.digestToID[nativeID] = id
		a.idToDigest[id] = nativeID
	}

	return id, true, nil
}

// Load loads a Docker image from the provided reader and returns its digest.
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
