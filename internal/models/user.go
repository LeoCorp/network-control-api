package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleViewer   = "viewer"
)

var ValidRoles = map[string]bool{
	RoleAdmin:    true,
	RoleOperator: true,
	RoleViewer:   true,
}

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func IsValidRole(role string) bool {
	return ValidRoles[role]
}
