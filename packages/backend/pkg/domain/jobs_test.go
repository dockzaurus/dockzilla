package domain_test

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"dockzilla/pkg/domain"
	errs "dockzilla/pkg/domain/errors"
	"github.com/stretchr/testify/require"
)

func TestNewPayload(t *testing.T) {
	t.Parallel()

	type args struct {
		body []byte
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "success - json object",
			args: args{body: []byte(`{"app_id":"app-1"}`)},
		},
		{
			name: "success - exactly the maximum size",
			args: args{body: bytes.Repeat([]byte("a"), domain.MaxPayloadSize)},
		},
		{
			name:    "error - empty payload",
			args:    args{body: []byte{}},
			wantErr: errs.ErrPayloadEmpty,
		},
		{
			name:    "error - nil payload",
			args:    args{body: nil},
			wantErr: errs.ErrPayloadEmpty,
		},
		{
			name:    "error - one byte over the maximum size",
			args:    args{body: bytes.Repeat([]byte("a"), domain.MaxPayloadSize+1)},
			wantErr: errs.ErrPayloadTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.NewPayload(tt.args.body)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)

				return
			}

			require.NoError(t, err)
			require.Equal(t, domain.Payload(tt.args.body), got)
		})
	}
}

func TestNewJobConfig(t *testing.T) {
	t.Parallel()

	runAfter := time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC)

	type args struct {
		opts []domain.JobOption
	}
	tests := []struct {
		name string
		args args
		want domain.JobConfig
	}{
		{
			name: "success - defaults when no option is given",
			args: args{},
			want: domain.JobConfig{MaxAttempts: 3},
		},
		{
			name: "success - run after",
			args: args{opts: []domain.JobOption{domain.WithRunAfter(runAfter)}},
			want: domain.JobConfig{MaxAttempts: 3, RunAfter: runAfter},
		},
		{
			name: "success - max attempts overrides the default",
			args: args{opts: []domain.JobOption{domain.WithMaxAttempts(7)}},
			want: domain.JobConfig{MaxAttempts: 7},
		},
		{
			name: "success - unique key",
			args: args{opts: []domain.JobOption{domain.WithUniqueKey("app-1")}},
			want: domain.JobConfig{MaxAttempts: 3, UniqueKey: domain.Key("app-1")},
		},
		{
			name: "success - last option of a kind wins",
			args: args{opts: []domain.JobOption{
				domain.WithMaxAttempts(7),
				domain.WithMaxAttempts(1),
			}},
			want: domain.JobConfig{MaxAttempts: 1},
		},
		{
			name: "success - every option combined",
			args: args{opts: []domain.JobOption{
				domain.WithRunAfter(runAfter),
				domain.WithMaxAttempts(5),
				domain.WithUniqueKey("app-1"),
			}},
			want: domain.JobConfig{
				RunAfter:    runAfter,
				MaxAttempts: 5,
				UniqueKey:   domain.Key("app-1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, domain.NewJobConfig(tt.args.opts...))
		})
	}
}

func TestAllKinds(t *testing.T) {
	t.Parallel()

	kinds := domain.AllKinds()

	// A substrate that routes by message type registers one handler per kind,
	// so a Kind missing from this list can never be consumed.
	require.ElementsMatch(t, []domain.Kind{
		domain.RunDeployment,
		domain.StartApp,
		domain.StopApp,
		domain.RestartApp,
	}, kinds)

	seen := make(map[domain.Kind]bool, len(kinds))
	for _, kind := range kinds {
		require.False(t, seen[kind], "duplicate kind %q", kind)
		seen[kind] = true
	}
}

func TestEnvelope_RoundTrip(t *testing.T) {
	t.Parallel()

	// The identifier travels inside the payload, so Register[T] must still find
	// the producer's arguments untouched under Args after a round trip.
	want := domain.Envelope{
		ID:   domain.UUID{0x01, 0x02, 0x03},
		Args: domain.Payload(`{"deployment_id":"dep-1"}`),
	}

	body, err := json.Marshal(want)
	require.NoError(t, err)

	var got domain.Envelope
	require.NoError(t, json.Unmarshal(body, &got))

	require.Equal(t, want.ID, got.ID)
	require.JSONEq(t, string(want.Args), string(got.Args))
}

func TestUUID_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		uuid domain.UUID
		want string
	}{
		{
			name: "success - zero value",
			uuid: domain.UUID{},
			want: "00000000000000000000000000000000",
		},
		{
			name: "success - hex encoded in order",
			uuid: domain.UUID{
				0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef,
				0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10,
			},
			want: "0123456789abcdeffedcba9876543210",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, tt.uuid.String())
		})
	}
}
