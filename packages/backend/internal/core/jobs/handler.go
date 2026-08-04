package jobs

import (
	"context"
	"dockzilla/pkg/domain"
)

type Handler interface {
	Enqueue(context.Context, domain.Kind, domain.Payload, ...JobOptions) error
	Ack(context.Context, []domain.Message) ([]string, error)
	Dequeue(context.Context, domain.Kind) error
	Fail(context.Context, domain.Message, bool) error
}
