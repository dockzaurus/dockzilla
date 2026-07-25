package domain

import (
	"encoding/hex"
	"fmt"
)

// ParseUUID parses a hex-encoded identifier. It fails when s is not valid
// hexadecimal or does not decode to exactly len(UUID) bytes, so a caller never
// receives a partially filled identifier.
func ParseUUID(s string) (UUID, error) {
	var u UUID

	b, err := hex.DecodeString(s)
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
