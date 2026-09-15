package domain

import (
	"encoding/hex"
	"fmt"
)

// UUID is the identifier assigned to every running service.
type UUID [16]byte

func encodeHex(dst []byte, src UUID) {
	hex.Encode(dst, src[:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], src[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], src[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], src[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:], src[10:])
}

// String returns the hex encoding of the identifier. The receiver is a value so
// that UUID (not just *UUID) satisfies fmt.Stringer and renders correctly in
// log output.
func (u UUID) String() string {
	var buf [36]byte
	encodeHex(buf[:], u)
	return string(buf[:])
}

// Bytes returns the UUID as a byte slice.
func (u UUID) Bytes() []byte {
	return u[:]
}

// MarshalText implements encoding.TextMarshaler, rendering the canonical dashed
// form. encoding/json consults it for any struct field of this type, so a UUID
// crosses the wire as a string rather than a 16-element byte array.
func (u UUID) MarshalText() ([]byte, error) {
	return []byte(u.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler, reading back exactly what
// MarshalText writes. A malformed identifier is an error rather than a partial
// value, so a caller never receives half a UUID.
func (u *UUID) UnmarshalText(text []byte) error {
	if len(text) != 36 {
		return fmt.Errorf("uuid: got %d bytes, want 36", len(text))
	}
	if text[8] != '-' || text[13] != '-' || text[18] != '-' || text[23] != '-' {
		return fmt.Errorf("uuid: %q is not the canonical dashed form", text)
	}

	var buf [32]byte
	copy(buf[0:8], text[0:8])
	copy(buf[8:12], text[9:13])
	copy(buf[12:16], text[14:18])
	copy(buf[16:20], text[19:23])
	copy(buf[20:32], text[24:36])

	// Decoding into scratch keeps a rejected identifier from leaving the
	// receiver half written.
	var out UUID
	if _, err := hex.Decode(out[:], buf[:]); err != nil {
		return fmt.Errorf("uuid: %w", err)
	}

	*u = out

	return nil
}
