package models

import (
	"time"

	"github.com/uptrace/bun"
)

// AppVolumes is a persistent volume mounted into an app's container. MountPath
// is unique per app, because two volumes cannot occupy the same path.
type AppVolumes struct {
	bun.BaseModel `bun:"table:app_volumes"`

	ID         int64  `bun:"id,pk,autoincrement,type:bigint"`
	Identifier string `bun:"identifier,type:uuid,unique,notnull"`

	AppID string `bun:"app_id,type:uuid,notnull"`
	// Name is the Docker volume name.
	Name      string `bun:"name,notnull"`
	MountPath string `bun:"mount_path,notnull"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:now()"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:now()"`

	App *Apps `bun:"rel:belongs-to,join:app_id=identifier"`
}
