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

// stubDeployment is the payload the tests decode, in the shape the published
// deployment.run/v1 contract describes.
func stubDeployment() domain.DeployArgsV1 {
	return domain.DeployArgsV1{
		DeploymentID: "70232173-b977-4d0a-aa3b-3a8a52ea0875",
		AppID:        "3f2a1c4e-8b6d-4f21-9c07-5d8e2a1b3c49",
		ImageRef:     "ghcr.io/acme/api:1.4.2",
		TriggeredBy:  "api",
	}
}

func TestRegisterRun(t *testing.T) {
	t.Parallel()

	errHandler := errors.New("pull image: connection refused")

	type args struct {
		payload domain.JobsPayload
		handler func(ctx context.Context, args domain.DeployArgsV1) error
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
				payload: domain.JobsPayload(`{
					"deployment_id":"70232173-b977-4d0a-aa3b-3a8a52ea0875",
					"app_id":"3f2a1c4e-8b6d-4f21-9c07-5d8e2a1b3c49",
					"image_ref":"ghcr.io/acme/api:1.4.2",
					"triggered_by":"api"
				}`),
				handler: func(_ context.Context, got domain.DeployArgsV1) error {
					if got != stubDeployment() {
						return errors.New("handler received the wrong arguments")
					}

					return nil
				},
			},
		},
		{
			name: "error - handler failure is returned as-is and stays retryable",
			args: args{
				payload: domain.JobsPayload(`{"deployment_id":"70232173-b977-4d0a-aa3b-3a8a52ea0875"}`),
				handler: func(context.Context, domain.DeployArgsV1) error { return errHandler },
			},
			wantErr: "pull image: connection refused",
		},
		{
			name: "error - undecodable payload is terminal",
			args: args{
				payload: domain.JobsPayload(`{"deployment_id":`),
				handler: func(context.Context, domain.DeployArgsV1) error {
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
				payload: domain.JobsPayload(`{"deployment_id":42}`),
				handler: func(context.Context, domain.DeployArgsV1) error {
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

func TestRegister_InstallsOneHandlerPerKind(t *testing.T) {
	t.Parallel()

	// A kind with no handler can never be consumed, so every kind the engine
	// advertises must be bindable to the contract generated for it.
	uc := &UseCase{registry: make(map[domain.Kind]entry)}

	Register(uc, domain.RunDeployment, time.Second,
		func(context.Context, domain.DeployArgsV1) error { return nil })
	Register(uc, domain.StartApp, time.Second,
		func(context.Context, domain.StartAppArgsV1) error { return nil })
	Register(uc, domain.StopApp, time.Second,
		func(context.Context, domain.StopAppArgsV1) error { return nil })
	Register(uc, domain.RestartApp, time.Second,
		func(context.Context, domain.RestartAppArgsV1) error { return nil })

	for _, kind := range domain.AllKinds() {
		_, ok := uc.registry[kind]
		require.True(t, ok, "no handler registered for kind %q", kind)
	}

	require.Len(t, uc.registry, len(domain.AllKinds()))
}
