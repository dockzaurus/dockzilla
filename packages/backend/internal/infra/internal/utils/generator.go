package utils

import (
	"dockzilla/pkg/domain"
	"github.com/google/uuid"
)

// Generator returns a new random domain identifier.
func Generator() domain.UUID {
	return domain.UUID(uuid.New())
}
