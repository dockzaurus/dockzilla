package schemas_test

import (
	"encoding/json"
	"log/slog"
	"testing"

	"dockzilla/internal/core/jobs/schemas"
	"dockzilla/internal/core/jobs/schemas/mocks"
	"dockzilla/pkg/domain"
	"github.com/stretchr/testify/require"
)

// testRef is the reference every test in this package registers against.
func testRef() domain.SchemaRef {
	return domain.SchemaRef{Kind: domain.StartApp, Version: domain.SchemaV1}
}

// testDocument is a small but real schema: one required string property and
// nothing else allowed, which is enough to exercise both validation outcomes.
func testDocument() json.RawMessage {
	return json.RawMessage(`{
	  "$schema": "https://json-schema.org/draft/2020-12/schema",
	  "type": "object",
	  "properties": {"app_id": {"type": "string", "minLength": 1}},
	  "required": ["app_id"],
	  "additionalProperties": false
	}`)
}

func testSchema() domain.Schema {
	return domain.Schema{
		Identifier: "0f8fad5bd9cb469fa16570867728950e",
		Ref:        testRef(),
		Document:   testDocument(),
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// newUseCase builds a UseCase wired to repo, plus any extra options the test
// needs, failing the test when construction breaks.
func newUseCase(
	t *testing.T,
	repo schemas.Repository,
	extra ...schemas.UseCaseOption,
) *schemas.UseCase {
	t.Helper()

	opts := append([]schemas.UseCaseOption{
		schemas.WithLogger(discardLogger()),
		schemas.WithRepository(repo),
	}, extra...)

	uc, err := schemas.NewUseCase(opts...)
	require.NoError(t, err)

	return uc
}

func newRepo(t *testing.T) *mocks.MockRepository {
	t.Helper()

	return mocks.NewMockRepository(t)
}
