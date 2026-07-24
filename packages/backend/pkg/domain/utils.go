package domain

import "encoding/hex"

// FromString cast the current string identifier into domain identifier.
func (u *UUID) FromString(s string) error {
	b, err := hex.DecodeString(s)
	if err != nil {
		return err
	}

	copy(u[:], b)

	return nil
}

func (u *UUID) String() string {
	return hex.EncodeToString(u[:])
}
