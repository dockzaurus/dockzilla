package registry

import (
	"context"
	"dockzilla/pkg/domain"
)

type Handler interface {
	ValidateSchema(ctx context.Context, schema *domain.Payload) error
	RegisterSchema(ctx context.Context, kind domain.Kind, version string, schema *domain.Payload) error
	RetrieveSchema(ctx context.Context, kind domain.Kind, version *string) (*domain.Payload, error)
}
