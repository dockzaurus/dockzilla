// Package handler adapts the core use cases to gin. Each type in it decodes the
// request, calls the use case it was given, and writes the response. It owns no
// paths: the route table that mounts these methods lives in the api package.
package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"dockzilla/internal/core/sample"
	"github.com/gin-gonic/gin"
)

// Sample serves the sample endpoints. The zero value is not usable; build one
// with NewSample.
type Sample struct {
	service sample.Handler
	logger  *slog.Logger
}

// SampleOption configures a Sample during construction. It is an interface rather
// than a bare function type so that options stay comparable in tests and can
// grow behaviour later without breaking callers.
type SampleOption interface {
	apply(s *Sample)
}

// sampleOptionFunc adapts a plain function to the SampleOption interface.
type sampleOptionFunc func(*Sample)

func (f sampleOptionFunc) apply(s *Sample) { f(s) }

// WithHandler sets the use case the handler delegates to. It is required:
// NewSample fails when no service is provided.
func WithHandler(service sample.Handler) SampleOption {
	return sampleOptionFunc(func(s *Sample) {
		s.service = service
	})
}

// WithLogger sets the structured logger used by the handler. It is required:
// NewSample fails when no logger is provided.
func WithLogger(logger *slog.Logger) SampleOption {
	return sampleOptionFunc(func(s *Sample) {
		s.logger = logger
	})
}

// NewSample builds a Sample from opts. It returns an error when a required
// option is missing, so a caller never receives a partially initialised Sample.
func NewSample(opts ...SampleOption) (*Sample, error) {
	s := &Sample{}
	for _, opt := range opts {
		opt.apply(s)
	}

	if s.service == nil {
		return nil, errors.New("sample handler: service is required")
	}

	if s.logger == nil {
		return nil, errors.New("sample handler: logger is required")
	}

	return s, nil
}

// SayHello writes the greeting for the name in the path, or a 500 when the use
// case fails.
func (s *Sample) SayHello(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Param("name")

	message, err := s.service.SayHello(ctx, name)
	if err != nil {
		s.logger.ErrorContext(ctx, "say hello",
			"name", name,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})

		return
	}

	c.JSON(http.StatusOK, gin.H{"message": message})
}

// SendHello delivers a greeting to the name in the path, or writes a 500 when
// the use case fails.
func (s *Sample) SendHello(c *gin.Context) {
	ctx := c.Request.Context()
	name := c.Param("name")

	if err := s.service.SendHello(ctx, name); err != nil {
		s.logger.ErrorContext(ctx, "send hello",
			"name", name,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})

		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "hello sent"})
}
