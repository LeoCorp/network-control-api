package monitoring

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"

	"Network-control-api/internal/models"
	"Network-control-api/internal/repositories"
	"Network-control-api/internal/websocket"
)

// AlertEngine evaluates metrics and persists alerts asynchronously.
type AlertEngine struct {
	log   *slog.Logger
	repo  repositories.AlertRepository
	rules []AlertRule

	criticalSink chan<- models.Alert
	eventSink    chan<- websocket.Event
	metricsCh    chan MetricEvent
	alertsCh     chan AlertCandidate

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	running atomic.Bool
}

func NewAlertEngine(log *slog.Logger, repo repositories.AlertRepository, buffer int, rules []AlertRule, criticalSink chan<- models.Alert, eventSink chan<- websocket.Event) *AlertEngine {
	if buffer <= 0 {
		buffer = 64
	}
	if len(rules) == 0 {
		rules = DefaultAlertRules()
	}

	return &AlertEngine{
		log:          log,
		repo:         repo,
		rules:        rules,
		criticalSink: criticalSink,
		eventSink:    eventSink,
		metricsCh:    make(chan MetricEvent, buffer),
		alertsCh:     make(chan AlertCandidate, buffer),
	}
}

func (e *AlertEngine) MetricsSink() chan<- MetricEvent {
	return e.metricsCh
}

func (e *AlertEngine) Start(parent context.Context) error {
	if !e.running.CompareAndSwap(false, true) {
		return errors.New("alert engine is already running")
	}

	e.ctx, e.cancel = context.WithCancel(parent)

	e.wg.Add(2)
	go e.runEvaluator()
	go e.runPersister()

	e.log.Info("alert engine started")
	return nil
}

func (e *AlertEngine) Stop() {
	if !e.running.CompareAndSwap(true, false) {
		return
	}

	e.cancel()
	e.wg.Wait()
	e.log.Info("alert engine stopped")
}

func (e *AlertEngine) IsRunning() bool {
	return e.running.Load()
}

func (e *AlertEngine) runEvaluator() {
	defer e.wg.Done()

	for {
		select {
		case <-e.ctx.Done():
			e.drainMetrics()
			return
		case event := <-e.metricsCh:
			e.evaluate(event)
		}
	}
}

func (e *AlertEngine) drainMetrics() {
	for {
		select {
		case event := <-e.metricsCh:
			e.evaluate(event)
		default:
			return
		}
	}
}

func (e *AlertEngine) evaluate(event MetricEvent) {
	candidates := EvaluateAlerts(event, e.rules)
	for _, candidate := range candidates {
		select {
		case <-e.ctx.Done():
			return
		case e.alertsCh <- candidate:
		default:
			e.log.Warn("alert channel full, dropping alert",
				slog.String("device_id", candidate.DeviceID.String()),
				slog.String("metric", candidate.Metric),
			)
		}
	}
}

func (e *AlertEngine) runPersister() {
	defer e.wg.Done()

	for {
		select {
		case <-e.ctx.Done():
			e.drainAlerts()
			return
		case candidate, ok := <-e.alertsCh:
			if !ok {
				return
			}
			e.persist(candidate)
		}
	}
}

func (e *AlertEngine) drainAlerts() {
	for {
		select {
		case candidate, ok := <-e.alertsCh:
			if !ok {
				return
			}
			e.persist(candidate)
		default:
			return
		}
	}
}

func (e *AlertEngine) persist(candidate AlertCandidate) {
	alert := &models.Alert{
		ID:         uuid.New(),
		DeviceID:   candidate.DeviceID,
		DeviceName: candidate.DeviceName,
		Severity:   candidate.Severity,
		Metric:     candidate.Metric,
		Message:    candidate.Message,
		Value:      candidate.Value,
		Threshold:  candidate.Threshold,
		CreatedAt:  candidate.TriggeredAt,
	}

	ctx := e.ctx
	if ctx.Err() != nil {
		ctx = context.Background()
	}

	if err := e.repo.Create(ctx, alert); err != nil {
		e.log.Error("failed to persist alert",
			slog.String("error", err.Error()),
			slog.String("device_id", candidate.DeviceID.String()),
			slog.String("severity", candidate.Severity),
		)
		return
	}

	e.log.Info("alert persisted",
		slog.String("alert_id", alert.ID.String()),
		slog.String("device_id", alert.DeviceID.String()),
		slog.String("severity", alert.Severity),
		slog.String("metric", alert.Metric),
	)

	websocket.Publish(e.eventSink, websocket.NewEvent(websocket.EventTypeAlert, *alert), e.log)
	e.forwardCriticalAlert(*alert)
}

func (e *AlertEngine) forwardCriticalAlert(alert models.Alert) {
	if alert.Severity != models.AlertSeverityCritical || e.criticalSink == nil {
		return
	}

	select {
	case <-e.ctx.Done():
		return
	case e.criticalSink <- alert:
	default:
		e.log.Warn("critical alert channel full, dropping incident trigger",
			slog.String("alert_id", alert.ID.String()),
			slog.String("device_id", alert.DeviceID.String()),
		)
	}
}
