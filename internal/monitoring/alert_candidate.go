package monitoring

import (
	"time"

	"github.com/google/uuid"
)

// AlertCandidate is an evaluated alert before persistence.
type AlertCandidate struct {
	DeviceID    uuid.UUID
	DeviceName  string
	Severity    string
	Metric      string
	Message     string
	Value       float64
	Threshold   float64
	TriggeredAt time.Time
}
