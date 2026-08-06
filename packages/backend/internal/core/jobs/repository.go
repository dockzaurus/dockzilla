package jobs

import (
	"context"

	"dockzilla/pkg/domain"
	"github.com/uptrace/bun"
)

// Repository is the port the job engine depends on. Implementations live in
// infra and adapt a substrate (PgQue, a hand-rolled jobs table) to this shape.
type Repository interface {
	// Insert enqueues msg inside the caller's unit of work. Implementations MUST
	// fail when no transaction is ambient — a silent pool fallback would
	// reintroduce the dual write this design exists to prevent.
	Insert(ctx context.Context, msg domain.Message, opts ...domain.JobOption) error

	// Consume blocks, delivering each message to dispatch. It returns when ctx
	// is cancelled or the substrate dies.
	Consume(ctx context.Context, dispatch domain.Dispatch) error
}

// Queue is the slice of the pgqueue component this adapter uses. Declaring it
// here rather than importing the concrete type keeps the substrate swappable
// and the adapter testable with a fake.
type Queue interface {
	Send(ctx context.Context, db bun.IDB, eventType string, payload []byte) error
	Consume(
		ctx context.Context,
		handlers map[domain.Kind]func(ctx context.Context, payload []byte, attempt int) error,
	) error
}
