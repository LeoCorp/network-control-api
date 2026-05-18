package monitoring

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"Network-control-api/internal/models"
	"Network-control-api/internal/websocket"
)

// Config controls monitoring engine behavior.
type Config struct {
	Interval      time.Duration
	DeviceRefresh time.Duration
	ChannelBuffer int
}

// Engine orchestrates concurrent metric generation and in-memory storage.
type Engine struct {
	log      *slog.Logger
	cfg      Config
	provider DeviceProvider
	store    *Store

	metricsSink chan<- MetricEvent
	eventSink   chan<- websocket.Event
	events      chan MetricEvent
	devices []models.Device
	devMu   sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	running atomic.Bool
}

func NewEngine(log *slog.Logger, cfg Config, provider DeviceProvider, store *Store, metricsSink chan<- MetricEvent, eventSink chan<- websocket.Event) *Engine {
	if cfg.ChannelBuffer <= 0 {
		cfg.ChannelBuffer = 64
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 5 * time.Second
	}
	if cfg.DeviceRefresh <= 0 {
		cfg.DeviceRefresh = 30 * time.Second
	}

	return &Engine{
		log:         log,
		cfg:         cfg,
		provider:    provider,
		store:       store,
		metricsSink: metricsSink,
		eventSink:   eventSink,
	}
}

func (e *Engine) Start(parent context.Context) error {
	if !e.running.CompareAndSwap(false, true) {
		return errors.New("monitoring engine is already running")
	}

	e.ctx, e.cancel = context.WithCancel(parent)
	e.events = make(chan MetricEvent, e.cfg.ChannelBuffer)

	if err := e.refreshDevices(); err != nil {
		e.cancel()
		e.running.Store(false)
		return err
	}

	e.wg.Add(3)
	go e.runDeviceRefresher()
	go e.runGenerator()
	go e.runProcessor()

	e.log.Info("monitoring engine started",
		slog.Duration("interval", e.cfg.Interval),
		slog.Duration("device_refresh", e.cfg.DeviceRefresh),
	)
	return nil
}

func (e *Engine) Stop() {
	if !e.running.CompareAndSwap(true, false) {
		return
	}

	e.cancel()
	e.wg.Wait()
	e.log.Info("monitoring engine stopped", slog.Int("metrics_in_memory", e.store.Count()))
}

func (e *Engine) IsRunning() bool {
	return e.running.Load()
}

func (e *Engine) Store() *Store {
	return e.store
}

func (e *Engine) GetRuntimeState(deviceID uuid.UUID) (DeviceRuntimeState, bool) {
	return e.store.Get(deviceID)
}

func (e *Engine) ListRuntimeStates() []DeviceRuntimeState {
	return e.store.List()
}

func (e *Engine) runDeviceRefresher() {
	defer e.wg.Done()

	ticker := time.NewTicker(e.cfg.DeviceRefresh)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			if err := e.refreshDevices(); err != nil {
				e.log.Warn("failed to refresh monitored devices", slog.String("error", err.Error()))
			}
		}
	}
}

func (e *Engine) refreshDevices() error {
	devices, err := e.provider.ListDevices(e.ctx)
	if err != nil {
		return err
	}

	e.devMu.Lock()
	e.devices = devices
	e.devMu.Unlock()

	e.store.SyncDevices(devices)

	e.log.Debug("monitored devices refreshed", slog.Int("count", len(devices)))
	return nil
}

func (e *Engine) runGenerator() {
	defer e.wg.Done()

	ticker := time.NewTicker(e.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.generateForDevices()
		}
	}
}

func (e *Engine) generateForDevices() {
	e.devMu.RLock()
	devices := make([]models.Device, len(e.devices))
	copy(devices, e.devices)
	e.devMu.RUnlock()

	if len(devices) == 0 {
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(devices))

	for _, device := range devices {
		go func(d models.Device) {
			defer wg.Done()
			e.publish(generateMetric(d))
		}(device)
	}

	wg.Wait()
}

func (e *Engine) publish(event MetricEvent) {
	select {
	case <-e.ctx.Done():
		return
	case e.events <- event:
	default:
		e.log.Warn("metric event channel full, dropping event",
			slog.String("device_id", event.DeviceID.String()),
		)
	}
}

func (e *Engine) runProcessor() {
	defer e.wg.Done()

	for {
		select {
		case <-e.ctx.Done():
			e.drainEvents()
			return
		case event := <-e.events:
			e.storeMetric(event)
		}
	}
}

func (e *Engine) drainEvents() {
	for {
		select {
		case event := <-e.events:
			e.storeMetric(event)
		default:
			return
		}
	}
}

func (e *Engine) storeMetric(event MetricEvent) {
	previousStatus, state := e.store.UpdateFromMetric(event)
	e.forwardToAlertEngine(event)
	e.publishRealtimeEvents(event, previousStatus, state)

	e.log.Debug("runtime state updated",
		slog.String("device_id", event.DeviceID.String()),
		slog.String("runtime_status", state.RuntimeStatus),
		slog.Float64("latency_ms", event.Latency),
		slog.Float64("packet_loss", event.PacketLoss),
		slog.Float64("cpu_usage", event.CPUUsage),
	)
}

func (e *Engine) publishRealtimeEvents(event MetricEvent, previousStatus string, state DeviceRuntimeState) {
	websocket.Publish(e.eventSink, websocket.NewEvent(websocket.EventTypeMetric, metricToPayload(event)), e.log)

	if previousStatus != state.RuntimeStatus {
		websocket.Publish(
			e.eventSink,
			websocket.NewEvent(websocket.EventTypeDeviceStatus, stateToStatusPayload(state, previousStatus)),
			e.log,
		)
	}
}

func (e *Engine) forwardToAlertEngine(event MetricEvent) {
	if e.metricsSink == nil {
		return
	}

	select {
	case <-e.ctx.Done():
		return
	case e.metricsSink <- event:
	default:
		e.log.Warn("alert metrics sink full, dropping metric event",
			slog.String("device_id", event.DeviceID.String()),
		)
	}
}
