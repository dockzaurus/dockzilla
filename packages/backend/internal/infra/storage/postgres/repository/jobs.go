// Package repository holds the Postgres-backed implementations of the ports
// declared by the application core.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"dockzilla/internal/core/jobs"
	"dockzilla/internal/infra/storage/postgres"
	"dockzilla/pkg/domain"
)

var _ jobs.Repository = (*Jobs)(nil)

// Jobs adapts the pgque-backed queue to the jobs.Repository port. It owns the
// two things pgqueue deliberately does not know about: this application's
// transaction plumbing and its domain vocabulary. The zero value is not
// usable; build one with NewJobs.
type Jobs struct {
	logger *slog.Logger
	queue  jobs.Queue
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

// JobWithQueue sets the queue substrate. Required.
func JobWithQueue(q jobs.Queue) JobsOption {
	return jobOptionFunc(func(s *Jobs) {
		s.queue = q
	})
}

// NewJobs builds a Jobs from opts, returning an error when a required option
// is missing.
func NewJobs(opts ...JobsOption) (*Jobs, error) {
	r := new(Jobs)

	for _, opt := range opts {
		opt.apply(r)
	}

	if r.logger == nil {
		return nil, errors.New("jobs repository: logger is required")
	}
	if r.queue == nil {
		return nil, errors.New("jobs repository: queue is required")
	}

	return r, nil
}

// Insert enqueues msg on the caller's transaction, so the job and the domain
// write it accompanies commit together or not at all. It fails with
// ErrNoTransaction when no transaction is ambient.
func (j *Jobs) Insert(ctx context.Context, msg domain.Message, opts ...domain.JobOption) error {
	cfg := domain.NewJobConfig(opts...)
	if err := checkSupported(cfg); err != nil {
		return err
	}

	// A nil fallback turns the usual transparent pool fallback into a hard
	// failure, which is what the port requires of every implementation.
	db := postgres.IDB(ctx, nil)
	if db == nil {
		return domain.ErrNoTransaction
	}

	body, err := json.Marshal(domain.Envelope{ID: msg.Header.Identifier, Args: msg.Payload})
	if err != nil {
		return fmt.Errorf("marshal envelope for %s: %w", msg.Header.Kind, err)
	}

	j.logger.DebugContext(ctx, "inserting job",
		"kind", string(msg.Header.Kind),
		"message_id", msg.Header.Identifier.String(),
	)

	if err := j.queue.Send(ctx, db, string(msg.Header.Kind), body); err != nil {
		return fmt.Errorf("send job %s: %w", msg.Header.Kind, err)
	}

	return nil
}

// Consume registers one substrate handler per domain.Kind and funnels every
// delivery into dispatch. It blocks until ctx is cancelled or the substrate
// dies.
func (j *Jobs) Consume(ctx context.Context, dispatch domain.Dispatch) error {
	kinds := domain.AllKinds()
	handlers := make(
		map[domain.Kind]func(ctx context.Context, payload []byte, attempt int) error,
		len(kinds),
	)

	for _, kind := range kinds {
		handlers[kind] = func(ctx context.Context, payload []byte, attempt int) error {
			msg, err := decode(kind, payload, attempt)
			if err != nil {
				return err
			}

			return dispatch(ctx, msg)
		}
	}

	j.logger.InfoContext(ctx, "consuming jobs", "kinds", len(kinds))

	if err := j.queue.Consume(ctx, handlers); err != nil {
		return fmt.Errorf("consume jobs: %w", err)
	}

	return nil
}

func decode(kind domain.Kind, payload []byte, attempt int) (domain.Message, error) {
	var env domain.Envelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return domain.Message{}, fmt.Errorf(
			"decode envelope for %s: %w", kind, jobs.Terminal(err),
		)
	}

	attempts := uint32(0)
	if attempt > 0 {
		attempts = uint32(attempt)
	}

	return domain.Message{
		Header: domain.HeaderFrame{
			Identifier: env.ID,
			Kind:       kind,
		},
		Payload:  env.Args,
		Attempts: attempts,
	}, nil
}

func checkSupported(cfg domain.JobConfig) error {
	if !cfg.RunAfter.IsZero() {
		return fmt.Errorf("%w: run after", domain.ErrUnsupportedOption)
	}

	if cfg.UniqueKey != "" {
		return fmt.Errorf("%w: unique key", domain.ErrUnsupportedOption)
	}

	return nil
}
