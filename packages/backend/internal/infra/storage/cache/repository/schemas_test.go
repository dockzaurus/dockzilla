package repository_test

import (
	"log/slog"
	"testing"

	"dockzilla/internal/infra/storage/cache/repository"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// NOTE: the round trip through Redis is not covered here. This module has no
// cache-backed test harness yet, and the adapter's contract — a failure is
// indistinguishable from a miss — is exercised against the use case in
// internal/core/jobs/schemas instead.
func TestNewSchemas(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    func(client *goredis.Client) []repository.SchemasOption
		wantErr string
	}{
		{
			name:    "error - no options",
			opts:    func(*goredis.Client) []repository.SchemasOption { return nil },
			wantErr: "schemas cache: logger is required",
		},
		{
			name: "error - client missing",
			opts: func(*goredis.Client) []repository.SchemasOption {
				return []repository.SchemasOption{
					repository.SchemasWithLogger(discardLogger()),
				}
			},
			wantErr: "schemas cache: client is required",
		},
		{
			name: "success - every required option supplied",
			opts: func(client *goredis.Client) []repository.SchemasOption {
				return []repository.SchemasOption{
					repository.SchemasWithLogger(discardLogger()),
					repository.SchemasWithClient(client),
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:6379"})
			t.Cleanup(func() { _ = client.Close() })

			cache, err := repository.NewSchemas(tt.opts(client)...)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				require.Nil(t, cache)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, cache)
		})
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
