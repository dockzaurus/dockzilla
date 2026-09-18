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

func TestGenerator_RoundTripsThroughDomainParser(t *testing.T) {
	t.Parallel()

	// A generated identifier must survive String -> ParseUUID, since that is
	// the canonical text form it travels and is stored as.
	id := utils.Generator()

	got, err := domain.ParseUUID(id.String())

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

func TestParseUUID_SatisfiesDomainUUIDParser(t *testing.T) {
	t.Parallel()

	// domain.ParseUUID is injected as a domain.UUIDParser, replacing the
	// former utils.UUIDParser wrapper so the app shares one parsing policy.
	var parser domain.UUIDParser = domain.ParseUUID

	id, err := parser("01234567-89ab-cdef-fedc-ba9876543210")

	require.NoError(t, err)
	require.NotEqual(t, domain.UUID{}, id)
}
