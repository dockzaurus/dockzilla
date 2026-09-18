package images

import (
	"time"

	"dockzilla/pkg/domain"
)

// Image represents a Docker image with its metadata.
type Image struct {
	Digest    string
	ID        ImageID
	SizeBytes int64
	CreatedAt time.Time
}

// ImageID is a unique identifier for an image, represented as a UUID.
type ImageID = domain.UUID
