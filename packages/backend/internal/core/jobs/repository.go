package jobs

import (
	"context"

	"dockzilla/pkg/domain"
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
