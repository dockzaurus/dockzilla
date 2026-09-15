package domain

import (
	"encoding/hex"
	"fmt"
)

// UUID is the identifier assigned to every running service.
type UUID [16]byte

// String returns the hex encoding of the identifier. The receiver is a value so
// that UUID (not just *UUID) satisfies fmt.Stringer and renders correctly in
// log output.
func (u UUID) String() string {
	return hex.EncodeToString(u[:])
}

// Bytes returns the UUID as a byte slice.
func (u UUID) Bytes() []byte {
	return u[:]
}

// MarshalJSON implements json.Marshaler, encoding the UUID as a hex string.
func (u UUID) MarshalJSON() ([]byte, error) {
	return []byte(`"` + u.String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler, decoding a hex string into the UUID.
func (u *UUID) UnmarshalJSON(data []byte) error {
	// Strip quotes
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return fmt.Errorf("invalid UUID JSON: %s", data)
	}
	hexStr := string(data[1 : len(data)-1])

	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		return fmt.Errorf("invalid UUID hex: %w", err)
	}
	if len(decoded) != 16 {
		return fmt.Errorf("invalid UUID length: got %d bytes, want 16", len(decoded))
	}

	copy(u[:], decoded)
	return nil
}
