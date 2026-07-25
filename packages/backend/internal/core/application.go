package core

import (
	"context"
	"dockzilla/pkg/domain"
	"errors"
	"log/slog"
	"sync"

	serviceloader "github.com/zixyos/goloader/service"
)

// Application type represent the Application structure.
type Application struct {
	serviceID   domain.UUID
	serviceName string

	handlers []domain.Service

	logger *slog.Logger

	cancel context.CancelFunc

	mu sync.RWMutex
	wg sync.WaitGroup
}

// ApplicationConfig type represent the configuration injection's function .
type ApplicationConfig func(*Application)

// WithLogger inject the logger to the Application instance.
func WithLogger(logger *slog.Logger) ApplicationConfig {
	return func(a *Application) {
		a.logger = logger
	}
}

// WithApplicationHandler injects the handlers to the application.
func WithApplicationHandler(handlers ...domain.Service) ApplicationConfig {
	return func(a *Application) {
		a.handlers = append(a.handlers, handlers...)
	}
}

// NewApplication return the new Application instance with the needed options.
func NewApplication(opts ...ApplicationConfig) *Application {
	a := new(Application)
	a.serviceName = "dockzilla-application"

	for _, opt := range opts {
		opt(a)
	}

	return a
}

// Run stary the main loop for the application service and handle all services.
func (a *Application) Run(ctx context.Context) error {
	ctx, a.cancel = context.WithCancel(ctx)
	a.logger.InfoContext(ctx, "starting application", "name", a.serviceName, "identifier", a.serviceID.String())

	for _, handler := range a.handlers {
		a.wg.Add(1)
		go func(h domain.Service) {
			defer a.wg.Done()
			a.logger.InfoContext(ctx, "starting service", "name", h.Name())
			if err := h.Run(ctx); err != nil {
				a.logger.WarnContext(ctx, "failed to start service", "name", h.Name(), "err", err)
			}
		}(handler)
	}

	<-ctx.Done()
	a.logger.InfoContext(ctx, "stopping application", "name", a.serviceName, "identifier", a.serviceID)
	return nil
}

// Stop gracefully shutdown the application service and it's children handlers.
func (a *Application) Stop(ctx context.Context) error {
	a.logger.InfoContext(ctx, "stopping application", "name", a.serviceName, "identifier", a.serviceID)
	if a.cancel != nil {
		a.cancel()
	}

	var errs []error
	for _, handler := range a.handlers {
		if err := handler.Stop(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	a.wg.Wait()
	return nil
}

// SetServiceID set the serviceID to the Application Service.
func (a *Application) SetServiceID(serviceID serviceloader.UUID) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	a.serviceID = domain.UUID(serviceID)
}

// Name return the Service name.
func (a *Application) Name() string {
	return a.serviceName
}
