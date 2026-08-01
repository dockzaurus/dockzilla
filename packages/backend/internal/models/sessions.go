package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Sessions is a browser login. The cookie carries an opaque random value and
// only its SHA-256 digest is stored, so a database dump yields nothing usable.
//
// Identifier exists so the dashboard can list active sessions and revoke one
// without ever seeing the token.
type Sessions struct {
	bun.BaseModel `bun:"table:sessions"`

	ID         int64  `bun:"id,pk,autoincrement,type:bigint"`
	Identifier string `bun:"identifier,type:uuid,unique,notnull"`

	UserID string `bun:"user_id,type:uuid,notnull"`
	// TokenHash is the SHA-256 of the opaque cookie value.
	TokenHash []byte `bun:"token_hash,type:bytea,notnull"`
	UserAgent string `bun:"user_agent,nullzero"`
	// ClientIP maps a PostgreSQL inet column. The driver exchanges it as text.
	ClientIP string `bun:"client_ip,type:inet,nullzero"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:now()"`
	// UpdatedAt doubles as last-seen, which drives the idle timeout. The
	// trigger also bumps it on revocation, so the two meanings overlap.
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:now()"`

	// ExpiresAt is the absolute expiry.
	ExpiresAt time.Time `bun:"expires_at,notnull"`
	RevokedAt time.Time `bun:"revoked_at,nullzero"`

	User *Users `bun:"rel:belongs-to,join:user_id=identifier"`
}

// IsActive reports whether the session may still authenticate a request.
func (s *Sessions) IsActive(now time.Time) bool {
	return s.RevokedAt.IsZero() && s.ExpiresAt.After(now)
}
