package domain

import "time"

// Provider names the system that vouches for an identity. The value set is
// open: LocalProvider today, an external provider once one is supported.
type Provider string

const (
	// LocalProvider marks credentials Dockzilla owns itself, verified against
	// the identity's PasswordHash.
	LocalProvider Provider = "local"
)

// Identity is one way a user can prove who they are. It is separate from User
// so that adding a provider is a new identity pointing at the same user rather
// than a change to the user itself.
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
