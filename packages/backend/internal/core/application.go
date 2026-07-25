// Package core holds the application service: the top-level orchestrator that
// owns every transport handler (HTTP today, workers later) and drives their
// startup and graceful shutdown as a single unit.
package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"dockzilla/pkg/domain"
	serviceloader "github.com/zixyos/goloader/service"
)

// _defaultServiceName is the name reported by an Application that was not given
// one explicitly.
const _defaultServiceName = "dockzilla-application"

// Verify at compile time that an Application is a service the loader can drive.
var _ serviceloader.Service = (*Application)(nil)

// Application is the root service of the backend. It runs every registered
// handler in its own goroutine and coordinates their shutdown when the process
// is asked to terminate.
//
// The zero value is not usable; build one with NewApplication.
type Application struct {
	serviceName string

	handlers []domain.Service

	logger *slog.Logger

	// mu guards serviceID and cancel. Both are written from the goroutine
	// that boots the service and read from the one that stops it.
	mu        sync.RWMutex
	serviceID domain.UUID
	cancel    context.CancelFunc

	wg sync.WaitGroup
}

// Option configures an Application during construction. It is an interface
// rather than a bare function type so that options stay comparable in tests and
// can grow behaviour later without breaking callers.
type Option interface {
	apply(a *Application)
}

// optionFunc adapts a plain function to the Option interface.
type optionFunc func(*Application)

func (f optionFunc) apply(a *Application) { f(a) }

// WithLogger sets the structured logger used by the application.
func WithLogger(logger *slog.Logger) Option {
	return optionFunc(func(a *Application) {
		a.logger = logger
	})
}

// WithApplicationHandler registers handlers to be run by the application. It
// can be passed more than once; handlers accumulate.
func WithApplicationHandler(handlers ...domain.Service) Option {
	return optionFunc(func(a *Application) {
		a.handlers = append(a.handlers, handlers...)
	})
}

// NewApplication returns a new Application configured with opts.
func NewApplication(opts ...Option) *Application {
	a := &Application{serviceName: _defaultServiceName}

	for _, opt := range opts {
		opt.apply(a)
	}

	return a
}

// Run starts every registered handler in its own goroutine and blocks until
// ctx is cancelled, either by its parent or by a call to Stop. A handler that
// fails to start is logged and does not abort the others.
func (a *Application) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)

	a.mu.Lock()
	a.cancel = cancel
	a.mu.Unlock()

	a.logger.InfoContext(ctx, "starting application",
		"name", a.serviceName,
		"identifier", a.id(),
	)

	for _, handler := range a.handlers {
		a.wg.Add(1)

		go func(h domain.Service) {
			defer a.wg.Done()

			a.logger.InfoContext(ctx, "starting service", "name", h.Name())

			if err := h.Run(ctx); err != nil {
				a.logger.WarnContext(ctx, "failed to start service",
					"name", h.Name(),
					"error", err,
				)
			}
		}(handler)
	}

	<-ctx.Done()

	a.logger.InfoContext(ctx, "application run loop exited",
		"name", a.serviceName,
		"identifier", a.id(),
	)

	return nil
}

// Stop gracefully shuts the application down: it cancels the context shared by
// every handler, stops each handler in turn, then waits for their goroutines to
// return. Failures are collected so one broken handler cannot prevent the
// others from stopping.
func (a *Application) Stop(ctx context.Context) error {
	a.logger.InfoContext(ctx, "stopping application",
		"name", a.serviceName,
		"identifier", a.id(),
	)

	a.mu.RLock()
	cancel := a.cancel
	a.mu.RUnlock()

	if cancel != nil {
		cancel()
	}

	var errs []error

	for _, handler := range a.handlers {
		if err := handler.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("stop %s: %w", handler.Name(), err))
		}
	}

	a.wg.Wait()

	return errors.Join(errs...)
}

// SetServiceID records the identifier assigned to this service by the service
// loader. It is safe to call concurrently with Run and Stop.
func (a *Application) SetServiceID(serviceID serviceloader.UUID) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.serviceID = domain.UUID(serviceID)
}

// Name returns the service name.
func (a *Application) Name() string {
	return a.serviceName
}

func (a *Application) id() string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.serviceID.String()
}
