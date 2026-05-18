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

// IncidentEngine creates incidents asynchronously from critical alerts.
type IncidentEngine struct {
	log  *slog.Logger
	repo repositories.IncidentRepository

	criticalCh chan models.Alert
	eventSink  chan<- websocket.Event

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	running atomic.Bool
}

func NewIncidentEngine(log *slog.Logger, repo repositories.IncidentRepository, buffer int, eventSink chan<- websocket.Event) *IncidentEngine {
	if buffer <= 0 {
		buffer = 64
	}

	return &IncidentEngine{
		log:        log,
		repo:       repo,
		criticalCh: make(chan models.Alert, buffer),
		eventSink:  eventSink,
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
	e.wg.Add(1)
	go e.runProcessor()

	e.log.Info("incident engine started")
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
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
