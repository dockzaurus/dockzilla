package models

import (
	"time"

	"github.com/uptrace/bun"
)

// SetupClaims is the one-time token that bootstraps the first admin. The
// installer writes the row and prints the plaintext to stdout once, so no
// default credential ever exists.
//
// Consume it with a conditional update rather than a read followed by a write,
// so there is no window between checking the token and burning it:
//
//	UPDATE setup_claims
//	SET consumed_at = now(), consumed_by_user_id = ?
//	WHERE token_hash = ? AND consumed_at IS NULL AND expires_at > now()
//	RETURNING id
//
// No row returned means the token was already used, expired, or wrong.
type SetupClaims struct {
	bun.BaseModel `bun:"table:setup_claims"`

	ID         int64  `bun:"id,pk,autoincrement,type:bigint"`
	Identifier string `bun:"identifier,type:uuid,unique,notnull"`

	// TokenHash is the SHA-256 of the claim token. The plaintext is never
	// stored and never logged.
	TokenHash  []byte    `bun:"token_hash,type:bytea,notnull"`
	ExpiresAt  time.Time `bun:"expires_at,notnull"`
	ConsumedAt time.Time `bun:"consumed_at,nullzero"`
	// ConsumedByUserID is set to the admin created by this claim.
	ConsumedByUserID string `bun:"consumed_by_user_id,type:uuid,nullzero"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:now()"`

	ConsumedBy *Users `bun:"rel:belongs-to,join:consumed_by_user_id=identifier"`
}

// IsClaimable reports whether the token can still be consumed.
func (c *SetupClaims) IsClaimable(now time.Time) bool {
	return c.ConsumedAt.IsZero() && c.ExpiresAt.After(now)
}
