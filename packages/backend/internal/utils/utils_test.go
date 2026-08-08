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
