package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Hosts is a registered server, reachable over SSH or the local Docker socket.
//
// No credential material lives here. SSHKeyRef is a reference into the secret
// engine, not a key.
type Hosts struct {
	bun.BaseModel `bun:"table:hosts"`

	ID         int64  `bun:"id,pk,autoincrement,type:bigint"`
	Identifier string `bun:"identifier,type:uuid,unique,notnull"`

	Name string `bun:"name,notnull"`
	// ConnectionType is one of the Connection constants, enum type
	// host_connection_type.
	ConnectionType string `bun:"connection_type,type:host_connection_type,notnull"`

	// SocketPath is required when ConnectionType is ConnectionLocalSocket.
	SocketPath string `bun:"socket_path,nullzero"`

	// SSHHost and SSHUser are required when ConnectionType is ConnectionSSH.
	SSHHost string `bun:"ssh_host,nullzero"`
	SSHPort int    `bun:"ssh_port,nullzero"`
	SSHUser string `bun:"ssh_user,nullzero"`
	// SSHKeyRef points at the secret engine. Never the key itself.
	SSHKeyRef string `bun:"ssh_key_ref,nullzero"`

	// Status is one of the HostStatus constants, enum type host_status.
	Status     string    `bun:"status,type:host_status,notnull,default:'unknown'"`
	LastSeenAt time.Time `bun:"last_seen_at,nullzero"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:now()"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:now()"`

	Apps []*Apps `bun:"rel:has-many,join:identifier=host_id"`
}
