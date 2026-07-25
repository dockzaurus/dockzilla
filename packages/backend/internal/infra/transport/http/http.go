// Package http exposes the HTTP transport of the backend. It is a thin adapter
// around the giniservice HTTP server that satisfies domain.Service, so the
// application core can start and stop it like any other handler.
package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"dockzilla/pkg/domain"
	ginihttp "github.com/zixyos/giniservice/http"
)

// Verify at compile time that a Server is a handler the application can run.
var _ domain.Service = (*Server)(nil)

// Server adapts the giniservice HTTP server to the domain.Service contract.
// The zero value is not usable; build one with NewServer.
type Server struct {
	logger *slog.Logger
	srv    *ginihttp.Server
	cfg    *ginihttp.Config
}

// Option configures a Server during construction. It is an interface rather
// than a bare function type so that options stay comparable in tests and can
// grow behaviour later without breaking callers.
type Option interface {
	apply(s *Server)
}

// optionFunc adapts a plain function to the Option interface.
type optionFunc func(*Server)

func (f optionFunc) apply(s *Server) { f(s) }

// WithLogger sets the structured logger used by the server. It is required:
// NewServer fails when no logger is provided.
func WithLogger(logger *slog.Logger) Option {
	return optionFunc(func(s *Server) {
		s.logger = logger
	})
}

// WithConfig sets the HTTP configuration (listen port, timeouts, telemetry).
// It is required: NewServer fails when no configuration is provided.
func WithConfig(cfg *ginihttp.Config) Option {
	return optionFunc(func(s *Server) {
		s.cfg = cfg
	})
}

// NewServer builds a Server from opts. It returns an error when a required
// option is missing or when the underlying giniservice server cannot be
// created, so a caller never receives a partially initialised Server.
func NewServer(opts ...Option) (*Server, error) {
	s := &Server{}
	for _, opt := range opts {
		opt.apply(s)
	}

	if s.logger == nil {
		return nil, errors.New("http server: logger is required")
	}

	if s.cfg == nil {
		return nil, errors.New("http server: config is required")
	}

	srv, err := ginihttp.NewHTTPServer(
		ginihttp.WithLogger(s.logger),
		ginihttp.WithHTTPServer(s.cfg),
	)
	if err != nil {
		return nil, fmt.Errorf("http server: %w", err)
	}

	s.srv = srv

	return s, nil
}

// Run starts the HTTP server. It returns once the server is listening or fails
// to bind; shutdown is driven by Stop rather than by ctx alone.
func (s *Server) Run(ctx context.Context) error {
	s.logger.InfoContext(ctx, "starting http server", "port", s.cfg.HTTPServer.Port)

	if err := s.srv.Run(ctx); err != nil {
		return fmt.Errorf("run http server: %w", err)
	}

	return nil
}

// Stop gracefully shuts the HTTP server down, honouring ctx's deadline.
func (s *Server) Stop(ctx context.Context) error {
	if err := s.srv.Stop(ctx); err != nil {
		return fmt.Errorf("stop http server: %w", err)
	}

	return nil
}

// Name returns the service name used by the application core and the logs.
func (s *Server) Name() string {
	return s.cfg.ServiceName
}
