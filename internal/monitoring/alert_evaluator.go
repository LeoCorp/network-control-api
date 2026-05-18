package monitoring

import (
	"fmt"
	"time"

	"Network-control-api/internal/models"
)

type AlertRule struct {
	Metric    string
	Threshold float64
	Severity  string
}

func DefaultAlertRules() []AlertRule {
	return []AlertRule{
		{Metric: "latency", Threshold: 200, Severity: models.AlertSeverityWarning},
		{Metric: "packet_loss", Threshold: 30, Severity: models.AlertSeverityCritical},
		{Metric: "cpu_usage", Threshold: 90, Severity: models.AlertSeverityWarning},
	}
}

func EvaluateAlerts(event MetricEvent, rules []AlertRule) []AlertCandidate {
	if len(rules) == 0 {
		rules = DefaultAlertRules()
	}

	candidates := make([]AlertCandidate, 0)
	triggeredAt := event.Timestamp
	if triggeredAt.IsZero() {
		triggeredAt = time.Now().UTC()
	}

	for _, rule := range rules {
		value, ok := metricValue(event, rule.Metric)
		if !ok || value <= rule.Threshold {
			continue
		}

		candidates = append(candidates, AlertCandidate{
			DeviceID:    event.DeviceID,
			DeviceName:  event.DeviceName,
			Severity:    rule.Severity,
			Metric:      rule.Metric,
			Message:     formatAlertMessage(rule, value),
			Value:       value,
			Threshold:   rule.Threshold,
			TriggeredAt: triggeredAt,
		})
	}

	return candidates
}

func metricValue(event MetricEvent, metric string) (float64, bool) {
	switch metric {
	case "latency":
		return event.Latency, true
	case "packet_loss":
		return event.PacketLoss, true
	case "cpu_usage":
		return event.CPUUsage, true
	default:
		return 0, false
	}
}

func formatAlertMessage(rule AlertRule, value float64) string {
	unit := ""
	switch rule.Metric {
	case "latency":
		unit = "ms"
	case "packet_loss", "cpu_usage":
		unit = "%"
	}

	if unit != "" {
		return fmt.Sprintf("%s exceeded threshold: %.2f%s > %.2f%s",
			rule.Metric, value, unit, rule.Threshold, unit)
	}

	return fmt.Sprintf("%s exceeded threshold: %.2f > %.2f",
		rule.Metric, value, rule.Threshold)
}
