package jobs_test

import (
	"log/slog"
	"testing"

	"dockzilla/internal/core/jobs"
	"dockzilla/pkg/domain"
	"github.com/stretchr/testify/require"
)

// testID is the identifier the stub generator hands out. It doubles as a
// domain.Generator, so a test can assert on the message the use case builds
// instead of matching a random UUID.
func testID() domain.UUID {
	return domain.UUID{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// newUseCase builds a UseCase wired to repo with the logger and generator every
// test shares, failing the test when construction breaks.
func newUseCase(t *testing.T, repo jobs.Repository) *jobs.UseCase {
	t.Helper()

	uc, err := jobs.New(
		jobs.WithLogger(discardLogger()),
		jobs.WithRepository(repo),
		jobs.WithGenerator(testID),
	)
	require.NoError(t, err)

	return uc
}
