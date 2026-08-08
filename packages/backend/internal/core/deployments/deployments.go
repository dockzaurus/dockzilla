package deployments

import "log/slog"

type UseCase struct {
	logger *slog.Logger
}

type UseCaseOption interface {
	apply(*UseCase)
}

type useCaseOptionFunc func(*UseCase)

func (f useCaseOptionFunc) apply(r *UseCase) { f(r) }

func NewUseCase(opts ...UseCaseOption) *UseCase {
	uc := new(UseCase)
	for _, opt := range opts {
		opt.apply(uc)
	}

	return uc
}
