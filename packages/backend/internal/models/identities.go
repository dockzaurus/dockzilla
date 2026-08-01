package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Identities is one way a user can prove who they are. The table is separate
// from Users so that adding an external provider later is an insert rather than
// a migration: a Zitadel identity is a second row pointing at the same user.
//
// PasswordHash lives here rather than on Users because it is credential
// material belonging to one provider. A database constraint enforces that a
// local identity has one and any other provider does not.
type Identities struct {
	bun.BaseModel `bun:"table:identities"`

	ID         int64  `bun:"id,pk,autoincrement,type:bigint"`
	Identifier string `bun:"identifier,type:uuid,unique,notnull"`

	UserID string `bun:"user_id,type:uuid,notnull"`

	// Provider is ProviderLocal in V0 and "oidc:<name>" once an external
	// provider is supported. Not an enum type: the value set is open.
	Provider string `bun:"provider,notnull"`
	// ProviderSubject is the email for a local identity and the sub claim for
	// an OIDC one.
	ProviderSubject string `bun:"provider_subject,notnull"`
	// PasswordHash is an argon2id PHC string. Empty unless Provider is local.
	PasswordHash string `bun:"password_hash,nullzero"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:now()"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:now()"`

	User *Users `bun:"rel:belongs-to,join:user_id=identifier"`
}

// IsLocal reports whether this identity carries a password Dockzilla owns.
func (i *Identities) IsLocal() bool {
	return i.Provider == ProviderLocal
}
