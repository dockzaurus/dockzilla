package domain

import "time"

type Provider string

const (
	LocalProvider Provider = "local"
)

type Identity struct {
	Identifier UUID
	UserID     UUID

	Provider        Provider
	ProviderSubject string

	PasswordHash []byte

	CreatedAt time.Time
	UpdatedAt time.Time

	User *User
}
