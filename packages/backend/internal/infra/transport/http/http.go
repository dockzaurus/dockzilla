package http

import (
	"context"
	"log/slog"

	ginihttp "github.com/zixyos/giniservice/http"
)

type Server struct {
	logger *slog.Logger
	srv    *ginihttp.Server
	cfg    *ginihttp.Config
}

type Options func(*Server)

func WithLogger(logger *slog.Logger) Options {
	return func(s *Server) {
		s.logger = logger
	}
}

func WithConfig(cfg *ginihttp.Config) Options {
	return func(s *Server) {
		s.cfg = cfg
	}
}

func NewServer(opts ...Options) *Server {
	s := new(Server)
	for _, opt := range opts {
		opt(s)
	}

	srv, err := ginihttp.NewHTTPServer(
		ginihttp.WithLogger(s.logger),
		ginihttp.WithHTTPServer(s.cfg),
	)

	if err != nil {
		return nil
	}

	s.srv = srv
	return s
}

func (s *Server) Run(ctx context.Context) error {
	s.logger.Info("starting http server", "port", s.cfg.HTTPServer.Port)
	return s.srv.Run(ctx)
}

func (s *Server) Stop(ctx context.Context) error {
	return s.srv.Stop(ctx)
}

func (s *Server) Name() string {
	return s.cfg.ServiceName
}
