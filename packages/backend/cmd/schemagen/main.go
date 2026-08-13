// Command schemagen generates the JSON Schema documents the schema catalog
// embeds, from the Go argument structs declared in pkg/domain.
//
// Run it after touching an argument struct:
//
//	task backend:schemas:gen
//
// Run it in verification mode to fail when the committed documents no longer
// match the structs:
//
//	task backend:schemas:check
//
// That check is the reason a version can be called immutable. Editing a
// published argument struct produces a diff against a committed file rather
// than a silent change of meaning, and the fix is almost always to add a new
// version next to the old one instead of accepting the diff.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dockzilla/internal/core/jobs/schemas/catalog"
	"github.com/invopop/jsonschema"
)

const (
	// _module and _argsPackage tell the reflector where to read the doc
	// comments that become each field's description.
	_module      = "dockzilla"
	_argsPackage = "./pkg/domain"

	_defaultRoot = "internal/core/jobs/schemas/catalog"

	_dirPerm  = 0o750
	_filePerm = 0o600
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	root := flag.String("root", _defaultRoot, "catalog package directory to write into")
	check := flag.Bool("check", false, "verify the committed documents instead of writing them")
	flag.Parse()

	reflector := &jsonschema.Reflector{
		// Anonymous drops the generated $id, which would otherwise bake this
		// module's path into a document served to external callers.
		Anonymous: true,
		// A stored schema is read on its own, so it has to be self-contained:
		// no $defs, no $ref out to a sibling document.
		ExpandedStruct: true,
		DoNotReference: true,
		// Required comes from an explicit `jsonschema:"required"` tag. A
		// published contract should say what it means rather than inherit it
		// from whether someone wrote omitempty.
		RequiredFromJSONSchemaTags: true,
	}

	if err := reflector.AddGoComments(_module, _argsPackage); err != nil {
		return fmt.Errorf("load argument doc comments: %w", err)
	}

	var stale []string

	for _, declared := range catalog.Types() {
		document, err := generate(reflector, declared)
		if err != nil {
			return err
		}

		target := filepath.Join(*root, filepath.FromSlash(catalog.Path(declared.Ref)))

		if !*check {
			if writeErr := write(target, document); writeErr != nil {
				return writeErr
			}

			continue
		}

		current, err := verify(target, document)
		if err != nil {
			return err
		}

		if !current {
			stale = append(stale, declared.Ref.String())
		}
	}

	if len(stale) > 0 {
		return fmt.Errorf(
			"committed schemas are out of date for %s.\n"+
				"A published version is frozen: run `task backend:schemas:gen` only if you meant "+
				"to change it, and add a new version instead if the change is not backward "+
				"compatible",
			strings.Join(stale, ", "),
		)
	}

	return nil
}

func generate(reflector *jsonschema.Reflector, declared catalog.Type) (json.RawMessage, error) {
	schema := reflector.Reflect(declared.Target)

	document, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal schema %s: %w", declared.Ref, err)
	}

	return append(document, '\n'), nil
}

func write(target string, document json.RawMessage) error {
	if err := os.MkdirAll(filepath.Dir(target), _dirPerm); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(target), err)
	}

	if err := os.WriteFile(target, document, _filePerm); err != nil {
		return fmt.Errorf("write %s: %w", target, err)
	}

	return nil
}

func verify(target string, document json.RawMessage) (bool, error) {
	committed, err := os.ReadFile(target) //nolint:gosec // the path comes from the catalog table.
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("read %s: %w", target, err)
	}

	return bytes.Equal(committed, document), nil
}
