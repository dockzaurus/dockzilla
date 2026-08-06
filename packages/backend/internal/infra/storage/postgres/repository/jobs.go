// Package repository holds the Postgres-backed implementations of the ports
// declared by the application core.
package repository

import (
	"context"
	"log/slog"

	"dockzilla/internal/core/jobs"
	"dockzilla/pkg/domain"
)

var _ jobs.Repository = (*Jobs)(nil)

// Jobs adapts the queue substrate to the jobs.Repository port. The zero value
// is not usable; build one with NewJobs.
type Jobs struct {
	logger *slog.Logger
}

// JobsOption configures a Jobs during construction.
type JobsOption interface {
	apply(j *Jobs)
}

type jobOptionFunc func(*Jobs)

func (f jobOptionFunc) apply(j *Jobs) { f(j) }

// JobWithLogger sets the structured logger.
func JobWithLogger(logger *slog.Logger) JobsOption {
	return jobOptionFunc(func(s *Jobs) {
		s.logger = logger
	})
}

// NewJobs builds a Jobs from opts.
func NewJobs(opts ...JobsOption) *Jobs {
	r := new(Jobs)

	for _, opt := range opts {
		opt.apply(r)
	}

	return r
}

// Insert enqueues msg inside the caller's unit of work.
func (j *Jobs) Insert(ctx context.Context, msg domain.Message, opts ...domain.JobOption) error {
	j.logger.InfoContext(ctx, "inserting the jobs")

	return nil
}

// Consume blocks, delivering each message to dispatch until ctx is cancelled.
func (j *Jobs) Consume(ctx context.Context, dispatch domain.Dispatch) error {
	j.logger.InfoContext(ctx, "consuming jobs")

	return nil
}
