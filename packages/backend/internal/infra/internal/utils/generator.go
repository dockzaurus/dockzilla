// Package utils is dead code: a byte-for-byte duplicate of the Generator in
// internal/utils, sitting under a doubly-nested internal/ path that nothing can
// import. It is kept only until the surrounding storage packages are wired up
// or the tree is deleted.
package utils

import (
	"dockzilla/pkg/domain"
	"github.com/google/uuid"
)

// Generator returns a new random domain identifier.
func Generator() domain.UUID {
	return domain.UUID(uuid.New())
}
