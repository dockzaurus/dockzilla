package jobs

import (
	"context"
	"dockzilla/pkg/domain"
	"log/slog"
	"sync"
)

var _ Handler = (*UseCase)(nil)

type UseCase struct {
	logger *slog.Logger

	mu     sync.Mutex
	cancel context.CancelFunc
}

type UCOption interface {
	apply(*UseCase)
}

type ucOptionFunc func(*UseCase)

func (f ucOptionFunc) apply(u *UseCase) { f(u) }

func WithLogger(logger *slog.Logger) UCOption {
	return ucOptionFunc(func(c *UseCase) {
		c.logger = logger
	})
}

// New create a new Jobs UseCase Instance.
func New(opts ...UCOption) (*UseCase, error) {
	uc := new(UseCase)

	for _, opt := range opts {
		opt.apply(uc)
	}

	return uc, nil
}

func (uc *UseCase) Enqueue(ctx context.Context, kind domain.Kind, payload domain.Payload, options ...JobOptions) error {
	uc.logger.InfoContext(ctx, "starting enqueuing job")
	return nil
}

func (uc *UseCase) Ack(ctx context.Context, messages []domain.Message) ([]string, error) {
	uc.logger.InfoContext(ctx, "starting acknowledging job")
	return nil, nil
}

func (uc *UseCase) Dequeue(ctx context.Context, kind domain.Kind) error {
	uc.logger.InfoContext(ctx, "starting dequeuing job")
	return nil
}

func (uc *UseCase) Fail(ctx context.Context, message domain.Message, b bool) error {
	uc.logger.InfoContext(ctx, "starting failing process job")
	return nil
}
