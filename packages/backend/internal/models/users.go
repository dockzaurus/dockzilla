package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Users is a person who can drive the control plane. Email is unique so two
// users cannot share one, but it is not the identity key: authentication looks
// up Identities, never this table.
type Users struct {
	bun.BaseModel `bun:"table:users"`

	ID         int64  `bun:"id,pk,autoincrement,type:bigint"`
	Identifier string `bun:"identifier,type:uuid,unique,notnull"`

	Email       string `bun:"email,notnull"`
	DisplayName string `bun:"display_name,nullzero"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:now()"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:now()"`

	Identities []*Identities `bun:"rel:has-many,join:identifier=user_id"`
	Sessions   []*Sessions   `bun:"rel:has-many,join:identifier=user_id"`
	APITokens  []*APITokens  `bun:"rel:has-many,join:identifier=user_id"`
}
