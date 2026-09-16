package domain_test

import (
	"bytes"
	"encoding/json"
	"fmt"
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
			require.Equal(t, domain.JobsPayload(tt.args.body), got)
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
		Args: domain.JobsPayload(`{"deployment_id":"dep-1"}`),
	}

	body, err := json.Marshal(want)
	require.NoError(t, err)

	var got domain.Envelope
	require.NoError(t, json.Unmarshal(body, &got))

	require.Equal(t, want.ID, got.ID)
	require.JSONEq(t, string(want.Args), string(got.Args))
}

func TestUUID_ValueSatisfiesStringer(t *testing.T) {
	t.Parallel()

	// UUID must implement fmt.Stringer as a value so logging a UUID value
	// renders its canonical form rather than its underlying byte array.
	require.Implements(t, (*fmt.Stringer)(nil), domain.UUID{})
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
			want: "00000000-0000-0000-0000-000000000000",
		},
		{
			name: "success - canonical dashed form",
			uuid: domain.UUID{
				0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef,
				0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10,
			},
			want: "01234567-89ab-cdef-fedc-ba9876543210",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, tt.uuid.String())
		})
	}
}

func TestUUID_TextRoundTrip(t *testing.T) {
	t.Parallel()

	// A UUID crosses the wire as a string, so ParseUUID must read back what
	// MarshalText writes byte for byte.
	want := domain.UUID{
		0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef,
		0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10,
	}

	text, err := want.MarshalText()
	require.NoError(t, err)
	require.Equal(t, want.String(), string(text))

	got, err := domain.ParseUUID(string(text))
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestUUID_Parse_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		text    string
		wantErr string
	}{
		{
			name:    "error - too short",
			text:    "0123456789abcdef",
			wantErr: "want 36",
		},
		{
			name:    "error - dashless",
			text:    "0123456789abcdeffedcba9876543210ffff",
			wantErr: "canonical dashed form",
		},
		{
			name:    "error - non-hex digit",
			text:    "0123456z-89ab-cdef-fedc-ba9876543210",
			wantErr: "uuid:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.ParseUUID(tt.text)

			require.ErrorContains(t, err, tt.wantErr)
			require.Equal(t, domain.UUID{}, got)
		})
	}
}

func TestUUID_JSONUnmarshal(t *testing.T) {
	t.Parallel()

	// The decode half of TestUUID_JSONMarshal: encoding/json finds
	// UnmarshalText, so a UUID field reads back from a plain JSON string
	// without the payload struct needing a decoder of its own.
	type payload struct {
		ID domain.UUID `json:"id"`
	}

	want := domain.UUID{0x70, 0x23, 0x21, 0x73}

	var got payload
	require.NoError(t, json.Unmarshal([]byte(`{"id":"`+want.String()+`"}`), &got))
	require.Equal(t, want, got.ID)
}

func TestUUID_JSONMarshal(t *testing.T) {
	t.Parallel()

	// encoding/json finds MarshalText, so a UUID in a payload is a string.
	type payload struct {
		ID domain.UUID `json:"id"`
	}

	want := payload{ID: domain.UUID{0x70, 0x23, 0x21, 0x73}}

	body, err := json.Marshal(want)
	require.NoError(t, err)
	require.JSONEq(t, `{"id":"`+want.ID.String()+`"}`, string(body))
}
