package monitoring

import (
	"time"

	"github.com/google/uuid"
)

// MetricEvent represents a single simulated telemetry reading for a device.
type MetricEvent struct {
	DeviceID   uuid.UUID `json:"device_id"`
	DeviceName string    `json:"device_name"`
	Latency    float64   `json:"latency_ms"`
	PacketLoss float64   `json:"packet_loss"`
	CPUUsage   float64   `json:"cpu_usage"`
	Timestamp  time.Time `json:"timestamp"`
}
