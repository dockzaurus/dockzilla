package redis

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
			wantErrMsg: "cache URL is required",
		},
		{
			name: "logger and config with invalid URL",
			opts: []Option{
				WithLogger(slog.New(slog.DiscardHandler)),
				WithConfig(&Config{URL: "not-a-redis-url"}),
			},
			wantErrMsg: "parse redis url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := New(tt.opts...)

			if err == nil {
				t.Fatalf("New() = %v, want error", c)
			}
			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("New() error = %v, want to contain %q", err,
					tt.wantErrMsg)
			}
		})
	}
}

func TestNew_Defaults(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	cfg := &Config{
		URL: "redis://localhost:6379/0",
	}

	c, err := New(
		WithLogger(logger),
		WithConfig(cfg),
	)
	if err != nil {
		t.Fatalf("New() = %v", err)
	}

	if c.Client().Options().DialTimeout != 5*time.Second {
		t.Errorf("DialTimeout = %v, want 5s",
			c.Client().Options().DialTimeout)
	}

	if c.Client().Options().ConnMaxLifetime != 30*time.Minute {
		t.Errorf("ConnMaxLifetime = %v, want 30m",
			c.Client().Options().ConnMaxLifetime)
	}

	if c.Name() != "redis-cache" {
		t.Errorf("Name() = %q, want %q", c.Name(), "redis-cache")
	}

	if c.Healthy() {
		t.Error("Healthy() = true, want false (before Run)")
	}
}

func TestNew_Overrides(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	cfg := &Config{
		URL:         "redis://localhost:6379/0",
		PoolSize:    5,
		ServiceName: "custom",
	}

	// Keep the original value to check it's not mutated.
	originalDialTimeout := cfg.DialTimeout

	c, err := New(
		WithLogger(logger),
		WithConfig(cfg),
	)
	if err != nil {
		t.Fatalf("New() = %v", err)
	}

	if c.Client().Options().PoolSize != 5 {
		t.Errorf("PoolSize = %d, want 5", c.Client().Options().PoolSize)
	}

	if c.Name() != "custom" {
		t.Errorf("Name() = %q, want %q", c.Name(), "custom")
	}

	// Check caller's config was not mutated.
	if cfg.DialTimeout != originalDialTimeout {
		t.Errorf("caller's Config.DialTimeout was mutated: %v != %v",
			cfg.DialTimeout, originalDialTimeout)
	}
}

func TestStop_WithoutRun(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	cfg := &Config{
		URL: "redis://localhost:6379/0",
	}

	c, err := New(
		WithLogger(logger),
		WithConfig(cfg),
	)
	if err != nil {
		t.Fatalf("New() = %v", err)
	}

	err = c.Stop(context.Background())
	if err != nil {
		t.Errorf("Stop() = %v, want nil", err)
	}
}

func TestRun_UnreachableCache(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	cfg := &Config{
		URL:          "redis://127.0.0.1:1/0",
		DialTimeout:  100 * time.Millisecond,
		PingInterval: time.Hour,
	}

	c, err := New(
		WithLogger(logger),
		WithConfig(cfg),
	)
	if err != nil {
		t.Fatalf("New() = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = c.Run(ctx)
	if err == nil {
		t.Fatal("Run() = nil, want error")
	}

	if c.Healthy() {
		t.Error("Healthy() = true, want false")
	}

	stopErr := c.Stop(context.Background())
	if stopErr != nil {
		t.Errorf("Stop() = %v, want nil", stopErr)
	}
}
