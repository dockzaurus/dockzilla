// Package sample holds the sample use case: a placeholder greeting service that
// exercises the wiring from the HTTP transport down to the application core. It
// knows nothing about HTTP, gin, or any other transport.
package sample

import (
	"context"
	"errors"
	"log/slog"
)

var _ Handler = (*UseCase)(nil)

// UseCase implements the sample greeting logic. The zero value is not usable;
// build one with New.
type UseCase struct {
	logger *slog.Logger
}

// Option configures a UseCase during construction. It is an interface rather
// than a bare function type so that options stay comparable in tests and can
// grow behaviour later without breaking callers.
type Option interface {
	apply(u *UseCase)
}

// optionFunc adapts a plain function to the Option interface.
type optionFunc func(*UseCase)

func (f optionFunc) apply(u *UseCase) { f(u) }

// WithLogger sets the structured logger used by the use case. It is required:
// New fails when no logger is provided.
func WithLogger(logger *slog.Logger) Option {
	return optionFunc(func(u *UseCase) {
		u.logger = logger
	})
}

// New builds a UseCase from opts. It returns an error when a required option is
// missing, so a caller never receives a partially initialised UseCase.
func New(opts ...Option) (*UseCase, error) {
	u := &UseCase{}
	for _, opt := range opts {
		opt.apply(u)
	}

	if u.logger == nil {
		return nil, errors.New("sample use case: logger is required")
	}

	return u, nil
}

// SayHello returns the greeting addressed to name.
func (u *UseCase) SayHello(ctx context.Context, name string) (string, error) {
	u.logger.InfoContext(ctx, "saying hello", "name", name)

	return "Hello, " + name, nil
}

// SendHello delivers a greeting to name. It is a placeholder: nothing leaves
// the process yet.
func (u *UseCase) SendHello(ctx context.Context, name string) error {
	u.logger.InfoContext(ctx, "sending hello", "name", name)

	return nil
}
