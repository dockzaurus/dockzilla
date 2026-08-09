package repository

import (
	"context"
	"dockzilla/internal/core/jobs/registry"
	"dockzilla/pkg/domain"
	"log/slog"

	"github.com/uptrace/bun"
)

var _ registry.Repository = (*Registry)(nil)

type Registry struct {
	logger *slog.Logger
	db     bun.IDB
}

type RegistryOption interface {
	apply(*Registry)
}

type registryOptionFunc func(*Registry)

func (f registryOptionFunc) apply(r *Registry) { f(r) }

func RegistryWithLogger(logger *slog.Logger) RegistryOption {
	return registryOptionFunc(func(r *Registry) {
		r.logger = logger
	})
}

func RegistryWithDB(db bun.IDB) RegistryOption {
	return registryOptionFunc(func(r *Registry) {
		r.db = db
	})
}

func NewRegistry(options ...RegistryOption) (*Registry, error) {
	r := new(Registry)

	for _, option := range options {
		option.apply(r)
	}

	return r, nil
}

func (r *Registry) RegisterSchema(ctx context.Context, kind domain.Kind, schema any) error {
	//TODO implement me
	panic("implement me")
}

func (r *Registry) ListSchemas(ctx context.Context, kind domain.Kind) ([]any, error) {
	//TODO implement me
	panic("implement me")
}
