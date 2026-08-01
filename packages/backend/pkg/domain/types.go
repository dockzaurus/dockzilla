// Package domain holds the vocabulary shared by every layer of the backend:
// the identifier type and the Service contract that each runnable component
// implements. It depends on nothing but the standard library, so any package
// may import it without creating a cycle.
package domain

import (
	"context"
)

// UUID is the identifier assigned to every running service.
//
//nolint:recvcheck // sql.Scanner and json.Unmarshaler require pointer receivers.
type UUID [16]byte

// Service is the contract implemented by anything the application can run: an
// HTTP transport, a worker pool, a scheduler. Implementations must be safe to
// Stop even if Run failed.
type Service interface {
	// Run starts the service. It should return once the service is running
	// or has failed to start, not block for the service's lifetime.
	Run(ctx context.Context) error

	// Stop shuts the service down, respecting ctx's deadline.
	Stop(ctx context.Context) error

	// Name identifies the service in logs.
	Name() string
}
