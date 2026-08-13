package domain_test

import (
	"testing"

	"dockzilla/pkg/domain"
	errs "dockzilla/pkg/domain/errors"
	"github.com/stretchr/testify/require"
)

func TestSchemaRef_String(t *testing.T) {
	t.Parallel()

	ref := domain.SchemaRef{Kind: domain.RunDeployment, Version: domain.SchemaV1}

	require.Equal(t, "deployment.run/v1", ref.String())
}

func TestParseSchemaRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ref  string
		want domain.SchemaRef
	}{
		{
			name: "success - a kind and a version",
			ref:  "deployment.run/v1",
			want: domain.SchemaRef{Kind: domain.RunDeployment, Version: domain.SchemaV1},
		},
		{
			name: "success - a kind whose name contains dots",
			ref:  "app.restart/v2",
			want: domain.SchemaRef{Kind: domain.RestartApp, Version: "v2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.ParseSchemaRef(tt.ref)

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParseSchemaRef_Rejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ref  string
	}{
		{name: "error - empty", ref: ""},
		{name: "error - no separator", ref: "deployment.run"},
		{name: "error - no version", ref: "deployment.run/"},
		{name: "error - no kind", ref: "/v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := domain.ParseSchemaRef(tt.ref)

			require.ErrorIs(t, err, errs.ErrInvalidSchemaRef)
		})
	}
}

func TestParseSchemaRef_RoundTripsString(t *testing.T) {
	t.Parallel()

	// The reference travels as a string in a job envelope and comes back as a
	// pair, so the two representations have to agree in both directions.
	for _, kind := range domain.AllKinds() {
		original := domain.SchemaRef{Kind: kind, Version: domain.SchemaV1}

		parsed, err := domain.ParseSchemaRef(original.String())

		require.NoError(t, err)
		require.Equal(t, original, parsed)
	}
}

func TestSchemaRef_IsComplete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ref  domain.SchemaRef
		want bool
	}{
		{
			name: "both halves set",
			ref:  domain.SchemaRef{Kind: domain.StartApp, Version: domain.SchemaV1},
			want: true,
		},
		{
			name: "version missing is not the latest version",
			ref:  domain.SchemaRef{Kind: domain.StartApp},
		},
		{
			name: "kind missing",
			ref:  domain.SchemaRef{Version: domain.SchemaV1},
		},
		{
			name: "zero value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, tt.ref.IsComplete())
		})
	}
}
