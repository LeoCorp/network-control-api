package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	AlertSeverityInfo     = "INFO"
	AlertSeverityWarning  = "WARNING"
	AlertSeverityCritical = "CRITICAL"
)

var ValidAlertSeverities = map[string]bool{
	AlertSeverityInfo:     true,
	AlertSeverityWarning:  true,
	AlertSeverityCritical: true,
}

type Alert struct {
	ID         uuid.UUID `json:"id"`
	DeviceID   uuid.UUID `json:"device_id"`
	DeviceName string    `json:"device_name"`
	Severity   string    `json:"severity"`
	Metric     string    `json:"metric"`
	Message    string    `json:"message"`
	Value      float64   `json:"value"`
	Threshold  float64   `json:"threshold"`
	CreatedAt  time.Time `json:"created_at"`
}

func IsValidAlertSeverity(severity string) bool {
	return ValidAlertSeverities[severity]
}
