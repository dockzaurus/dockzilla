package pgqueue_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"dockzilla/pkg/queue/pgqueue"
	"github.com/NikolayS/pgque-go"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	type args struct {
		opts []pgqueue.Option
	}
	tests := []struct {
		name    string
		args    args
		wantErr string
	}{
		{
			name:    "error - no options",
			args:    args{},
			wantErr: "pgqueue: logger is required",
		},
		{
			name: "error - queue name missing",
			args: args{opts: []pgqueue.Option{
				pgqueue.WithLogger(discardLogger()),
			}},
			wantErr: "pgqueue: queue name is required",
		},
		{
			name: "error - empty queue name is not a name",
			args: args{opts: []pgqueue.Option{
				pgqueue.WithLogger(discardLogger()),
				pgqueue.WithQueue(""),
			}},
			wantErr: "pgqueue: queue name is required",
		},
		{
			name: "error - consumer name missing",
			args: args{opts: []pgqueue.Option{
				pgqueue.WithLogger(discardLogger()),
				pgqueue.WithQueue("dckz"),
			}},
			wantErr: "pgqueue: consumer name is required",
		},
		{
			name: "error - client missing",
			args: args{opts: []pgqueue.Option{
				pgqueue.WithLogger(discardLogger()),
				pgqueue.WithQueue("dckz"),
				pgqueue.WithConsumer("dckz-worker"),
			}},
			wantErr: "pgqueue: client is required",
		},
		{
			name: "success - every required option supplied",
			args: args{opts: []pgqueue.Option{
				pgqueue.WithLogger(discardLogger()),
				pgqueue.WithQueue("dckz"),
				pgqueue.WithConsumer("dckz-worker"),
				pgqueue.WithClient(&pgque.Client{}),
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			q, err := pgqueue.New(tt.args.opts...)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				require.Nil(t, q)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, q)
		})
	}
}

func TestQueue_RunTickerStopsWithTheContext(t *testing.T) {
	t.Parallel()

	q, err := pgqueue.New(
		pgqueue.WithLogger(discardLogger()),
		pgqueue.WithQueue("dckz"),
		pgqueue.WithConsumer("dckz-worker"),
		pgqueue.WithClient(&pgque.Client{}),
		// Long enough that the loop is guaranteed to observe the cancellation
		// before it ever reaches the substrate.
		pgqueue.WithTick(time.Hour),
	)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	tickerErr := make(chan error, 1)
	go func() { tickerErr <- q.RunTicker(ctx) }()

	select {
	case err := <-tickerErr:
		require.ErrorIs(t, err, context.Canceled)
		require.EqualError(t, err, "run ticker: context canceled")
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for RunTicker to return")
	}
}

func TestQueue_Client(t *testing.T) {
	t.Parallel()

	client := &pgque.Client{}

	q, err := pgqueue.New(
		pgqueue.WithLogger(discardLogger()),
		pgqueue.WithQueue("dckz"),
		pgqueue.WithConsumer("dckz-worker"),
		pgqueue.WithClient(client),
	)
	require.NoError(t, err)

	require.Same(t, client, q.Client())
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
