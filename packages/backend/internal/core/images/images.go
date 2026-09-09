package images

import (
	"time"
	"dockzilla/pkg/domain"
)

type Image struct {
	Digest string
	ID ImageID
	SizeBytes int64
	CreatedAt time.Time
}

type ImageID = domain.UUID