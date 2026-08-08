package cache_test

import (
	"log/slog"
	"testing"

	"dockzilla/internal/infra/storage/cache"
	"dockzilla/pkg/domain"
	"dockzilla/pkg/storage/redis"
	"github.com/stretchr/testify/require"
)

func TestNewStorage(t *testing.T) {
	t.Parallel()

	type args struct {
		opts []cache.Option
	}
	tests := []struct {
		name     string
		args     args
		wantErr  string
		wantName string
	}{
		{
			name:    "error - no options",
			args:    args{},
			wantErr: "redis cache: logger is required",
		},
		{
			name: "error - logger without config",
			args: args{opts: []cache.Option{
				cache.WithLogger(discardLogger()),
			}},
			wantErr: "redis cache: config is required",
		},
		{
			name: "error - the underlying component rejects an empty url",
			args: args{opts: []cache.Option{
				cache.WithLogger(discardLogger()),
				cache.WithConfig(&redis.Config{}),
			}},
			wantErr: "redis cache: redis: cache URL is required",
		},
		{
			name: "success - default service name",
			args: args{opts: []cache.Option{
				cache.WithLogger(discardLogger()),
				cache.WithConfig(&redis.Config{URL: "redis://localhost:6379/0"}),
			}},
			wantName: "redis-cache",
		},
		{
			name: "success - service name from the config",
			args: args{opts: []cache.Option{
				cache.WithLogger(discardLogger()),
				cache.WithConfig(&redis.Config{
					URL:         "redis://localhost:6379/0",
					ServiceName: "custom",
				}),
			}},
			wantName: "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Building the cache opens no connection: redis.New only prepares
			// the client, so this stays a unit test.
			s, err := cache.NewStorage(tt.args.opts...)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				require.Nil(t, s)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, s)
			require.Equal(t, tt.wantName, s.Name())
			require.NotNil(t, s.Client())
			require.False(t, s.Healthy(), "healthy before Run")
			require.Implements(t, (*domain.Service)(nil), s)
		})
	}
}

func TestStorage_StopWithoutRun(t *testing.T) {
	t.Parallel()

	s, err := cache.NewStorage(
		cache.WithLogger(discardLogger()),
		cache.WithConfig(&redis.Config{URL: "redis://localhost:6379/0"}),
	)
	require.NoError(t, err)

	// The Service contract requires Stop to be safe even when Run never ran.
	require.NoError(t, s.Stop(t.Context()))
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
