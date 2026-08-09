package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"dockzilla/pkg/domain"
)

var _ Handler = (*UseCase)(nil)

// UseCase implements the job engine's enqueue surface. The zero value is not
// usable; build one with New.
type UseCase struct {
	logger *slog.Logger
	repo   Repository

	generator domain.Generator
	registry  map[domain.Kind]entry
}

type entry struct {
	timeout time.Duration
	run     func(ctx context.Context, payload domain.JobsPayload) error
}

// UCOption configures a UseCase during construction.
type UCOption interface {
	apply(u *UseCase)
}

type ucOptionFunc func(*UseCase)

func (f ucOptionFunc) apply(u *UseCase) { f(u) }

// WithLogger sets the structured logger.
func WithLogger(logger *slog.Logger) UCOption {
	return ucOptionFunc(func(c *UseCase) {
		c.logger = logger
	})
}

// WithRepository sets the queue substrate adapter. Required.
func WithRepository(repo Repository) UCOption {
	return ucOptionFunc(func(c *UseCase) {
		c.repo = repo
	})
}

// WithGenerator sets the uuid generator.
func WithGenerator(generator domain.Generator) UCOption {
	return ucOptionFunc(func(c *UseCase) {
		c.generator = generator
	})
}

// New builds a UseCase from opts, returning an error when a required option is
// missing so a caller never receives a partially initialised UseCase.
func New(opts ...UCOption) (*UseCase, error) {
	uc := &UseCase{
		registry: make(map[domain.Kind]entry),
	}

	for _, opt := range opts {
		opt.apply(uc)
	}

	if uc.logger == nil {
		return nil, errors.New("jobs use case: logger is required")
	}
	if uc.repo == nil {
		return nil, errors.New("jobs use case: repository is required")
	}
	if uc.generator == nil {
		return nil, errors.New("jobs use case: generator is required")
	}
	return uc, nil
}

// Enqueue schedules kind with payload for async execution. It joins the
// caller's transaction, so both the domain write and the enqueue commit
// together or neither commits.
func (uc *UseCase) Enqueue(
	ctx context.Context,
	kind domain.Kind,
	payload domain.JobsPayload,
	opts ...domain.JobOption,
) error {
	identifier := uc.generator()
	msg := domain.Message{
		Header: domain.HeaderFrame{
			Identifier: identifier,
			Kind:       kind,
		},
		Payload: payload,
	}

	uc.logger.DebugContext(ctx, "enqueuing job",
		"kind", string(kind),
		"message_id", msg.Header.Identifier.String(),
	)

	if err := uc.repo.Insert(ctx, msg, opts...); err != nil {
		return fmt.Errorf("insert job: %w", err)
	}

	return nil
}
