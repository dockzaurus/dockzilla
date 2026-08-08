package postgres_test

import (
	"log/slog"
	"testing"

	"dockzilla/internal/infra/storage/postgres"
	"dockzilla/pkg/domain"
	"dockzilla/pkg/storage/pg"

	"github.com/stretchr/testify/require"
)

func TestNewStorage(t *testing.T) {
	t.Parallel()

	type args struct {
		opts []postgres.Option
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
			wantErr: "postgres storage: logger is required",
		},
		{
			name: "error - logger without config",
			args: args{opts: []postgres.Option{
				postgres.WithLogger(discardLogger()),
			}},
			wantErr: "postgres storage: config is required",
		},
		{
			name: "error - the underlying component rejects an empty url",
			args: args{opts: []postgres.Option{
				postgres.WithLogger(discardLogger()),
				postgres.WithConfig(&pg.Config{}),
			}},
			wantErr: "postgres storage: pg: database URL is required",
		},
		{
			name: "success - default service name",
			args: args{opts: []postgres.Option{
				postgres.WithLogger(discardLogger()),
				postgres.WithConfig(&pg.Config{URL: "postgres://localhost:5432/test"}),
			}},
			wantName: "postgres-storage",
		},
		{
			name: "success - service name from the config",
			args: args{opts: []postgres.Option{
				postgres.WithLogger(discardLogger()),
				postgres.WithConfig(&pg.Config{
					URL:         "postgres://localhost:5432/test",
					ServiceName: "custom",
				}),
			}},
			wantName: "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Building the storage opens no connection: pg.New only prepares
			// the pool, so this stays a unit test.
			s, err := postgres.NewStorage(tt.args.opts...)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				require.Nil(t, s)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, s)
			require.Equal(t, tt.wantName, s.Name())
			require.NotNil(t, s.DB())
			require.False(t, s.Healthy(), "healthy before Run")
			require.Implements(t, (*domain.Service)(nil), s)
		})
	}
}

func TestStorage_StopWithoutRun(t *testing.T) {
	t.Parallel()

	s, err := postgres.NewStorage(
		postgres.WithLogger(discardLogger()),
		postgres.WithConfig(&pg.Config{URL: "postgres://localhost:5432/test"}),
	)

	require.NoError(t, err)
	require.NoError(t, s.Stop(t.Context()))
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
