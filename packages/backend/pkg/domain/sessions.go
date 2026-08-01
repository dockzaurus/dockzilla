package domain

import "time"

type TokenHash [64]byte
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

	Users *Users
}
