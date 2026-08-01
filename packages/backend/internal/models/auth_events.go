package models

import (
	"time"

	"github.com/uptrace/bun"
)

// AuthEvents is the audit trail. A platform that runs arbitrary containers
// wants a record of who authenticated and when.
//
// UserID is nullable and detaches rather than cascades on delete, for two
// reasons: a failed login for an unknown address has no user to point at, and
// deleting a user must not erase the record that they existed and acted.
//
// Nothing here may contain credential material, plaintext or hashed.
type AuthEvents struct {
	bun.BaseModel `bun:"table:auth_events"`

	ID         int64  `bun:"id,pk,autoincrement,type:bigint"`
	Identifier string `bun:"identifier,type:uuid,unique,notnull"`

	UserID string `bun:"user_id,type:uuid,nullzero"`
	// Kind is one of the AuthEvent constants. Not an enum type: the set grows
	// with every auditable action.
	Kind string `bun:"kind,notnull"`
	// ClientIP maps a PostgreSQL inet column. The driver exchanges it as text.
	ClientIP string `bun:"client_ip,type:inet,nullzero"`
	// RequestID links the entry back to the API call that produced it.
	RequestID string `bun:"request_id,nullzero"`
	// Detail carries event-specific context, for example the failure reason.
	Detail map[string]any `bun:"detail,type:jsonb,nullzero"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:now()"`

	User *Users `bun:"rel:belongs-to,join:user_id=identifier"`
}
