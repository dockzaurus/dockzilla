package jobs_test

import (
	"context"
	"testing"
	"time"

	"dockzilla/internal/core/jobs"
	"dockzilla/internal/core/jobs/mocks"
	"dockzilla/pkg/domain"

	"github.com/stretchr/testify/require"
)

func TestNewEngine(t *testing.T) {
	t.Parallel()

	type args struct {
		opts func(uc *jobs.UseCase) []jobs.EngineOption
	}
	tests := []struct {
		name    string
		args    args
		wantErr string
	}{
		{
			name: "error - no options",
			args: args{
				opts: func(*jobs.UseCase) []jobs.EngineOption { return nil },
			},
			wantErr: "jobs engine: logger is required",
		},
		{
			name: "error - use case missing",
			args: args{
				opts: func(*jobs.UseCase) []jobs.EngineOption {
					return []jobs.EngineOption{jobs.WithEngineLogger(discardLogger())}
				},
			},
			wantErr: "jobs engine: use case is required",
		},
		{
			name: "success - every required option supplied",
			args: args{
				opts: func(uc *jobs.UseCase) []jobs.EngineOption {
					return []jobs.EngineOption{
						jobs.WithEngineLogger(discardLogger()),
						jobs.WithUseCase(uc),
					}
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := newUseCase(t, mocks.NewMockRepository(t))

			engine, err := jobs.NewEngine(tt.args.opts(uc)...)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				// A partially initialised engine must never escape NewEngine.
				require.Nil(t, engine)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, engine)
		})
	}
}

func TestEngine_Lifecycle(t *testing.T) {
	t.Parallel()

	type args struct {
		run func(ctx context.Context, engine *jobs.Engine) error
	}
	tests := []struct {
		name    string
		args    args
		wantErr string
	}{
		{
			name: "success - run returns immediately",
			args: args{
				run: func(ctx context.Context, engine *jobs.Engine) error {
					return engine.Run(ctx)
				},
			},
		},
		{
			name: "success - stop without run",
			args: args{
				run: func(ctx context.Context, engine *jobs.Engine) error {
					return engine.Stop(ctx)
				},
			},
		},
		{
			name: "success - run then stop",
			args: args{
				run: func(ctx context.Context, engine *jobs.Engine) error {
					if err := engine.Run(ctx); err != nil {
						return err
					}

					return engine.Stop(ctx)
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// The consume loop is not wired yet, so the repository must stay
			// untouched — the mock's expectation assert catches it if that
			// changes without the test following.
			engine := newEngine(t, mocks.NewMockRepository(t))

			err := tt.args.run(t.Context(), engine)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestEngine_Name(t *testing.T) {
	t.Parallel()

	engine := newEngine(t, mocks.NewMockRepository(t))

	require.Equal(t, "jobs-engine", engine.Name())
}

func TestEngine_ImplementsService(t *testing.T) {
	t.Parallel()

	require.Implements(t, (*domain.Service)(nil), newEngine(t, mocks.NewMockRepository(t)))
}

func TestRegister(t *testing.T) {
	t.Parallel()

	type deployArgs struct {
		DeploymentID string `json:"deployment_id"`
	}

	type args struct {
		kinds []domain.Kind
	}
	tests := []struct {
		name      string
		args      args
		wantPanic string
	}{
		{
			name: "success - one handler per kind",
			args: args{
				kinds: []domain.Kind{domain.StartApp, domain.StopApp, domain.RestartApp},
			},
		},
		{
			name: "error - duplicate kind panics",
			args: args{
				kinds: []domain.Kind{domain.RunDeployment, domain.RunDeployment},
			},
			wantPanic: `jobs: duplicate handler for kind "deployment.run"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := newUseCase(t, mocks.NewMockRepository(t))

			register := func() {
				for _, kind := range tt.args.kinds {
					jobs.Register(uc, kind, time.Second,
						func(context.Context, deployArgs) error { return nil },
					)
				}
			}

			if tt.wantPanic != "" {
				require.PanicsWithValue(t, tt.wantPanic, register)

				return
			}

			require.NotPanics(t, register)
		})
	}
}

// newEngine builds an Engine over repo with the logger and use case every test
// shares, failing the test when construction breaks.
func newEngine(t *testing.T, repo jobs.Repository) *jobs.Engine {
	t.Helper()

	engine, err := jobs.NewEngine(
		jobs.WithEngineLogger(discardLogger()),
		jobs.WithUseCase(newUseCase(t, repo)),
	)
	require.NoError(t, err)

	return engine
}
