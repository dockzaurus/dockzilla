package domain

import "encoding/hex"

type UUID [16]byte

func (u *UUID) String() string {
	return hex.EncodeToString(u[:])
}

func (u *UUID) FromString(s string) error {
	b, err := hex.DecodeString(s)
	if err != nil {
		return err
	}

	copy(u[:], b)

	return nil
}
