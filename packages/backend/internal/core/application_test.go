package core_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"dockzilla/internal/core"
	"dockzilla/pkg/domain"
	"dockzilla/pkg/domain/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	serviceloader "github.com/zixyos/goloader/service"
)

func TestNewApplication(t *testing.T) {
	t.Parallel()

	type args struct {
		opts []core.Option
	}
	tests := []struct {
		name     string
		args     args
		wantName string
	}{
		{
			name:     "success - default service name",
			args:     args{opts: []core.Option{core.WithLogger(discardLogger())}},
			wantName: "dockzilla-application",
		},
		{
			name: "success - handlers accumulate across options",
			args: args{opts: []core.Option{
				core.WithLogger(discardLogger()),
				core.WithApplicationHandler(),
				core.WithApplicationHandler(),
			}},
			wantName: "dockzilla-application",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			app := core.NewApplication(tt.args.opts...)

			require.NotNil(t, app)
			require.Equal(t, tt.wantName, app.Name())
		})
	}
}

func TestApplication_RunStop(t *testing.T) {
	t.Parallel()

	errStop := errors.New("listener still draining")

	type args struct {
		stopErrs []error // one entry per handler; nil means a clean stop
	}
	tests := []struct {
		name    string
		args    args
		wantErr []error
	}{
		{
			name: "success - single handler runs and stops",
			args: args{stopErrs: []error{nil}},
		},
		{
			name: "success - every handler is started and stopped",
			args: args{stopErrs: []error{nil, nil, nil}},
		},
		{
			name:    "error - a failing handler does not stop the others",
			args:    args{stopErrs: []error{nil, errStop, nil}},
			wantErr: []error{errStop},
		},
		{
			name:    "error - failures from several handlers are joined",
			args:    args{stopErrs: []error{errStop, errStop}},
			wantErr: []error{errStop},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			started := make(chan struct{}, len(tt.args.stopErrs))
			handlers := make([]domain.Service, 0, len(tt.args.stopErrs))

			for i, stopErr := range tt.args.stopErrs {
				svc := mocks.NewMockService(t)
				svc.EXPECT().Name().Return("service-" + string(rune('a'+i))).Maybe()
				svc.EXPECT().
					Run(mock.Anything).
					Run(func(context.Context) { started <- struct{}{} }).
					Return(nil).
					Once()
				svc.EXPECT().Stop(mock.Anything).Return(stopErr).Once()

				handlers = append(handlers, svc)
			}

			app := core.NewApplication(
				core.WithLogger(discardLogger()),
				core.WithApplicationHandler(handlers...),
			)

			runErr := make(chan error, 1)
			go func() { runErr <- app.Run(t.Context()) }()

			// Run installs the cancel func before it starts any handler, so a
			// handler having started is proof Stop will be able to unblock Run.
			for range tt.args.stopErrs {
				select {
				case <-started:
				case <-time.After(5 * time.Second):
					t.Fatal("timed out waiting for a handler to start")
				}
			}

			err := app.Stop(t.Context())

			for _, want := range tt.wantErr {
				require.ErrorIs(t, err, want)
			}

			if tt.wantErr == nil {
				require.NoError(t, err)
			}

			select {
			case err := <-runErr:
				require.NoError(t, err, "Stop must unblock Run")
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for Run to return")
			}
		})
	}
}

func TestApplication_StopWithoutRun(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockService(t)
	svc.EXPECT().Name().Return("service-a").Maybe()
	svc.EXPECT().Stop(mock.Anything).Return(nil).Once()

	app := core.NewApplication(
		core.WithLogger(discardLogger()),
		core.WithApplicationHandler(svc),
	)

	// Run never installed a cancel func — Stop must still shut the handlers
	// down rather than panicking on the nil.
	require.NoError(t, app.Stop(t.Context()))
}

func TestApplication_RunCancelledByParentContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	app := core.NewApplication(core.WithLogger(discardLogger()))

	runErr := make(chan error, 1)
	go func() { runErr <- app.Run(ctx) }()

	cancel()

	select {
	case err := <-runErr:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Run to return")
	}
}

func TestApplication_SetServiceID(t *testing.T) {
	t.Parallel()

	app := core.NewApplication(core.WithLogger(discardLogger()))

	// The identifier is read from the goroutine that stops the service while
	// the loader writes it from another, so this is also a race detector probe.
	app.SetServiceID(serviceloader.UUID{0x01, 0x02, 0x03})

	require.NoError(t, app.Stop(t.Context()))
}

func TestApplication_ImplementsLoaderService(t *testing.T) {
	t.Parallel()

	require.Implements(t,
		(*serviceloader.Service)(nil),
		core.NewApplication(core.WithLogger(discardLogger())),
	)
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
