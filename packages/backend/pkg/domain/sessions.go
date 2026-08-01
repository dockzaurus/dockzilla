package domain

import "time"

// TokenHash is the digest of the opaque value a session cookie carries. Only
// the digest is kept, so a database dump yields nothing that can be replayed.
type TokenHash [64]byte

// Session is a browser login. The identifier is separate from the token so the
// dashboard can list and revoke sessions without ever handling the secret.
type Session struct {
	Identifier UUID
	UserID     UUID

	TokenHash []byte
	UserAgent string
	ClientIP  string

	CreatedAt *time.Time
	UpdatedAt *time.Time

	ExpiresAt time.Time
	RevokedAt *time.Time

	Users *User
}
