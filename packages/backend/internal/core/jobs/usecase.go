package jobs

import (
	"context"
	"dockzilla/pkg/domain"
	"log/slog"
	"sync"

	"github.com/uptrace/bun"
)

var _ Handler = (*UseCase)(nil)

type UseCase struct {
	logger *slog.Logger

	mu     sync.Mutex
	cancel context.CancelFunc
}

func New() *UseCase {
	return &UseCase{}
}

func (uc *UseCase) Enqueue(ctx context.Context, tx bun.Tx, kind domain.Kind, payload domain.Payload, options ...domain.JobOptions) error {
	//TODO implement me
	panic("implement me")
}

func (uc *UseCase) Ack(ctx context.Context, messages []domain.Message) ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func (uc *UseCase) Dequeue(ctx context.Context, tx bun.Tx, kind domain.Kind) error {
	//TODO implement me
	panic("implement me")
}

func (uc *UseCase) Fail(ctx context.Context, message domain.Message, b bool) error {
	//TODO implement me
	panic("implement me")
}
