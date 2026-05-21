package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	IncidentStatusOpen          = "OPEN"
	IncidentStatusInvestigating = "INVESTIGATING"
	IncidentStatusResolved      = "RESOLVED"
)

var ValidIncidentStatuses = map[string]bool{
	IncidentStatusOpen:          true,
	IncidentStatusInvestigating: true,
	IncidentStatusResolved:      true,
}

func IsActiveIncidentStatus(status string) bool {
	return status == IncidentStatusOpen || status == IncidentStatusInvestigating
}

func IsValidIncidentStatus(status string) bool {
	return ValidIncidentStatuses[status]
}

type Incident struct {
	ID          uuid.UUID  `json:"id"`
	DeviceID    uuid.UUID  `json:"device_id"`
	DeviceName  string     `json:"device_name"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status"`
	Escalated   bool       `json:"escalated"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}
