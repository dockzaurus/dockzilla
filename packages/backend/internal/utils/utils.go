// Package utils provides the identifier generators used to tag running
// services.
package utils

import (
	"errors"

	"dockzilla/pkg/domain"
	"github.com/google/uuid"
	serviceloader "github.com/zixyos/goloader/service"
)

// Generator returns a new random domain identifier.
func Generator() domain.UUID {
	return domain.UUID(uuid.New())
}

// UUIDParser return the domain.UUID parsed from string.
func UUIDParser(uuidStr string) (domain.UUID, error) {
	parsedUUID, err := uuid.Parse(uuidStr)
	if err != nil {
		return domain.UUID{}, errors.Join(errors.New("failed to parse uuid"), err)
	}
	return domain.UUID(parsedUUID), nil
}

// ServiceIDGenerator returns a new random identifier in the shape the service
// loader expects. It is passed to serviceloader.WithGenerator so every service
// gets an identifier at startup.
func ServiceIDGenerator() serviceloader.UUID {
	return serviceloader.UUID(Generator())
}
