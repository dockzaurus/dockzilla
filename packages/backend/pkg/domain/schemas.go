package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	errs "dockzilla/pkg/domain/errors"
)

// SchemaVersion names one revision of a Kind's payload contract.
//
// A version is published once and never edited. Producers and consumers of a
// job can be running different builds at the same time during a rolling
// deploy, so the meaning of "deployment.run/v1" has to be the same on every
// replica: changing a payload's shape means registering a new version, never
// rewriting an existing one.
type SchemaVersion string

// SchemaV1 is the first version every job kind ships with.
const SchemaV1 SchemaVersion = "v1"

// _schemaRefSeparator separates the kind from the version in a reference. A
// Kind never contains one, so cutting on the first occurrence is unambiguous.
const _schemaRefSeparator = "/"

// SchemaRef identifies a single schema in the registry. It is the registry's
// key, the value carried in an envelope's $schema field, and the path of the
// HTTP endpoint that serves the document.
type SchemaRef struct {
	Kind    Kind
	Version SchemaVersion
}

// ParseSchemaRef parses the "<kind>/<version>" form produced by
// SchemaRef.String, returning ErrInvalidSchemaRef when either half is missing.
func ParseSchemaRef(ref string) (SchemaRef, error) {
	kind, version, found := strings.Cut(ref, _schemaRefSeparator)
	if !found || kind == "" || version == "" {
		return SchemaRef{}, fmt.Errorf("%w: %q", errs.ErrInvalidSchemaRef, ref)
	}

	return SchemaRef{Kind: Kind(kind), Version: SchemaVersion(version)}, nil
}

// String renders the reference as "<kind>/<version>".
func (r SchemaRef) String() string {
	return string(r.Kind) + _schemaRefSeparator + string(r.Version)
}

// IsComplete reports whether both halves of the reference are set. Half a
// reference names nothing: every schema is addressed by kind and version
// together, and a missing version is not "the latest one".
func (r SchemaRef) IsComplete() bool {
	return r.Kind != "" && r.Version != ""
}

// Schema is one registered payload contract as the registry stores it.
// Document is the JSON Schema itself, kept as raw bytes because the registry
// stores and serves it verbatim and only the validator ever interprets it.
type Schema struct {
	Identifier string
	Ref        SchemaRef
	Document   json.RawMessage
	CreatedAt  time.Time
}
