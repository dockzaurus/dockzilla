package jobs

import (
	"context"

	"dockzilla/pkg/domain"
)

// Handler is the slice of the job engine that producing use cases import to
// schedule async work. The consumer declares it, so the transport never
// depends on more of the core than it uses.
type Handler interface {
	// Enqueue schedules work inside the caller's transaction. It must fail when
	// no transaction is ambient — a silent pool fallback would reintroduce the
	// dual write this design exists to prevent.
	Enqueue(
		ctx context.Context,
		kind domain.Kind,
		payload domain.JobsPayload,
		opts ...domain.JobOption,
	) error
}
