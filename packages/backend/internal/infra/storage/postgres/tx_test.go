package postgres_test

import (
	"context"
	"testing"

	"dockzilla/internal/infra/storage/postgres"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestIDB(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx      func(t *testing.T) context.Context
		fallback bun.IDB
	}
	tests := []struct {
		name        string
		args        args
		wantAmbient bool
		wantNil     bool
	}{
		{
			name: "success - the ambient transaction wins over the fallback",
			args: args{
				ctx: func(t *testing.T) context.Context {
					t.Helper()

					return postgres.WithTx(t.Context(), bun.Tx{})
				},
				fallback: (*bun.DB)(nil),
			},
			wantAmbient: true,
		},
		{
			name: "success - the fallback is used outside a transaction",
			args: args{
				ctx: func(t *testing.T) context.Context {
					t.Helper()

					return t.Context()
				},
				fallback: (*bun.DB)(nil),
			},
		},
		{
			name: "success - a nil fallback turns 'no transaction' into a hard failure",
			args: args{
				ctx: func(t *testing.T) context.Context {
					t.Helper()

					return t.Context()
				},
				fallback: nil,
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := postgres.IDB(tt.args.ctx(t), tt.args.fallback)

			if tt.wantNil {
				require.Nil(t, got)

				return
			}

			if tt.wantAmbient {
				require.IsType(t, bun.Tx{}, got)

				return
			}

			require.Equal(t, tt.args.fallback, got)
		})
	}
}

func TestWithTx_DoesNotLeakAcrossContexts(t *testing.T) {
	t.Parallel()

	base := t.Context()
	withTx := postgres.WithTx(base, bun.Tx{})

	// A sibling context must not see the transaction stored on another one, or
	// an unrelated repository call would silently join someone else's tx.
	require.Nil(t, postgres.IDB(base, nil))
	require.NotNil(t, postgres.IDB(withTx, nil))

	// Nesting overwrites rather than merges: the innermost tx is the one a
	// repository joins.
	require.NotNil(t, postgres.IDB(postgres.WithTx(withTx, bun.Tx{}), nil))
}

func TestNewTransactor(t *testing.T) {
	t.Parallel()

	// A Transactor over a nil handle is still constructible; it fails on use,
	// which is a database-backed concern and belongs in an e2e test.
	require.NotNil(t, postgres.NewTransactor(nil))
}
