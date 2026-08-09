package registry

import (
	"context"
	"dockzilla/pkg/domain"
)

type Handler interface {
	ValidateSchema(ctx context.Context, schema any) (bool, error)
	RegisterSchema(ctx context.Context, kind domain.Kind, version string, schema any) (bool, error)
	RetrieveSchema(ctx context.Context, kind domain.Kind, version *string) (any, error)
}
