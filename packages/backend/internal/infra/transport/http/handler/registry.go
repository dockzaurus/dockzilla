package handler

import (
	"dockzilla/internal/core/jobs/registry"
	"dockzilla/internal/infra/transport/http/api"
	"log/slog"

	"github.com/gin-gonic/gin"
)

var _ (api.RegistryHandler) = (*Registry)(nil)

type Registry struct {
	logger *slog.Logger
	uc     registry.Handler
}

type RegistryOption interface {
	apply(*Registry)
}
type registryOptionFunc func(*Registry)

func (fn registryOptionFunc) apply(r *Registry) {}

func RegistryWithLogger(logger *slog.Logger) RegistryOption {
	return registryOptionFunc(func(s *Registry) {
		s.logger = logger
	})
}

func RegistryWithUc(uc registry.Handler) RegistryOption {
	return registryOptionFunc(func(s *Registry) {
		s.uc = uc
	})
}

func NewRegistry(options ...RegistryOption) (*Registry, error) {
	r := new(Registry)

	for _, option := range options {
		option.apply(r)
	}

	return r, nil
}

func (r *Registry) RegisterSchema(c *gin.Context) {
	//TODO implement me
	r.logger.InfoContext(c, "registering schema")
}

func (r *Registry) ListSchemas(c *gin.Context) {
	//TODO implement me
	r.logger.InfoContext(c, "listing schemas")
}

func (r *Registry) RetrieveSchema(c *gin.Context) {
	//TODO implement me
	r.logger.InfoContext(c, "retrieving schema")
}
