package monitoring

import (
	"testing"

	"github.com/google/uuid"
)

func TestEvaluateStatus(t *testing.T) {
	t.Parallel()

	thresholds := DefaultThresholds()
	base := MetricEvent{DeviceID: uuid.New()}

	tests := []struct {
		name   string
		event  MetricEvent
		expect string
	}{
		{
			name:   "online",
			event:  base,
			expect: RuntimeStatusOnline,
		},
		{
			name: "warning latency",
			event: MetricEvent{
				DeviceID: base.DeviceID,
				Latency:  90,
			},
			expect: RuntimeStatusWarning,
		},
		{
			name: "down packet loss",
			event: MetricEvent{
				DeviceID:   base.DeviceID,
				PacketLoss: 10,
			},
			expect: RuntimeStatusDown,
		},
		{
			name: "down cpu",
			event: MetricEvent{
				DeviceID: base.DeviceID,
				CPUUsage: 99,
			},
			expect: RuntimeStatusDown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := EvaluateStatus(tt.event, thresholds); got != tt.expect {
				t.Fatalf("EvaluateStatus() = %q, want %q", got, tt.expect)
			}
		})
	}
}
