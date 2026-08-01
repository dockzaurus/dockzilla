package models

import (
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

// Jobs is the async work queue. Nothing in V0 works without it: an image pull
// takes minutes, so no deployment can run inside a request.
//
// Workers claim with FOR UPDATE SKIP LOCKED, which removes the need for an
// external queue:
//
//	SELECT * FROM jobs
//	WHERE status = 'pending' AND run_after <= now()
//	ORDER BY run_after
//	LIMIT 1
//	FOR UPDATE SKIP LOCKED
type Jobs struct {
	bun.BaseModel `bun:"table:jobs"`

	ID         int64  `bun:"id,pk,autoincrement,type:bigint"`
	Identifier string `bun:"identifier,type:uuid,unique,notnull"`

	// Kind is not an enum type: one value is added per job kind.
	Kind string `bun:"kind,notnull"`
	// Payload stays untyped here. Each job kind decodes it into its own struct,
	// so adding a kind does not touch this model.
	Payload json.RawMessage `bun:"payload,type:jsonb,notnull"`
	// Status is one of the Job constants, enum type job_status.
	Status string `bun:"status,type:job_status,notnull,default:'pending'"`

	Attempts    int `bun:"attempts,notnull,default:0"`
	MaxAttempts int `bun:"max_attempts,notnull,default:3"`

	// RunAfter supports both scheduling and retry backoff.
	RunAfter time.Time `bun:"run_after,nullzero,notnull,default:now()"`
	// LockedAt and LockedBy identify the worker holding the job, so a crashed
	// worker's jobs can be reclaimed.
	LockedAt  time.Time `bun:"locked_at,nullzero"`
	LockedBy  string    `bun:"locked_by,nullzero"`
	LastError string    `bun:"last_error,nullzero"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:now()"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:now()"`
}

// CanRetry reports whether a failed job has attempts left.
func (j *Jobs) CanRetry() bool {
	return j.Attempts < j.MaxAttempts
}
