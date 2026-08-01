package models

import (
	"time"

	"github.com/uptrace/bun"
)

// RegistryCredentials is auth for pulling from a private registry.
//
// Modelling it as its own entity, referenced by apps, avoids having to choose
// between per-host and per-app credentials: it covers both.
type RegistryCredentials struct {
	bun.BaseModel `bun:"table:registry_credentials"`

	ID         int64  `bun:"id,pk,autoincrement,type:bigint"`
	Identifier string `bun:"identifier,type:uuid,unique,notnull"`

	Name string `bun:"name,notnull"`
	// RegistryHost is the registry this credential authenticates against, for
	// example "ghcr.io".
	RegistryHost string `bun:"registry_host,notnull"`
	Username     string `bun:"username,notnull"`
	// PasswordEncrypted is ciphertext produced by the secret engine. This table
	// stores an opaque blob and never plaintext.
	PasswordEncrypted []byte `bun:"password_encrypted,type:bytea,notnull"`
	// EncryptionKeyVersion records which master key version produced the
	// ciphertext. Without it, rotating the master key leaves every value
	// undecryptable and unidentifiable.
	EncryptionKeyVersion int `bun:"encryption_key_version,notnull"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:now()"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:now()"`

	Apps []*Apps `bun:"rel:has-many,join:identifier=registry_credential_id"`
}
