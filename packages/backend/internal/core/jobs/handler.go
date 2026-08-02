package jobs

import (
	"context"
	"dockzilla/pkg/domain"

	"github.com/uptrace/bun"
)

type Handler interface {
	Enqueue(context.Context, bun.Tx, domain.Kind, domain.Payload, ...domain.JobOptions) error
	Ack(context.Context, []domain.Message) ([]string, error)
	Dequeue(context.Context, bun.Tx, domain.Kind) error
	Fail(context.Context, domain.Message, bool) error
}
