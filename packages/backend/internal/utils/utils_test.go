package utils_test

import (
	"testing"

	"dockzilla/internal/utils"
	"dockzilla/pkg/domain"
	"github.com/stretchr/testify/require"
)

func TestGenerator(t *testing.T) {
	t.Parallel()

	const runs = 100

	seen := make(map[domain.UUID]bool, runs)

	for range runs {
		id := utils.Generator()

		require.NotEqual(t, domain.UUID{}, id, "generator returned the zero identifier")
		require.False(t, seen[id], "generator repeated an identifier")
		seen[id] = true
	}
}

func TestUUIDParser(t *testing.T) {
	t.Parallel()

	// Every accepted spelling denotes the same 16 bytes.
	want := domain.UUID{
		0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef,
		0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10,
	}

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "success - canonical form",
			input: "01234567-89ab-cdef-fedc-ba9876543210",
		},
		{
			name:  "success - uppercase",
			input: "01234567-89AB-CDEF-FEDC-BA9876543210",
		},
		{
			name:  "success - braced",
			input: "{01234567-89ab-cdef-fedc-ba9876543210}",
		},
		{
			name:  "success - urn prefixed",
			input: "urn:uuid:01234567-89ab-cdef-fedc-ba9876543210",
		},
		{
			// domain.UUID.String() renders the bare hex, so the parser has to
			// accept what our own logs print.
			name:  "success - undashed hex",
			input: "0123456789abcdeffedcba9876543210",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := utils.UUIDParser(tt.input)

			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}
}

func TestUUIDParser_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "failure - empty string",
			input: "",
		},
		{
			name:  "failure - too short",
			input: "01234567-89ab-cdef-fedc-ba98765432",
		},
		{
			name:  "failure - too long",
			input: "01234567-89ab-cdef-fedc-ba9876543210ff",
		},
		{
			name:  "failure - non hex character",
			input: "01234567-89ab-cdef-fedc-ba987654321z",
		},
		{
			name:  "failure - dashes misplaced",
			input: "0123456789-ab-cdef-fedc-ba9876543210",
		},
		{
			name:  "failure - unrelated text",
			input: "not-a-uuid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := utils.UUIDParser(tt.input)

			require.Error(t, err)
			require.ErrorContains(t, err, "failed to parse uuid")
			// A rejected input must not look like a usable identifier.
			require.Equal(t, domain.UUID{}, got)
		})
	}
}

func TestUUIDParser_KeepsProviderError(t *testing.T) {
	t.Parallel()

	_, err := utils.UUIDParser("nope")

	// errors.Join keeps the provider's message alongside ours, so the cause
	// survives the wrap.
	require.ErrorContains(t, err, "failed to parse uuid")
	require.ErrorContains(t, err, "invalid UUID length: 4")
}

func TestUUIDParser_RoundTripsGenerator(t *testing.T) {
	t.Parallel()

	id := utils.Generator()

	got, err := utils.UUIDParser(id.String())

	require.NoError(t, err)
	require.Equal(t, id, got)
}

func TestServiceIDGenerator(t *testing.T) {
	t.Parallel()

	id := utils.ServiceIDGenerator()

	// The loader's UUID and the domain's are the same 16 bytes, so a service
	// identifier stays readable in logs after the conversion.
	require.Len(t, id, 16)
	require.NotEqual(t, domain.UUID{}, domain.UUID(id))
	require.NotEqual(t, id, utils.ServiceIDGenerator())
}

func TestGenerator_SatisfiesDomainGenerator(t *testing.T) {
	t.Parallel()

	// utils.Generator is injected as a domain.Generator into the jobs use case.
	var gen domain.Generator = utils.Generator

	require.NotEqual(t, domain.UUID{}, gen())
}

func TestUUIDParser_SatisfiesDomainUUIDParser(t *testing.T) {
	t.Parallel()

	// utils.UUIDParser is injected as a domain.UUIDParser.
	var parser domain.UUIDParser = utils.UUIDParser

	id, err := parser("01234567-89ab-cdef-fedc-ba9876543210")

	require.NoError(t, err)
	require.NotEqual(t, domain.UUID{}, id)
}
