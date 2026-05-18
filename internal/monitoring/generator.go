package monitoring

import (
	"math/rand/v2"
	"time"

	"Network-control-api/internal/models"
)

func generateMetric(device models.Device) MetricEvent {
	return MetricEvent{
		DeviceID:   device.ID,
		DeviceName: device.Name,
		Latency:    round2(rand.Float64() * 250),       // 0–250 ms
		PacketLoss: round2(rand.Float64() * 50),        // 0–50 %
		CPUUsage:   round2(rand.Float64() * 100),       // 0–100 %
		Timestamp:  time.Now().UTC(),
	}
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
