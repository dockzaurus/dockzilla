package models

import (
	"time"

	"github.com/uptrace/bun"
)

// AppEnvVars is one environment variable for one app, encrypted at rest.
//
// ValueEncrypted is bytea and never text. The secret engine owns the
// encryption; this table stores an opaque blob and the key version that
// produced it.
type AppEnvVars struct {
	bun.BaseModel `bun:"table:app_env_vars"`

	ID         int64  `bun:"id,pk,autoincrement,type:bigint"`
	Identifier string `bun:"identifier,type:uuid,unique,notnull"`

	AppID string `bun:"app_id,type:uuid,notnull"`
	Key   string `bun:"key,notnull"`
	// ValueEncrypted is ciphertext. Plaintext never reaches this table.
	ValueEncrypted []byte `bun:"value_encrypted,type:bytea,notnull"`
	// EncryptionKeyVersion records which master key version produced the
	// ciphertext, so the key can be rotated.
	EncryptionKeyVersion int `bun:"encryption_key_version,notnull"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:now()"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:now()"`

	App *Apps `bun:"rel:belongs-to,join:app_id=identifier"`
}
