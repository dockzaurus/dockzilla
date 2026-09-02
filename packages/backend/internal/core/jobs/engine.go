// Package jobs implements the job engine's enqueue and dispatch surfaces.
package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"dockzilla/pkg/domain"
	errs "dockzilla/pkg/domain/errors"
)

var _ domain.Service = (*Engine)(nil)

// Engine runs the job dispatch loop as a domain.Service. The zero value is not
// usable; build one with NewEngine.
type Engine struct {
	logger *slog.Logger
	uc     *UseCase
	repo   Repository
}

// EngineOption configures an Engine during construction.
type EngineOption interface {
	apply(e *Engine)
}

type engineOptionFunc func(*Engine)

func (f engineOptionFunc) apply(e *Engine) { f(e) }

// WithEngineLogger sets the structured logger.
func WithEngineLogger(logger *slog.Logger) EngineOption {
	return engineOptionFunc(func(e *Engine) {
		e.logger = logger
	})
}

// WithUseCase sets the use case holding the handler registry. Required.
func WithUseCase(uc *UseCase) EngineOption {
	return engineOptionFunc(func(e *Engine) {
		e.uc = uc
		e.repo = uc.repo
	})
}

// NewEngine builds an Engine from opts, returning an error when a required
// option is missing.
func NewEngine(opts ...EngineOption) (*Engine, error) {
	e := &Engine{}

	for _, opt := range opts {
		opt.apply(e)
	}

	if e.logger == nil {
		return nil, errors.New("jobs engine: logger is required")
	}
	if e.uc == nil {
		return nil, errors.New("jobs engine: use case is required")
	}

	return e, nil
}

// Run starts the consume loop in a goroutine and returns immediately, per the
// domain.Service contract.
func (e *Engine) Run(ctx context.Context) error {
	// TODO: spawn pool, call repo.Consume with dispatch callback
	e.logger.InfoContext(ctx, "jobs engine started")
	return nil
}

// Stop cancels the consume loop and waits for in-flight handlers to drain.
func (e *Engine) Stop(ctx context.Context) error {
	// TODO: cancel consume ctx, wait on pool WaitGroup bounded by ctx
	e.logger.InfoContext(ctx, "jobs engine stopped")
	return nil
}

// Name identifies the service in logs.
func (e *Engine) Name() string {
	return "jobs-engine"
}

// Register binds kind to a typed handler with a timeout. The handler receives
// the unmarshalled payload as T. It panics when kind is already registered.
func Register[T any](
	uc *UseCase,
	kind domain.Kind,
	timeout time.Duration,
	fn func(ctx context.Context, args T) error,
) {
	if _, dup := uc.registry[kind]; dup {
		panic(fmt.Sprintf("jobs: duplicate handler for kind %q", kind))
	}
	uc.registry[kind] = entry{
		timeout: timeout,
		run: func(ctx context.Context, payload domain.JobsPayload) error {
			var args T
			if err := json.Unmarshal(payload, &args); err != nil {
				return errs.Terminal(fmt.Errorf("decode %s: %w", kind, err))
			}
			return fn(ctx, args)
		},
	}
}
