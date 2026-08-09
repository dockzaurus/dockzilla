package deployments

import (
	"context"
	"dockzilla/pkg/domain"
)

// Handler defines the HTTP handler interface for deployments.
type Handler interface {
	Create(ctx context.Context, deployment *domain.CreateDeploymentInput) (domain.UUID, error)
}
