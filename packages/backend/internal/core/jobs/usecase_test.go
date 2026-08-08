package jobs_test

import (
	"errors"
	"testing"

	"dockzilla/internal/core/jobs"
	"dockzilla/internal/core/jobs/mocks"
	"dockzilla/pkg/domain"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	type args struct {
		opts func(repo jobs.Repository) []jobs.UCOption
	}
	tests := []struct {
		name    string
		args    args
		wantErr string
	}{
		{
			name: "error - no options",
			args: args{
				opts: func(jobs.Repository) []jobs.UCOption { return nil },
			},
			wantErr: "jobs use case: logger is required",
		},
		{
			name: "error - repository missing",
			args: args{
				opts: func(jobs.Repository) []jobs.UCOption {
					return []jobs.UCOption{jobs.WithLogger(discardLogger())}
				},
			},
			wantErr: "jobs use case: repository is required",
		},
		{
			name: "error - generator missing",
			args: args{
				opts: func(repo jobs.Repository) []jobs.UCOption {
					return []jobs.UCOption{
						jobs.WithLogger(discardLogger()),
						jobs.WithRepository(repo),
					}
				},
			},
			wantErr: "jobs use case: generator is required",
		},
		{
			name: "success - every required option supplied",
			args: args{
				opts: func(repo jobs.Repository) []jobs.UCOption {
					return []jobs.UCOption{
						jobs.WithLogger(discardLogger()),
						jobs.WithRepository(repo),
						jobs.WithGenerator(testID),
					}
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := mocks.NewMockRepository(t)

			uc, err := jobs.New(tt.args.opts(repo)...)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				// A partially initialised use case must never escape New.
				require.Nil(t, uc)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, uc)
		})
	}
}

func TestUseCase_Enqueue(t *testing.T) {
	t.Parallel()

	errInsert := errors.New("no ambient transaction")

	type args struct {
		kind    domain.Kind
		payload domain.Payload
		opts    []domain.JobOption
	}
	tests := []struct {
		name      string
		args      args
		insertErr error
		wantErr   string
	}{
		{
			name: "success - message carries the generated id, kind and payload",
			args: args{
				kind:    domain.RunDeployment,
				payload: domain.Payload(`{"deployment_id":"dep-1"}`),
			},
		},
		{
			name: "success - job options reach the repository untouched",
			args: args{
				kind:    domain.StartApp,
				payload: domain.Payload(`{"app_id":"app-1"}`),
				opts: []domain.JobOption{
					domain.WithMaxAttempts(5),
					domain.WithUniqueKey(domain.Key("app-1")),
				},
			},
		},
		{
			name: "error - repository insert fails",
			args: args{
				kind:    domain.StopApp,
				payload: domain.Payload(`{"app_id":"app-1"}`),
			},
			insertErr: errInsert,
			wantErr:   "insert job: no ambient transaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := mocks.NewMockRepository(t)
			uc := newUseCase(t, repo)

			wantMsg := domain.Message{
				Header: domain.HeaderFrame{
					Identifier: testID(),
					Kind:       tt.args.kind,
				},
				Payload: tt.args.payload,
			}

			// The generated mock only passes the variadic slice through to
			// Called when the caller supplied one, so the expectation has to
			// match that arity exactly.
			var wantOpts []any
			if len(tt.args.opts) > 0 {
				wantOpts = append(wantOpts, tt.args.opts)
			}

			repo.EXPECT().
				Insert(mock.Anything, wantMsg, wantOpts...).
				Return(tt.insertErr).
				Once()

			err := uc.Enqueue(t.Context(), tt.args.kind, tt.args.payload, tt.args.opts...)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				require.ErrorIs(t, err, tt.insertErr)

				return
			}

			require.NoError(t, err)
		})
	}
}
