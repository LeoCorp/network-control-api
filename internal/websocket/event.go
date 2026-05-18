package websocket

import (
	"time"

	"github.com/google/uuid"

	"Network-control-api/internal/models"
)

// Event types broadcast to websocket clients.
const (
	EventTypeMetric       = "metric"
	EventTypeDeviceStatus = "device_status"
	EventTypeAlert        = "alert"
	EventTypeIncident     = "incident"
)

// Event is a realtime message sent to connected clients.
type Event struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Payload   any       `json:"payload"`
}

type MetricPayload struct {
	DeviceID   uuid.UUID `json:"device_id"`
	DeviceName string    `json:"device_name"`
	Latency    float64   `json:"latency_ms"`
	PacketLoss float64   `json:"packet_loss"`
	CPUUsage   float64   `json:"cpu_usage"`
	Timestamp  time.Time `json:"timestamp"`
}

type DeviceStatusPayload struct {
	DeviceID       uuid.UUID      `json:"device_id"`
	DeviceName     string         `json:"device_name"`
	RuntimeStatus  string         `json:"runtime_status"`
	PreviousStatus string         `json:"previous_status"`
	Metrics        *MetricPayload `json:"metrics,omitempty"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type IncidentPayload struct {
	Incident models.Incident `json:"incident"`
	AlertID  string          `json:"alert_id,omitempty"`
	Action   string          `json:"action"`
}

func NewEvent(eventType string, payload any) Event {
	return Event{
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}
}
