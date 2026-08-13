package repository_test

import (
	"testing"

	"dockzilla/internal/infra/storage/postgres/repository"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// NOTE: the statements this adapter builds are not covered here. It talks to
// bun directly rather than through a port, so there is no seam to fake, and
// this module has no database-backed test harness yet. Until it does, the
// insert's ON CONFLICT clause and the archived_at filters are verified by
// running the service, not by this package.
func TestNewSchemas(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    func(db bun.IDB) []repository.SchemasOption
		wantErr string
	}{
		{
			name:    "error - no options",
			opts:    func(bun.IDB) []repository.SchemasOption { return nil },
			wantErr: "schemas repository: logger is required",
		},
		{
			name: "error - database missing",
			opts: func(bun.IDB) []repository.SchemasOption {
				return []repository.SchemasOption{
					repository.SchemasWithLogger(discardLogger()),
				}
			},
			wantErr: "schemas repository: database is required",
		},
		{
			name: "success - every required option supplied",
			opts: func(db bun.IDB) []repository.SchemasOption {
				return []repository.SchemasOption{
					repository.SchemasWithLogger(discardLogger()),
					repository.SchemasWithDB(db),
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, err := repository.NewSchemas(tt.opts(new(bun.DB))...)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				require.Nil(t, repo)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, repo)
		})
	}
}
