package deployments

import (
	"context"

	"dockzilla/pkg/domain"
)

// Repository defines the storage interface for deployments.
type Repository interface {
	Insert(ctx context.Context, deployment domain.Deployment) (domain.UUID, error)
}
