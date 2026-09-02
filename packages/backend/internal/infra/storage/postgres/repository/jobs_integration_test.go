package repository_test

// Integration tests for the jobs repository, run against a real Postgres with
// the PgQue schema installed.
//
// They exist because the property the port is built around — an enqueue lives
// or dies with the transaction that made it — is one a fake cannot check. A
// fake queue rolls back a write it never made, so it agrees with every
// assertion below while the real substrate could be committing on its own
// connection and losing nothing but the deployment row.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"dockzilla/internal/infra/storage/postgres"
	"dockzilla/pkg/domain"
	errs "dockzilla/pkg/domain/errors"
	"dockzilla/pkg/queue/pgqueue"
	"github.com/NikolayS/pgque-go"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// _testQueuePrefix names every queue these tests create, so a leaked one is
// recognisable in a database someone is also using by hand.
const _testQueuePrefix = "dckz_test_"

// TestJobs_Insert_TransactionBoundary is the test the port exists for.
//
// The commit case is not decoration. Drop it and a Send that silently wrote
// nothing would still satisfy the rollback assertion, leaving a green test that
// proves only that an empty queue is empty.
func TestJobs_Insert_TransactionBoundary(t *testing.T) {
	t.Parallel()

	errRollback := errors.New("caller changed its mind")

	tests := []struct {
		name      string
		txErr     error
		wantCount int
	}{
		{
			name:      "commit - the job is in the queue",
			wantCount: 1,
		},
		{
			name:      "rollback - the queue is left empty",
			txErr:     errRollback,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dsn := testDSN(t)
			db := openTestDB(t, dsn)
			queue := createTestQueue(t, db)
			repo := newRepo(t, newTestQueue(t, dsn, queue))

			err := postgres.NewTransactor(db).
				RunInTx(t.Context(), func(ctx context.Context, _ bun.Tx) error {
					if err := repo.Insert(ctx, newMessage(
						domain.RunDeployment, `{"deployment_id":"dep-1"}`,
					)); err != nil {
						return err
					}

					return tt.txErr
				})

			if tt.txErr != nil {
				require.ErrorIs(t, err, tt.txErr)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.wantCount, countEvents(t, db, queue))
		})
	}
}

// TestJobs_Insert_WithoutTransactionWritesNothing pins the guard against the
// real substrate. The unit test already asserts the error; what matters here is
// that nothing reached the queue on the way to returning it.
func TestJobs_Insert_WithoutTransactionWritesNothing(t *testing.T) {
	t.Parallel()

	dsn := testDSN(t)
	db := openTestDB(t, dsn)
	queue := createTestQueue(t, db)
	repo := newRepo(t, newTestQueue(t, dsn, queue))

	// t.Context() carries no transaction, which is the whole point.
	err := repo.Insert(t.Context(), newMessage(
		domain.RunDeployment, `{"deployment_id":"dep-1"}`,
	))

	require.ErrorIs(t, err, errs.ErrNoTransaction)
	require.Zero(t, countEvents(t, db, queue))
}

// testDSN returns the database to test against, skipping when none is
// configured. The skip names the variable to set: an integration test that
// quietly passes because it never ran is worse than not having one.
func testDSN(t *testing.T) string {
	t.Helper()

	for _, key := range []string{"DOCKZILLA_TEST_DATABASE_URL", "DATABASE_URL"} {
		if dsn := os.Getenv(key); dsn != "" {
			return dsn
		}
	}

	t.Skip("set DOCKZILLA_TEST_DATABASE_URL (or DATABASE_URL) to run the jobs queue integration tests")

	return ""
}

// openTestDB dials dsn and checks the PgQue schema is installed. A missing
// schema fails rather than skips: the caller pointed us at a database on
// purpose, so staying quiet would hide the one thing worth knowing.
func openTestDB(t *testing.T, dsn string) *bun.DB {
	t.Helper()

	db := bun.NewDB(
		sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn))),
		pgdialect.New(),
	)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	require.NoError(t, db.PingContext(ctx), "cannot reach the test database")

	var version string
	require.NoError(t,
		db.NewRaw("SELECT pgque.version()").Scan(ctx, &version),
		"pgque schema is not installed — run: task backend:pgque:install",
	)

	return db
}

// createTestQueue creates a queue owned by this test alone and drops it
// afterwards, so parallel subtests never count each other's events.
func createTestQueue(t *testing.T, db *bun.DB) string {
	t.Helper()

	// PgQue caps queue names at 57 bytes (the pg_notify limit), which the
	// prefix plus a nanosecond timestamp stays well inside.
	name := fmt.Sprintf("%s%d", _testQueuePrefix, time.Now().UnixNano())

	var queueID int
	require.NoError(t,
		db.NewRaw("SELECT pgque.create_queue(?)", name).Scan(t.Context(), &queueID),
	)

	t.Cleanup(func() {
		// t.Context() is already cancelled by the time cleanups run, so the
		// drop needs a context of its own or it never reaches the server.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var dropped int
		if err := db.NewRaw("SELECT pgque.drop_queue(?, true)", name).
			Scan(ctx, &dropped); err != nil {
			t.Logf("drop test queue %s: %v", name, err)
		}
	})

	return name
}

// newTestQueue builds the real pgqueue substrate pointed at name.
func newTestQueue(t *testing.T, dsn, name string) *pgqueue.Queue {
	t.Helper()

	// Send runs on the caller's transaction and never touches this client, but
	// the constructor requires one for Consume and RunTicker.
	client, err := pgque.Connect(t.Context(), dsn)
	require.NoError(t, err)
	t.Cleanup(client.Close)

	queue, err := pgqueue.New(
		pgqueue.WithLogger(discardLogger()),
		pgqueue.WithQueue(name),
		pgqueue.WithConsumer("dckz-test"),
		pgqueue.WithClient(client),
	)
	require.NoError(t, err)

	return queue
}

// countEvents counts everything sitting in the queue's data tables. PgQue keeps
// them in an inheritance tree under queue_data_pfx, so selecting from the
// parent sees the rotated children too.
func countEvents(t *testing.T, db *bun.DB, queue string) int {
	t.Helper()

	ctx := t.Context()

	var prefix string
	require.NoError(t,
		db.NewRaw(
			"SELECT queue_data_pfx FROM pgque.queue WHERE queue_name = ?", queue,
		).Scan(ctx, &prefix),
	)

	// prefix is a table name PgQue generated and stored itself, not caller
	// input, and it cannot be bound as a parameter.
	var count int
	require.NoError(t, db.NewRaw("SELECT count(*) FROM "+prefix).Scan(ctx, &count))

	return count
}
