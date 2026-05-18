package monitoring

import (
	"testing"

	"github.com/google/uuid"

	"Network-control-api/internal/models"
)

func TestEvaluateAlerts(t *testing.T) {
	t.Parallel()

	event := MetricEvent{
		DeviceID:   uuid.New(),
		DeviceName: "router-1",
		Latency:    210,
		PacketLoss: 35,
		CPUUsage:   95,
	}

	candidates := EvaluateAlerts(event, DefaultAlertRules())
	if len(candidates) != 3 {
		t.Fatalf("expected 3 alerts, got %d", len(candidates))
	}

	severities := map[string]bool{}
	for _, c := range candidates {
		severities[c.Severity] = true
	}

	if !severities[models.AlertSeverityWarning] || !severities[models.AlertSeverityCritical] {
		t.Fatalf("unexpected severities: %+v", severities)
	}
}

func TestEvaluateAlerts_NoBreach(t *testing.T) {
	t.Parallel()

	event := MetricEvent{
		DeviceID:   uuid.New(),
		Latency:    50,
		PacketLoss: 1,
		CPUUsage:   40,
	}

	candidates := EvaluateAlerts(event, DefaultAlertRules())
	if len(candidates) != 0 {
		t.Fatalf("expected no alerts, got %d", len(candidates))
	}
}
