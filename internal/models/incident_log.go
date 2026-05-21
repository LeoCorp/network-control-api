package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	ActionCreated       = "created"
	ActionStatusChanged = "status_changed"
	ActionAlertLinked   = "alert_linked"
	ActionEscalated     = "escalated"
)

type IncidentLog struct {
	ID         uuid.UUID              `json:"id"`
	IncidentID uuid.UUID              `json:"incident_id"`
	UserID     *uuid.UUID             `json:"user_id,omitempty"` // NULL for system/auto actions
	Action     string                 `json:"action"`
	Message    string                 `json:"message"`
	Metadata   map[string]any         `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}
