package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Apps is a deployed application.
//
// Name becomes both a container name and a subdomain, so the database enforces
// DNS label rules on it. Getting that wrong surfaces much later as a
// certificate that will not issue.
//
// Rows are archived rather than deleted: deleting an app would take its
// deployment history with it, and that history is what rollback reads. The
// foreign key from deployments is RESTRICT, so the database enforces this.
type Apps struct {
	bun.BaseModel `bun:"table:apps"`

	ID         int64  `bun:"id,pk,autoincrement,type:bigint"`
	Identifier string `bun:"identifier,type:uuid,unique,notnull"`

	Name   string `bun:"name,notnull"`
	HostID string `bun:"host_id,type:uuid,notnull"`
	// ImageRef is the desired image, as the operator wrote it. The digest
	// actually run is recorded per deployment.
	ImageRef             string `bun:"image_ref,notnull"`
	RegistryCredentialID string `bun:"registry_credential_id,type:uuid,nullzero"`
	// DesiredState is one of the DesiredState constants, enum type
	// app_desired_state. Observed state is deliberately absent until the
	// reconciler's source-of-truth question is settled.
	DesiredState string `bun:"desired_state,type:app_desired_state,notnull,default:'running'"`

	ArchivedAt time.Time `bun:"archived_at,nullzero"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:now()"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:now()"`

	Host               *Hosts               `bun:"rel:belongs-to,join:host_id=identifier"`
	RegistryCredential *RegistryCredentials `bun:"rel:belongs-to,join:registry_credential_id=identifier"`
	Deployments        []*Deployments       `bun:"rel:has-many,join:identifier=app_id"`
	EnvVars            []*AppEnvVars        `bun:"rel:has-many,join:identifier=app_id"`
	Volumes            []*AppVolumes        `bun:"rel:has-many,join:identifier=app_id"`
}

// IsArchived reports whether the app has been retired. An archived app keeps
// its deployment history and frees its name for reuse.
func (a *Apps) IsArchived() bool {
	return !a.ArchivedAt.IsZero()
}
