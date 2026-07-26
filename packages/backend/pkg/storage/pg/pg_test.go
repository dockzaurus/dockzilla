package pg

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestNew_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		opts       []Option
		wantErrMsg string
	}{
		{
			name:       "no options",
			opts:       []Option{},
			wantErrMsg: "logger is required",
		},
		{
			name: "logger but no config",
			opts: []Option{
				WithLogger(slog.New(slog.DiscardHandler)),
			},
			wantErrMsg: "config is required",
		},
		{
			name: "logger and config but empty URL",
			opts: []Option{
				WithLogger(slog.New(slog.DiscardHandler)),
				WithConfig(&Config{}),
			},
			wantErrMsg: "database URL is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, err := New(tt.opts...)

			if err == nil {
				t.Fatalf("New() = %v, want error", s)
			}
			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("New() error = %v, want to contain %q", err, tt.wantErrMsg)
			}
		})
	}
}

func TestNew_Defaults(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	cfg := &Config{
		URL: "postgres://localhost:5432/test",
	}

	s, err := New(
		WithLogger(logger),
		WithConfig(cfg),
	)
	if err != nil {
		t.Fatalf("New() = %v", err)
	}

	if s.DB().Stats().MaxOpenConnections != 20 {
		t.Errorf("MaxOpenConnections = %d, want 20", s.DB().Stats().MaxOpenConnections)
	}

	if s.Name() != "postgres-storage" {
		t.Errorf("Name() = %q, want %q", s.Name(), "postgres-storage")
	}

	if s.Healthy() {
		t.Error("Healthy() = true, want false (before Run)")
	}
}

func TestNew_Overrides(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	cfg := &Config{
		URL:          "postgres://localhost:5432/test",
		MaxOpenConns: 5,
		ServiceName:  "custom",
	}

	// Keep the original value to check it's not mutated.
	originalMaxIdleConns := cfg.MaxIdleConns

	s, err := New(
		WithLogger(logger),
		WithConfig(cfg),
	)
	if err != nil {
		t.Fatalf("New() = %v", err)
	}

	if s.DB().Stats().MaxOpenConnections != 5 {
		t.Errorf("MaxOpenConnections = %d, want 5", s.DB().Stats().MaxOpenConnections)
	}

	if s.Name() != "custom" {
		t.Errorf("Name() = %q, want %q", s.Name(), "custom")
	}

	// Check caller's config was not mutated.
	if cfg.MaxIdleConns != originalMaxIdleConns {
		t.Errorf("caller's Config.MaxIdleConns was mutated: %d != %d",
			cfg.MaxIdleConns, originalMaxIdleConns)
	}
}

func TestStop_WithoutRun(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	cfg := &Config{
		URL: "postgres://localhost:5432/test",
	}

	s, err := New(
		WithLogger(logger),
		WithConfig(cfg),
	)
	if err != nil {
		t.Fatalf("New() = %v", err)
	}

	err = s.Stop(context.Background())
	if err != nil {
		t.Errorf("Stop() = %v, want nil", err)
	}
}

func TestRun_UnreachableDatabase(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	cfg := &Config{
		URL:          "postgres://127.0.0.1:1/test?sslmode=disable",
		DialTimeout:  100 * time.Millisecond,
		PingInterval: time.Hour,
	}

	s, err := New(
		WithLogger(logger),
		WithConfig(cfg),
	)
	if err != nil {
		t.Fatalf("New() = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = s.Run(ctx)
	if err == nil {
		t.Fatal("Run() = nil, want error")
	}

	if s.Healthy() {
		t.Error("Healthy() = true, want false")
	}

	stopErr := s.Stop(context.Background())
	if stopErr != nil {
		t.Errorf("Stop() = %v, want nil", stopErr)
	}
}
