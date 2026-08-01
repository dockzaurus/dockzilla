package domain

import (
	"crypto/rand"
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"strings"
)

// NewUUID returns a randomly generated version 4 identifier. The domain
// generates identifiers itself so an entity is complete before it is persisted:
// an aggregate can reference its children in memory, and nothing has to wait on
// a database round trip to learn its own id.
func NewUUID() (UUID, error) {
	var u UUID

	if _, err := rand.Read(u[:]); err != nil {
		return u, fmt.Errorf("generate uuid: %w", err)
	}

	u[6] = (u[6] & 0x0f) | 0x40 // version 4
	u[8] = (u[8] & 0x3f) | 0x80 // RFC 4122 variant

	return u, nil
}

// ParseUUID parses a hex-encoded identifier, with or without the canonical
// hyphens. It fails when s is not valid hexadecimal or does not decode to
// exactly len(UUID) bytes, so a caller never receives a partially filled
// identifier.
func ParseUUID(s string) (UUID, error) {
	var u UUID

	// PostgreSQL renders uuid values hyphenated, callers often pass the compact
	// form, and both name the same identifier.
	compact := strings.ReplaceAll(s, "-", "")

	b, err := hex.DecodeString(compact)
	if err != nil {
		return u, fmt.Errorf("decode uuid %q: %w", s, err)
	}

	if len(b) != len(u) {
		return u, fmt.Errorf("decode uuid %q: got %d bytes, want %d", s, len(b), len(u))
	}

	copy(u[:], b)

	return u, nil
}

// String returns the hex encoding of the identifier. The receiver is a value so
// that UUID (not just *UUID) satisfies fmt.Stringer and renders correctly in
// log output.
func (u UUID) String() string {
	return hex.EncodeToString(u[:])
}

// Canonical returns the hyphenated 8-4-4-4-12 form. This is what crosses a
// boundary: the JSON body, a URL path, a SQL statement. String stays compact
// because it exists for log lines.
func (u UUID) Canonical() string {
	h := hex.EncodeToString(u[:])

	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}

// IsZero reports whether the identifier has never been set. Useful for guarding
// against an entity that skipped NewUUID.
func (u UUID) IsZero() bool {
	return u == UUID{}
}

// Value implements driver.Valuer so a UUID can be written to a PostgreSQL uuid
// column. The canonical form is used because the driver sends parameters as
// text and PostgreSQL parses that form unambiguously.
func (u UUID) Value() (driver.Value, error) {
	if u.IsZero() {
		//nolint:nilnil // driver.Valuer encodes a SQL NULL as (nil, nil).
		return nil, nil
	}

	return u.Canonical(), nil
}

// Scan implements sql.Scanner. It accepts the raw 16 bytes, the canonical
// hyphenated text PostgreSQL returns, and the compact hex form, so a value
// survives a round trip regardless of which representation the driver picked.
func (u *UUID) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*u = UUID{}

		return nil
	case UUID:
		*u = v

		return nil
	case []byte:
		if len(v) == len(*u) {
			copy(u[:], v)

			return nil
		}

		parsed, err := ParseUUID(string(v))
		if err != nil {
			return fmt.Errorf("scan uuid: %w", err)
		}

		*u = parsed

		return nil
	case string:
		parsed, err := ParseUUID(v)
		if err != nil {
			return fmt.Errorf("scan uuid: %w", err)
		}

		*u = parsed

		return nil
	default:
		return fmt.Errorf("scan uuid: unsupported source type %T", src)
	}
}

// MarshalJSON renders the canonical form, which is the only representation that
// leaves the process.
func (u UUID) MarshalJSON() ([]byte, error) {
	return []byte(`"` + u.Canonical() + `"`), nil
}

// UnmarshalJSON accepts either the canonical or the compact form.
func (u *UUID) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)

	if s == "" || s == "null" {
		*u = UUID{}

		return nil
	}

	parsed, err := ParseUUID(s)
	if err != nil {
		return fmt.Errorf("unmarshal uuid: %w", err)
	}

	*u = parsed

	return nil
}
