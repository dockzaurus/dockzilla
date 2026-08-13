package catalog_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"dockzilla/internal/core/jobs/schemas/catalog"
	"dockzilla/pkg/domain"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/require"
)

func TestTypes_CoverEveryJobKind(t *testing.T) {
	t.Parallel()

	// A kind the engine can consume but nobody published a contract for would
	// fail validation at dispatch, in production, on the first job of that
	// kind. Catching it here is the point of the catalog being a table.
	declared := make(map[domain.Kind]bool)
	for _, entry := range catalog.Types() {
		declared[entry.Ref.Kind] = true
	}

	for _, kind := range domain.AllKinds() {
		require.Truef(t, declared[kind],
			"job kind %q has no entry in catalog.Types(); add one and regenerate", kind,
		)
	}
}

func TestTypes_DeclareCompleteReferences(t *testing.T) {
	t.Parallel()

	for _, entry := range catalog.Types() {
		require.Truef(t, entry.Ref.IsComplete(), "catalog entry %q is missing a half", entry.Ref)
		require.NotNilf(t, entry.Target, "catalog entry %s has no target type", entry.Ref)
	}
}

func TestDocuments_AreCompilableSchemas(t *testing.T) {
	t.Parallel()

	documents, err := catalog.Documents()
	require.NoError(t, err)
	require.Len(t, documents, len(catalog.Types()))

	for _, document := range documents {
		t.Run(document.Ref.String(), func(t *testing.T) {
			t.Parallel()

			// Bootstrap publishes these unconditionally at boot, so a document
			// the validator cannot compile would take the process down rather
			// than surface as a failed request.
			value, err := jsonschema.UnmarshalJSON(bytes.NewReader(document.Document))
			require.NoError(t, err)

			compiler := jsonschema.NewCompiler()
			require.NoError(t, compiler.AddResource(document.Ref.String(), value))

			_, err = compiler.Compile(document.Ref.String())
			require.NoError(t, err)
		})
	}
}

func TestDocuments_AreSelfContained(t *testing.T) {
	t.Parallel()

	documents, err := catalog.Documents()
	require.NoError(t, err)

	for _, document := range documents {
		var decoded map[string]any
		require.NoError(t, json.Unmarshal(document.Document, &decoded))

		// Each document is stored in its own row and served on its own, so a
		// $ref out to a sibling would resolve to nothing for the caller.
		require.NotContainsf(t, decoded, "$defs", "%s must not reference definitions", document.Ref)
		require.NotContainsf(t, decoded, "$ref", "%s must not reference definitions", document.Ref)
	}
}

func TestPath_LocatesADocumentByReference(t *testing.T) {
	t.Parallel()

	ref := domain.SchemaRef{Kind: domain.RunDeployment, Version: domain.SchemaV1}

	require.Equal(t, "schema/deployment.run/v1.json", catalog.Path(ref))
}
