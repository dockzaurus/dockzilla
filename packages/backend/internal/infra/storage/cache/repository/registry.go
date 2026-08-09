package repository

import (
	"context"
	"dockzilla/internal/core/jobs/registry"
	"dockzilla/pkg/domain"
	"log/slog"
)

var _ registry.CacheRepository = (*Registry)(nil)

type Registry struct {
	logger *slog.Logger
}

type RegistryOption interface {
	apply(*Registry)
}

type registryOptionFunc func(*Registry)

func (f registryOptionFunc) apply(r *Registry) { f(r) }

func WithLogger(logger *slog.Logger) RegistryOption {
	return registryOptionFunc(func(r *Registry) {
		r.logger = logger
	})
}

func (r *Registry) CacheSchema(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (r *Registry) FetchSchema(ctx context.Context, kind domain.Kind) (any, error) {
	//TODO implement me
	panic("implement me")
}
