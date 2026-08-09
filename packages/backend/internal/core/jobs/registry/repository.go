package registry

import (
	"context"
	"dockzilla/pkg/domain"
)

type Repository interface {
	RegisterSchema(ctx context.Context, kind domain.Kind, schema any) error
	ListSchemas(ctx context.Context, kind domain.Kind) ([]any, error)
}

type CacheRepository interface {
	CacheSchema(ctx context.Context) error
	FetchSchema(ctx context.Context, kind domain.Kind) (any, error)
}
