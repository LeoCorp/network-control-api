package monitoring

import (
	"time"

	"github.com/google/uuid"
)

// DeviceRuntimeState holds live metrics and derived runtime status for a device.
type DeviceRuntimeState struct {
	DeviceID      uuid.UUID   `json:"device_id"`
	DeviceName    string      `json:"device_name"`
	RuntimeStatus string      `json:"runtime_status"`
	Metrics       *MetricEvent `json:"metrics,omitempty"`
	UpdatedAt     time.Time   `json:"updated_at"`
}
