package registry

import (
	"context"
	"dockzilla/pkg/domain"
	"log/slog"
)

var _ Handler = (*UseCase)(nil)

type UseCase struct {
	logger *slog.Logger

	repository Repository
	cache      CacheRepository
}

type UseCaseOption interface {
	apply(*UseCase)
}

type useCaseOptionFunc func(*UseCase)

func (f useCaseOptionFunc) apply(r *UseCase) { f(r) }

func WithLogger(logger *slog.Logger) UseCaseOption {
	return useCaseOptionFunc(func(c *UseCase) {
		c.logger = logger
	})
}

func WithRepository(repository Repository) UseCaseOption {
	return useCaseOptionFunc(func(c *UseCase) {
		c.repository = repository
	})
}

func WithCache(cache CacheRepository) UseCaseOption {
	return useCaseOptionFunc(func(c *UseCase) {
		c.cache = cache
	})
}

func NewUseCase(opts ...UseCaseOption) (*UseCase, error) {
	uc := new(UseCase)
	for _, opt := range opts {
		opt.apply(uc)
	}

	return uc, nil
}

func (u UseCase) ValidateSchema(ctx context.Context, schema *domain.Payload) error {
	//TODO implement me
	panic("implement me")
}

func (u UseCase) RegisterSchema(ctx context.Context, kind domain.Kind, version string, schema *domain.Payload) error {
	//TODO implement me
	panic("implement me")
}

func (u UseCase) RetrieveSchema(ctx context.Context, kind domain.Kind, version *string) (*domain.Payload, error) {
	//TODO implement me
	panic("implement me")
}
