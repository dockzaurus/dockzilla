// Package catalog is the set of payload contracts this binary ships with.
//
// It holds two halves of the same fact. Types is the table that maps a schema
// reference to the Go struct the payload is unmarshalled into, and it is the
// one place a new job kind has to be declared. The schema directory holds the
// JSON Schema generated from each of those structs, committed and embedded, so
// what the binary publishes at boot is a reviewed artefact rather than
// whatever the reflection library happens to produce today.
//
// Regenerate with `task backend:schemas:gen` after touching an argument
// struct. `task backend:schemas:check` fails when the two halves disagree,
// which is what stops an edit to a frozen version from reaching production:
// the diff has to be looked at, and the answer is almost always to add a new
// version rather than to accept it.
package catalog

import (
	"embed"
	"fmt"
	"io/fs"
	"path"

	"dockzilla/pkg/domain"
)

// _root is the directory the generated documents live in, inside this package
// and inside the embedded filesystem alike.
const _root = "schema"

// documents holds the generated schemas. //go:embed requires a package-level
// variable and the filesystem is read-only, so this is not shared mutable
// state.
//
//go:embed schema
var documents embed.FS

// Type binds a schema reference to the Go type the payload decodes into.
// cmd/schemagen reflects over Target to produce the document; the job engine
// unmarshals into the same type when it dispatches the kind.
type Type struct {
	Ref    domain.SchemaRef
	Target any
}

// Types returns every payload contract the binary ships with. Adding a job
// kind means adding a line here and regenerating; the catalog test fails when
// a domain.Kind has no entry.
func Types() []Type {
	return []Type{
		{
			Ref:    domain.SchemaRef{Kind: domain.RunDeployment, Version: domain.SchemaV1},
			Target: domain.DeployArgsV1{},
		},
		{
			Ref:    domain.SchemaRef{Kind: domain.StartApp, Version: domain.SchemaV1},
			Target: domain.StartAppArgsV1{},
		},
		{
			Ref:    domain.SchemaRef{Kind: domain.StopApp, Version: domain.SchemaV1},
			Target: domain.StopAppArgsV1{},
		},
		{
			Ref:    domain.SchemaRef{Kind: domain.RestartApp, Version: domain.SchemaV1},
			Target: domain.RestartAppArgsV1{},
		},
	}
}

// Path returns the location of ref's document relative to this package.
func Path(ref domain.SchemaRef) string {
	return path.Join(_root, string(ref.Kind), string(ref.Version)+".json")
}

// Documents returns every built-in schema, ready to be published by the
// registry's Bootstrap. It fails when a declared type has no generated
// document, which means the catalog was not regenerated after a type was
// added.
func Documents() ([]domain.Schema, error) {
	types := Types()
	schemas := make([]domain.Schema, 0, len(types))

	for _, declared := range types {
		document, err := fs.ReadFile(documents, Path(declared.Ref))
		if err != nil {
			return nil, fmt.Errorf("read built-in schema %s: %w", declared.Ref, err)
		}

		schemas = append(schemas, domain.Schema{
			Ref:      declared.Ref,
			Document: document,
		})
	}

	return schemas, nil
}
