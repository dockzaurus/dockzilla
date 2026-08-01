package domain

import "time"

// User is a person who can drive the control plane. It carries no credential
// material: proving who the user is belongs to Identity, so adding a second
// provider never touches this type.
type User struct {
	Identifier  UUID
	Email       string
	DisplayName string

	CreatedAt *time.Time
	UpdatedAt *time.Time
}
