package monitoring

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"Network-control-api/internal/models"
	"Network-control-api/internal/repositories"
	"Network-control-api/internal/websocket"
)

// IncidentEngine creates incidents asynchronously from critical alerts and manages escalation.
type IncidentEngine struct {
	log               *slog.Logger
	repo              repositories.IncidentRepository
	deviceRepo        repositories.DeviceRepository
	escalationSeconds int

	criticalCh chan models.Alert
	eventSink  chan<- websocket.Event

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	running atomic.Bool
}

func NewIncidentEngine(log *slog.Logger, repo repositories.IncidentRepository, deviceRepo repositories.DeviceRepository, buffer int, escalationSeconds int, eventSink chan<- websocket.Event) *IncidentEngine {
	if buffer <= 0 {
		buffer = 64
	}
	if escalationSeconds <= 0 {
		escalationSeconds = 30
	}

	return &IncidentEngine{
		log:               log,
		repo:              repo,
		deviceRepo:        deviceRepo,
		escalationSeconds: escalationSeconds,
		criticalCh:        make(chan models.Alert, buffer),
		eventSink:         eventSink,
	}
}

func (e *IncidentEngine) CriticalAlertsSink() chan<- models.Alert {
	return e.criticalCh
}

func (e *IncidentEngine) Start(parent context.Context) error {
	if !e.running.CompareAndSwap(false, true) {
		return errors.New("incident engine is already running")
	}

	e.ctx, e.cancel = context.WithCancel(parent)
	e.wg.Add(2)
	go e.runProcessor()
	go e.runEscalationWorker()

	e.log.Info("incident engine started", slog.Int("escalation_seconds", e.escalationSeconds))
	return nil
}

func (e *IncidentEngine) Stop() {
	if !e.running.CompareAndSwap(true, false) {
		return
	}

	e.cancel()
	e.wg.Wait()
	e.log.Info("incident engine stopped")
}

func (e *IncidentEngine) IsRunning() bool {
	return e.running.Load()
}

func (e *IncidentEngine) runProcessor() {
	defer e.wg.Done()

	for {
		select {
		case <-e.ctx.Done():
			e.drainCriticalAlerts()
			return
		case alert := <-e.criticalCh:
			go e.processCriticalAlert(alert)
		}
	}
}

func (e *IncidentEngine) runEscalationWorker() {
	defer e.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.escalateActiveIncidents()
		}
	}
}

func (e *IncidentEngine) escalateActiveIncidents() {
	activeIncidents, err := e.repo.FindAllActive(e.ctx)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			e.log.Error("failed to find active incidents for escalation", slog.String("error", err.Error()))
		}
		return
	}

	now := time.Now().UTC()
	escalationThreshold := time.Duration(e.escalationSeconds) * time.Second

	for _, incident := range activeIncidents {
		if incident.Escalated || incident.Status != models.IncidentStatusOpen {
			continue
		}

		age := now.Sub(incident.CreatedAt)
		if age >= escalationThreshold {
			e.log.Info("escalating incident automatically",
				slog.String("incident_id", incident.ID.String()),
				slog.Duration("age", age),
			)

			escalatedTitle := "[ESCALATED] " + incident.Title
			if err := e.repo.Escalate(e.ctx, incident.ID, escalatedTitle); err != nil {
				e.log.Error("failed to escalate incident",
					slog.String("incident_id", incident.ID.String()),
					slog.String("error", err.Error()),
				)
				continue
			}

			// Write escalation audit log with JSONB metadata
			logEntry := &models.IncidentLog{
				ID:         uuid.New(),
				IncidentID: incident.ID,
				UserID:     nil,
				Action:     models.ActionEscalated,
				Message:    "Incident automatically escalated due to response delay.",
				Metadata: map[string]any{
					"escalation_seconds": e.escalationSeconds,
					"age_seconds":        int(age.Seconds()),
				},
				CreatedAt: now,
			}
			if err := e.repo.CreateLog(e.ctx, logEntry); err != nil {
				e.log.Warn("failed to create escalation audit log", slog.String("error", err.Error()))
			}

			// Update device status in PostgreSQL to offline due to escalation
			if err := e.deviceRepo.UpdateStatus(e.ctx, incident.DeviceID, "offline"); err != nil {
				e.log.Error("failed to update device status to offline on escalation",
					slog.String("device_id", incident.DeviceID.String()),
					slog.String("error", err.Error()),
				)
			}

			incident.Escalated = true
			incident.Title = escalatedTitle
			incident.UpdatedAt = now

			e.publishIncident(incident, "", "escalated")
		}
	}
}

func (e *IncidentEngine) drainCriticalAlerts() {
	for {
		select {
		case alert := <-e.criticalCh:
			go e.processCriticalAlert(alert)
		default:
			return
		}
	}
}

func (e *IncidentEngine) processCriticalAlert(alert models.Alert) {
	ctx := e.ctx
	if ctx.Err() != nil {
		ctx = context.Background()
	}

	active, err := e.repo.FindActiveByDevice(ctx, alert.DeviceID)
	if err != nil && !errors.Is(err, repositories.ErrNotFound) {
		e.log.Error("failed to find active incident",
			slog.String("error", err.Error()),
			slog.String("device_id", alert.DeviceID.String()),
		)
		return
	}

	if active != nil {
		if err := e.repo.LinkAlert(ctx, active.ID, alert.ID); err != nil {
			e.log.Error("failed to link alert to active incident",
				slog.String("error", err.Error()),
				slog.String("incident_id", active.ID.String()),
				slog.String("alert_id", alert.ID.String()),
			)
			return
		}

		e.log.Info("critical alert linked to active incident",
			slog.String("incident_id", active.ID.String()),
			slog.String("alert_id", alert.ID.String()),
			slog.String("device_id", alert.DeviceID.String()),
		)
		e.publishIncident(*active, alert.ID.String(), "linked")
		return
	}

	incident := buildIncidentFromAlert(alert)
	if err := e.repo.CreateWithAlert(ctx, incident, alert.ID); err != nil {
		if errors.Is(err, repositories.ErrDuplicateActiveIncident) {
			e.linkToActiveIncident(ctx, alert)
			return
		}

		e.log.Error("failed to create incident from critical alert",
			slog.String("error", err.Error()),
			slog.String("alert_id", alert.ID.String()),
		)
		return
	}

	// Update device status in PostgreSQL to degraded on incident creation
	if err := e.deviceRepo.UpdateStatus(ctx, alert.DeviceID, "degraded"); err != nil {
		e.log.Warn("failed to update device status to degraded on incident creation",
			slog.String("device_id", alert.DeviceID.String()),
			slog.String("error", err.Error()),
		)
	}

	e.log.Info("incident created from critical alert",
		slog.String("incident_id", incident.ID.String()),
		slog.String("alert_id", alert.ID.String()),
		slog.String("device_id", alert.DeviceID.String()),
	)
	e.publishIncident(*incident, alert.ID.String(), "created")
}

func (e *IncidentEngine) linkToActiveIncident(ctx context.Context, alert models.Alert) {
	active, err := e.repo.FindActiveByDevice(ctx, alert.DeviceID)
	if err != nil {
		e.log.Warn("could not link alert after duplicate incident race",
			slog.String("alert_id", alert.ID.String()),
			slog.String("error", err.Error()),
		)
		return
	}

	if err := e.repo.LinkAlert(ctx, active.ID, alert.ID); err != nil {
		e.log.Error("failed to link alert after duplicate incident race",
			slog.String("error", err.Error()),
			slog.String("incident_id", active.ID.String()),
			slog.String("alert_id", alert.ID.String()),
		)
		return
	}

	e.log.Info("critical alert linked after incident race",
		slog.String("incident_id", active.ID.String()),
		slog.String("alert_id", alert.ID.String()),
	)
	e.publishIncident(*active, alert.ID.String(), "linked")
}

func (e *IncidentEngine) publishIncident(incident models.Incident, alertID, action string) {
	websocket.Publish(
		e.eventSink,
		websocket.NewEvent(websocket.EventTypeIncident, incidentPayload(incident, alertID, action)),
		e.log,
	)
}

func buildIncidentFromAlert(alert models.Alert) *models.Incident {
	now := time.Now().UTC()
	return &models.Incident{
		ID:          uuid.New(),
		DeviceID:    alert.DeviceID,
		DeviceName:  alert.DeviceName,
		Title:       fmt.Sprintf("Critical %s on %s", alert.Metric, alert.DeviceName),
		Description: alert.Message,
		Status:      models.IncidentStatusOpen,
		Escalated:   false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}


