package jobs

import (
	"context"
	"dockzilla/pkg/domain"
	"log/slog"
	"time"
)

var _ domain.Service = (*Engine)(nil)

type Engine struct {
	logger *slog.Logger
}

type EngineOption interface {
	apply(*Engine)
}
type engineOptionFunc func(*Engine)

func (f engineOptionFunc) apply(e *Engine) { f(e) }

func EngineWithLogger(logger *slog.Logger) EngineOption {
	return engineOptionFunc(func(e *Engine) {
		e.logger = logger
	})
}

func (e *Engine) Run(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (e *Engine) Stop(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (e *Engine) Name() string {
	//TODO implement me
	panic("implement me")
}

func Register[T any](uc *UseCase, kind domain.Kind, timeout time.Duration, fun func(ctx context.Context, args T) error) error {
	return nil
}
