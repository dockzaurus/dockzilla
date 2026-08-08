package jobs

// The handler Register installs is reachable only through the unexported
// registry, so its decode-then-run behaviour is tested from inside the package.
// The mocks package imports jobs, so it cannot be imported back here — none of
// these tests need a repository anyway.

import (
	"context"
	"errors"
	"testing"
	"time"

	"dockzilla/pkg/domain"
	errs "dockzilla/pkg/domain/errors"
	"github.com/stretchr/testify/require"
)

type deployArgs struct {
	DeploymentID string `json:"deployment_id"`
	Replicas     int    `json:"replicas"`
}

func TestRegisterRun(t *testing.T) {
	t.Parallel()

	errHandler := errors.New("pull image: connection refused")

	type args struct {
		payload domain.Payload
		handler func(ctx context.Context, args deployArgs) error
	}
	tests := []struct {
		name         string
		args         args
		wantErr      string
		wantTerminal bool
	}{
		{
			name: "success - payload decoded into the handler's type",
			args: args{
				payload: domain.Payload(`{"deployment_id":"dep-1","replicas":3}`),
				handler: func(_ context.Context, got deployArgs) error {
					want := deployArgs{DeploymentID: "dep-1", Replicas: 3}
					if got != want {
						return errors.New("handler received the wrong arguments")
					}

					return nil
				},
			},
		},
		{
			name: "error - handler failure is returned as-is and stays retryable",
			args: args{
				payload: domain.Payload(`{"deployment_id":"dep-1"}`),
				handler: func(context.Context, deployArgs) error { return errHandler },
			},
			wantErr: "pull image: connection refused",
		},
		{
			name: "error - undecodable payload is terminal",
			args: args{
				payload: domain.Payload(`{"deployment_id":`),
				handler: func(context.Context, deployArgs) error {
					t.Error("handler ran on a payload that failed to decode")

					return nil
				},
			},
			wantErr:      "decode deployment.run:",
			wantTerminal: true,
		},
		{
			name: "error - payload of the wrong shape is terminal",
			args: args{
				payload: domain.Payload(`{"replicas":"three"}`),
				handler: func(context.Context, deployArgs) error {
					t.Error("handler ran on a payload that failed to decode")

					return nil
				},
			},
			wantErr:      "decode deployment.run:",
			wantTerminal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := &UseCase{registry: make(map[domain.Kind]entry)}

			Register(uc, domain.RunDeployment, 30*time.Second, tt.args.handler)

			got, ok := uc.registry[domain.RunDeployment]
			require.True(t, ok, "Register did not install a handler")
			require.Equal(t, 30*time.Second, got.timeout)

			err := got.run(t.Context(), tt.args.payload)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				require.Equal(t, tt.wantTerminal, errs.IsTerminal(err),
					"terminal classification decides dead-letter vs retry")

				return
			}

			require.NoError(t, err)
		})
	}
}
