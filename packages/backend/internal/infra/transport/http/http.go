// Package http exposes the HTTP transport of the backend. It is a thin adapter
// around the giniservice HTTP server that satisfies domain.Service, so the
// application core can start and stop it like any other handler.
package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	ginihttp "github.com/zixyos/giniservice/http"
)

// Server adapts the giniservice HTTP server to the domain.Service contract.
// The zero value is not usable; build one with NewServer.
type Server struct {
	logger *slog.Logger
	srv    *ginihttp.Server
	cfg    *ginihttp.Config
}

// Option configures a Server during construction.
type Option func(*Server)

// WithLogger sets the structured logger used by the server. It is required:
// NewServer fails when no logger is provided.
func WithLogger(logger *slog.Logger) Option {
	return func(s *Server) {
		s.logger = logger
	}
}

// WithConfig sets the HTTP configuration (listen port, timeouts, telemetry).
// It is required: NewServer fails when no configuration is provided.
func WithConfig(cfg *ginihttp.Config) Option {
	return func(s *Server) {
		s.cfg = cfg
	}
}

// NewServer builds a Server from opts. It returns an error when a required
// option is missing or when the underlying giniservice server cannot be
// created, so a caller never receives a partially initialised Server.
func NewServer(opts ...Option) (*Server, error) {
	s := new(Server)
	for _, opt := range opts {
		opt(s)
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
