package repository_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"dockzilla/internal/core/jobs"
	"dockzilla/internal/core/jobs/mocks"
	"dockzilla/internal/infra/storage/postgres"
	"dockzilla/internal/infra/storage/postgres/repository"
	"dockzilla/pkg/domain"
	errs "dockzilla/pkg/domain/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestNewJobs(t *testing.T) {
	t.Parallel()

	type args struct {
		opts func(queue jobs.Queue) []repository.JobsOption
	}
	tests := []struct {
		name    string
		args    args
		wantErr string
	}{
		{
			name: "error - no options",
			args: args{
				opts: func(jobs.Queue) []repository.JobsOption { return nil },
			},
			wantErr: "jobs repository: logger is required",
		},
		{
			name: "error - logger without queue",
			args: args{
				opts: func(jobs.Queue) []repository.JobsOption {
					return []repository.JobsOption{repository.JobWithLogger(discardLogger())}
				},
			},
			wantErr: "jobs repository: queue is required",
		},
		{
			name: "success - every required option supplied",
			args: args{
				opts: func(queue jobs.Queue) []repository.JobsOption {
					return []repository.JobsOption{
						repository.JobWithLogger(discardLogger()),
						repository.JobWithQueue(queue),
					}
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			queue := mocks.NewMockQueue(t)

			repo, err := repository.NewJobs(tt.args.opts(queue)...)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				require.Nil(t, repo)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, repo)
			require.Implements(t, (*jobs.Repository)(nil), repo)
		})
	}
}

func TestJobs_Insert(t *testing.T) {
	t.Parallel()

	errSend := errors.New("connection reset")

	type args struct {
		msg    domain.Message
		opts   []domain.JobOption
		withTx bool
	}
	tests := []struct {
		name     string
		args     args
		sendErr  error
		wantSend bool
		wantErr  error
		wantMsg  string
	}{
		{
			name: "success - envelope carries the identifier and the args",
			args: args{
				msg:    newMessage(domain.RunDeployment, `{"deployment_id":"dep-1"}`),
				withTx: true,
			},
			wantSend: true,
		},
		{
			name: "success - supported options do not block the send",
			args: args{
				msg:    newMessage(domain.StartApp, `{"app_id":"app-1"}`),
				opts:   []domain.JobOption{domain.WithMaxAttempts(5)},
				withTx: true,
			},
			wantSend: true,
		},
		{
			name: "error - no ambient transaction",
			args: args{
				msg:    newMessage(domain.StartApp, `{"app_id":"app-1"}`),
				withTx: false,
			},
			wantErr: errs.ErrNoTransaction,
		},
		{
			name: "error - run after is unsupported by pgque",
			args: args{
				msg:    newMessage(domain.StartApp, `{"app_id":"app-1"}`),
				opts:   []domain.JobOption{domain.WithRunAfter(time.Now().Add(time.Hour))},
				withTx: true,
			},
			wantErr: errs.ErrUnsupportedOption,
			wantMsg: "jobs repository: option unsupported by pgque: run after",
		},
		{
			name: "error - unique key is unsupported by pgque",
			args: args{
				msg:    newMessage(domain.StartApp, `{"app_id":"app-1"}`),
				opts:   []domain.JobOption{domain.WithUniqueKey("app-1")},
				withTx: true,
			},
			wantErr: errs.ErrUnsupportedOption,
			wantMsg: "jobs repository: option unsupported by pgque: unique key",
		},
		{
			name: "error - the option check runs before the transaction check",
			args: args{
				msg:    newMessage(domain.StartApp, `{"app_id":"app-1"}`),
				opts:   []domain.JobOption{domain.WithUniqueKey("app-1")},
				withTx: false,
			},
			wantErr: errs.ErrUnsupportedOption,
		},
		{
			name: "error - the substrate rejects the send",
			args: args{
				msg:    newMessage(domain.StopApp, `{"app_id":"app-1"}`),
				withTx: true,
			},
			sendErr:  errSend,
			wantSend: true,
			wantErr:  errSend,
			wantMsg:  "send job app.stop: connection reset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			queue := mocks.NewMockQueue(t)

			if tt.wantSend {
				queue.EXPECT().
					Send(mock.Anything, mock.Anything, string(tt.args.msg.Header.Kind), mock.Anything).
					Run(func(_ context.Context, db bun.IDB, _ string, payload []byte) {
						require.NotNil(t, db, "the send must run on the caller's transaction")

						var env domain.Envelope
						require.NoError(t, json.Unmarshal(payload, &env))
						require.Equal(t, tt.args.msg.Header.Identifier, env.ID)
						require.JSONEq(t, string(tt.args.msg.Payload), string(env.Args))
					}).
					Return(tt.sendErr).
					Once()
			}

			ctx := t.Context()
			if tt.args.withTx {
				ctx = postgres.WithTx(ctx, bun.Tx{})
			}

			err := newRepo(t, queue).Insert(ctx, tt.args.msg, tt.args.opts...)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)

				if tt.wantMsg != "" {
					require.EqualError(t, err, tt.wantMsg)
				}

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestJobs_Consume(t *testing.T) {
	t.Parallel()

	errConsume := errors.New("substrate died")

	tests := []struct {
		name       string
		consumeErr error
		wantErr    string
	}{
		{
			name: "success - one handler registered per kind",
		},
		{
			name:       "error - substrate failure is wrapped",
			consumeErr: errConsume,
			wantErr:    "consume jobs: substrate died",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var registered map[domain.Kind]func(ctx context.Context, payload []byte, attempt int) error

			queue := mocks.NewMockQueue(t)
			queue.EXPECT().
				Consume(mock.Anything, mock.Anything).
				Run(func(
					_ context.Context,
					handlers map[domain.Kind]func(ctx context.Context, payload []byte, attempt int) error,
				) {
					registered = handlers
				}).
				Return(tt.consumeErr).
				Once()

			err := newRepo(t, queue).Consume(t.Context(), func(context.Context, domain.Message) error {
				return nil
			})

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			// A kind with no handler can never be delivered, so the registered
			// set must match the domain's list exactly.
			require.Len(t, registered, len(domain.AllKinds()))

			for _, kind := range domain.AllKinds() {
				require.Contains(t, registered, kind)
			}
		})
	}
}

func TestJobs_ConsumeDispatch(t *testing.T) {
	t.Parallel()

	errDispatch := errors.New("handler failed")

	type args struct {
		kind    domain.Kind
		payload string
		attempt int
	}
	tests := []struct {
		name         string
		args         args
		dispatchErr  error
		wantDispatch bool
		wantMsg      domain.Message
		wantErr      string
		wantTerminal bool
	}{
		{
			name: "success - envelope is decoded into a message",
			args: args{
				kind:    domain.RunDeployment,
				payload: envelope(`{"deployment_id":"dep-1"}`),
				attempt: 0,
			},
			wantDispatch: true,
		},
		{
			name: "success - the substrate's retry count becomes the attempt count",
			args: args{
				kind:    domain.StartApp,
				payload: envelope(`{"app_id":"app-1"}`),
				attempt: 3,
			},
			wantDispatch: true,
		},
		{
			name: "success - a negative retry count floors at zero",
			args: args{
				kind:    domain.StartApp,
				payload: envelope(`{"app_id":"app-1"}`),
				attempt: -1,
			},
			wantDispatch: true,
		},
		{
			name: "error - dispatch failure reaches the substrate for retry",
			args: args{
				kind:    domain.StopApp,
				payload: envelope(`{}`),
			},
			dispatchErr:  errDispatch,
			wantDispatch: true,
			wantErr:      "handler failed",
		},
		{
			name: "error - an undecodable envelope is terminal and never dispatched",
			args: args{
				kind:    domain.RestartApp,
				payload: `{"id":`,
			},
			wantErr:      "decode envelope for app.restart:",
			wantTerminal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var registered map[domain.Kind]func(ctx context.Context, payload []byte, attempt int) error

			queue := mocks.NewMockQueue(t)
			queue.EXPECT().
				Consume(mock.Anything, mock.Anything).
				Run(func(
					_ context.Context,
					handlers map[domain.Kind]func(ctx context.Context, payload []byte, attempt int) error,
				) {
					registered = handlers
				}).
				Return(nil).
				Once()

			var got domain.Message

			dispatched := false
			dispatch := func(_ context.Context, msg domain.Message) error {
				dispatched = true
				got = msg

				return tt.dispatchErr
			}

			require.NoError(t, newRepo(t, queue).Consume(t.Context(), dispatch))

			handler, ok := registered[tt.args.kind]
			require.True(t, ok)

			err := handler(t.Context(), []byte(tt.args.payload), tt.args.attempt)

			require.Equal(t, tt.wantDispatch, dispatched)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				require.Equal(t, tt.wantTerminal, errs.IsTerminal(err),
					"terminal classification decides dead-letter vs retry")

				return
			}

			require.NoError(t, err)

			wantAttempts := uint32(0)
			if tt.args.attempt > 0 {
				wantAttempts = uint32(tt.args.attempt)
			}

			// The kind is taken from the handler it was registered under, not
			// from the wire, so a message never arrives mislabelled.
			require.Equal(t, tt.args.kind, got.Header.Kind)
			require.Equal(t, domain.UUID{0x01, 0x02, 0x03}, got.Header.Identifier)
			require.Equal(t, wantAttempts, got.Attempts)
		})
	}
}

// envelope builds a wire payload with the same marshaller Insert uses, so the
// fixtures cannot drift from the producer's encoding. NOTE: domain.UUID is a
// [16]byte with no MarshalJSON, so the identifier goes over the wire as a JSON
// array of bytes rather than as the hex string UUID.String prints.
func envelope(args string) string {
	body, err := json.Marshal(domain.Envelope{
		ID:   domain.UUID{0x01, 0x02, 0x03},
		Args: domain.Payload(args),
	})
	if err != nil {
		panic(err)
	}

	return string(body)
}

func newMessage(kind domain.Kind, payload string) domain.Message {
	return domain.Message{
		Header: domain.HeaderFrame{
			Identifier: domain.UUID{0x01, 0x02, 0x03},
			Kind:       kind,
		},
		Payload: domain.Payload(payload),
	}
}

func newRepo(t *testing.T, queue jobs.Queue) *repository.Jobs {
	t.Helper()

	repo, err := repository.NewJobs(
		repository.JobWithLogger(discardLogger()),
		repository.JobWithQueue(queue),
	)
	require.NoError(t, err)

	return repo
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
