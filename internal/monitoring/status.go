package monitoring

// Runtime status values derived from live metrics.
const (
	RuntimeStatusOnline  = "ONLINE"
	RuntimeStatusWarning = "WARNING"
	RuntimeStatusDown    = "DOWN"
	RuntimeStatusUnknown = "UNKNOWN"
)

// Thresholds define metric boundaries for runtime status evaluation.
type Thresholds struct {
	WarningLatency    float64
	WarningPacketLoss float64
	WarningCPU        float64
	DownLatency       float64
	DownPacketLoss    float64
	DownCPU           float64
}

func DefaultThresholds() Thresholds {
	return Thresholds{
		WarningLatency:    80,
		WarningPacketLoss: 3,
		WarningCPU:        80,
		DownLatency:       150,
		DownPacketLoss:    8,
		DownCPU:           95,
	}
}

func EvaluateStatus(event MetricEvent, t Thresholds) string {
	if isDown(event, t) {
		return RuntimeStatusDown
	}
	if isWarning(event, t) {
		return RuntimeStatusWarning
	}
	return RuntimeStatusOnline
}

func isDown(event MetricEvent, t Thresholds) bool {
	return event.Latency >= t.DownLatency ||
		event.PacketLoss >= t.DownPacketLoss ||
		event.CPUUsage >= t.DownCPU
}

func isWarning(event MetricEvent, t Thresholds) bool {
	return event.Latency >= t.WarningLatency ||
		event.PacketLoss >= t.WarningPacketLoss ||
		event.CPUUsage >= t.WarningCPU
}
