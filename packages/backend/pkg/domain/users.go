package domain

import "time"

type User struct {
	Identifier  UUID
	Email       string
	DisplayName string

	CreatedAt *time.Time
	UpdatedAt *time.Time
}
