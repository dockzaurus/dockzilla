package utils

import (
	"dockzilla/pkg/domain"

	"github.com/google/uuid"
	serviceloader "github.com/zixyos/goloader/service"
)

func Generator() domain.UUID {
	return domain.UUID(uuid.New())
}

func ServiceIDGenerator() serviceloader.UUID {
	return serviceloader.UUID(Generator())
}
