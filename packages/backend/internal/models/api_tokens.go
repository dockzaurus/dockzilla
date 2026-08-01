package models

import (
	"time"

	"github.com/uptrace/bun"
)

// APITokens is a non-interactive credential for the CLI and for automation.
// Separate from Sessions on purpose: the CLI never holds a cookie and the
// dashboard never holds a bearer token.
//
// Only the SHA-256 digest is stored. The plaintext is shown once, at creation.
// A password KDF would be the wrong tool here because the token is already a
// high-entropy random value.
type APITokens struct {
	bun.BaseModel `bun:"table:api_tokens"`

	ID         int64  `bun:"id,pk,autoincrement,type:bigint"`
	Identifier string `bun:"identifier,type:uuid,unique,notnull"`

	UserID string `bun:"user_id,type:uuid,notnull"`
	Name   string `bun:"name,notnull"`
	// TokenHash is the SHA-256 of the token, prefix included.
	TokenHash []byte `bun:"token_hash,type:bytea,notnull"`
	// TokenPrefix holds a displayable fragment such as "dckz_a1b2c3" so the
	// dashboard can tell tokens apart without holding the secret.
	TokenPrefix string `bun:"token_prefix,notnull"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:now()"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:now()"`

	// LastUsedAt is written by the auth middleware. Update it with a throttle,
	// otherwise every authenticated CLI call becomes a database write.
	LastUsedAt time.Time `bun:"last_used_at,nullzero"`
	ExpiresAt  time.Time `bun:"expires_at,nullzero"`
	RevokedAt  time.Time `bun:"revoked_at,nullzero"`

	User *Users `bun:"rel:belongs-to,join:user_id=identifier"`
}

// IsActive reports whether the token may still authenticate a request. A zero
// ExpiresAt means the token does not expire.
func (t *APITokens) IsActive(now time.Time) bool {
	if !t.RevokedAt.IsZero() {
		return false
	}

	return t.ExpiresAt.IsZero() || t.ExpiresAt.After(now)
}
