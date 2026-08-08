package sample_test

import (
	"log/slog"
	"testing"

	"dockzilla/internal/core/sample"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	type args struct {
		opts []sample.Option
	}
	tests := []struct {
		name    string
		args    args
		wantErr string
	}{
		{
			name:    "error - no options",
			args:    args{},
			wantErr: "sample use case: logger is required",
		},
		{
			name: "success - logger supplied",
			args: args{opts: []sample.Option{
				sample.WithLogger(slog.New(slog.DiscardHandler)),
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc, err := sample.New(tt.args.opts...)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				require.Nil(t, uc)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, uc)
		})
	}
}

func TestUseCase_SayHello(t *testing.T) {
	t.Parallel()

	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "success - greets the name",
			args: args{name: "killian"},
			want: "Hello, killian",
		},
		{
			name: "success - empty name",
			args: args{name: ""},
			want: "Hello, ",
		},
		{
			name: "success - name is not escaped or trimmed",
			args: args{name: "  <b>ada</b> "},
			want: "Hello,   <b>ada</b> ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := newUseCase(t)

			got, err := uc.SayHello(t.Context(), tt.args.name)

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestUseCase_SendHello(t *testing.T) {
	t.Parallel()

	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "success - named recipient",
			args: args{name: "killian"},
		},
		{
			name: "success - empty recipient",
			args: args{name: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := newUseCase(t)

			// Nothing leaves the process yet, so the contract under test is
			// only that the placeholder never fails its caller.
			require.NoError(t, uc.SendHello(t.Context(), tt.args.name))
		})
	}
}

func newUseCase(t *testing.T) *sample.UseCase {
	t.Helper()

	uc, err := sample.New(sample.WithLogger(slog.New(slog.DiscardHandler)))
	require.NoError(t, err)

	return uc
}
