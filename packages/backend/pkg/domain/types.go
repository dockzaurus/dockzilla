package domain

import (
	"context"
)

// UUID type represent the domain type for UUID.
type UUID [16]byte

// Service type represent the service definitions.
type Service interface {
	Run(context.Context) error
	Stop(ctx context.Context) error
	Name() string
}
