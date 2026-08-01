package domain

import (
	"encoding/hex"
)

// UUID is the identifier assigned to every running service.
type UUID [16]byte

// String returns the hex encoding of the identifier. The receiver is a value so
// that UUID (not just *UUID) satisfies fmt.Stringer and renders correctly in
// log output.
func (u UUID) String() string {
	return hex.EncodeToString(u[:])
}
