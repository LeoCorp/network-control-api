package monitoring

import (
	"Network-control-api/internal/models"
	"Network-control-api/internal/websocket"
)

func metricToPayload(event MetricEvent) websocket.MetricPayload {
	return websocket.MetricPayload{
		DeviceID:   event.DeviceID,
		DeviceName: event.DeviceName,
		Latency:    event.Latency,
		PacketLoss: event.PacketLoss,
		CPUUsage:   event.CPUUsage,
		Timestamp:  event.Timestamp,
	}
}

func stateToStatusPayload(state DeviceRuntimeState, previousStatus string) websocket.DeviceStatusPayload {
	var metrics *websocket.MetricPayload
	if state.Metrics != nil {
		payload := metricToPayload(*state.Metrics)
		metrics = &payload
	}

	return websocket.DeviceStatusPayload{
		DeviceID:       state.DeviceID,
		DeviceName:     state.DeviceName,
		RuntimeStatus:  state.RuntimeStatus,
		PreviousStatus: previousStatus,
		Metrics:        metrics,
		UpdatedAt:      state.UpdatedAt,
	}
}

func incidentPayload(incident models.Incident, alertID, action string) websocket.IncidentPayload {
	return websocket.IncidentPayload{
		Incident: incident,
		AlertID:  alertID,
		Action:   action,
	}
}
