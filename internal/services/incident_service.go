package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"Network-control-api/internal/models"
	"Network-control-api/internal/repositories"
	"Network-control-api/internal/websocket"
)

type IncidentDetails struct {
	Incident *models.Incident     `json:"incident"`
	Alerts   []models.Alert       `json:"alerts"`
	Logs     []models.IncidentLog `json:"logs"`
}

type IncidentService struct {
	repo       repositories.IncidentRepository
	deviceRepo repositories.DeviceRepository
	eventSink  chan<- websocket.Event
}

func NewIncidentService(repo repositories.IncidentRepository, deviceRepo repositories.DeviceRepository, eventSink chan<- websocket.Event) *IncidentService {
	return &IncidentService{
		repo:       repo,
		deviceRepo: deviceRepo,
		eventSink:  eventSink,
	}
}

func (s *IncidentService) ListIncidents(ctx context.Context, status string, deviceID *uuid.UUID, page, limit int) (*repositories.PaginatedResult[models.Incident], error) {
	return s.repo.List(ctx, status, deviceID, page, limit)
}

func (s *IncidentService) GetIncidentDetails(ctx context.Context, id uuid.UUID) (*IncidentDetails, error) {
	incident, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	alerts, err := s.repo.GetLinkedAlerts(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get linked alerts: %w", err)
	}

	logs, err := s.repo.GetLogs(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get incident logs: %w", err)
	}

	return &IncidentDetails{
		Incident: incident,
		Alerts:   alerts,
		Logs:     logs,
	}, nil
}

func (s *IncidentService) UpdateIncidentStatus(ctx context.Context, id uuid.UUID, newStatus string, userID uuid.UUID) (*models.Incident, error) {
	if !models.IsValidIncidentStatus(newStatus) {
		return nil, fmt.Errorf("invalid incident status: %s", newStatus)
	}

	incident, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if incident.Status == newStatus {
		return incident, nil
	}

	if incident.Status == models.IncidentStatusResolved {
		return nil, errors.New("cannot reopen a resolved incident")
	}

	oldStatus := incident.Status
	incident.Status = newStatus
	incident.UpdatedAt = time.Now().UTC()

	var resolvedAt *time.Time
	if newStatus == models.IncidentStatusResolved {
		now := time.Now().UTC()
		resolvedAt = &now
		incident.ResolvedAt = resolvedAt
	}

	if err := s.repo.UpdateStatus(ctx, id, newStatus, resolvedAt); err != nil {
		return nil, err
	}

	// Log the status transition
	logEntry := &models.IncidentLog{
		ID:         uuid.New(),
		IncidentID: id,
		UserID:     &userID,
		Action:     models.ActionStatusChanged,
		Message:    fmt.Sprintf("Incident status updated from %s to %s.", oldStatus, newStatus),
		Metadata: map[string]any{
			"old_status": oldStatus,
			"new_status": newStatus,
		},
		CreatedAt: time.Now().UTC(),
	}
	_ = s.repo.CreateLog(ctx, logEntry)

	// Sync device status in PostgreSQL
	if newStatus == models.IncidentStatusResolved {
		// Check if there are other active incidents for the device
		activeInc, err := s.repo.FindActiveByDevice(ctx, incident.DeviceID)
		if err != nil && !errors.Is(err, repositories.ErrNotFound) {
			return nil, fmt.Errorf("check active incidents: %w", err)
		}
		if activeInc == nil {
			// No other active incidents! Reset status to online
			_ = s.deviceRepo.UpdateStatus(ctx, incident.DeviceID, "online")
		} else {
			// Still active incidents. Keep degraded/offline
			targetStatus := "degraded"
			if activeInc.Escalated {
				targetStatus = "offline"
			}
			_ = s.deviceRepo.UpdateStatus(ctx, incident.DeviceID, targetStatus)
		}
	} else if newStatus == models.IncidentStatusInvestigating {
		// Operator took ownership, device is degraded
		_ = s.deviceRepo.UpdateStatus(ctx, incident.DeviceID, "degraded")
	}

	// Publish WebSocket event
	if s.eventSink != nil {
		websocket.Publish(
			s.eventSink,
			websocket.NewEvent(websocket.EventTypeIncident, map[string]any{
				"id":          incident.ID.String(),
				"device_id":   incident.DeviceID.String(),
				"device_name": incident.DeviceName,
				"title":       incident.Title,
				"status":      incident.Status,
				"escalated":   incident.Escalated,
				"action":      "updated",
			}),
			nil,
		)
	}

	return incident, nil
}
