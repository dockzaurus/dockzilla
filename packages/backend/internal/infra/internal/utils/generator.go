package utils

import (
	"dockzilla/pkg/domain"

	"github.com/google/uuid"
)

func Generator() domain.UUID {
	return domain.UUID(uuid.New())
}
