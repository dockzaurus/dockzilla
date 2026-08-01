package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Deployments is the ledger. Rows are append-only: Status moves during a
// deployment's life and then freezes.
//
// ImageDigest is what makes rollback real. Rolling back to a previous ImageRef
// is meaningless once the tag has moved, which is the normal case with :latest
// or a mutable release tag. Resolve the digest at pull time, store it here, and
// roll back to the digest.
type Deployments struct {
	bun.BaseModel `bun:"table:deployments"`

	ID         int64  `bun:"id,pk,autoincrement,type:bigint"`
	Identifier string `bun:"identifier,type:uuid,unique,notnull"`

	AppID string `bun:"app_id,type:uuid,notnull"`
	// ImageRef is the reference as requested.
	ImageRef string `bun:"image_ref,notnull"`
	// ImageDigest is the immutable sha256 digest resolved at pull time. Empty
	// until the pull succeeds.
	ImageDigest string `bun:"image_digest,nullzero"`
	// Status is one of the Deployment constants, enum type deployment_status.
	Status      string `bun:"status,type:deployment_status,notnull,default:'queued'"`
	ContainerID string `bun:"container_id,nullzero"`

	// TriggeredBy is one of the TriggeredBy constants, enum type
	// deployment_trigger.
	TriggeredBy string `bun:"triggered_by,type:deployment_trigger,notnull"`
	// TriggeredByUserID detaches rather than cascades, so deleting a user does
	// not remove deployment history.
	TriggeredByUserID string `bun:"triggered_by_user_id,type:uuid,nullzero"`

	// ErrorCode is a stable machine-readable code, matching the API's error
	// contract. ErrorMessage is human-facing and free to change.
	ErrorCode    string `bun:"error_code,nullzero"`
	ErrorMessage string `bun:"error_message,nullzero"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:now()"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:now()"`

	StartedAt  time.Time `bun:"started_at,nullzero"`
	FinishedAt time.Time `bun:"finished_at,nullzero"`

	App             *Apps  `bun:"rel:belongs-to,join:app_id=identifier"`
	TriggeredByUser *Users `bun:"rel:belongs-to,join:triggered_by_user_id=identifier"`
}

// IsTerminal reports whether the deployment has stopped moving.
func (d *Deployments) IsTerminal() bool {
	switch d.Status {
	case DeploymentRunning, DeploymentFailed, DeploymentSuperseded:
		return true
	default:
		return false
	}
}
