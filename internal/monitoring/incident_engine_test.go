package monitoring

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"Network-control-api/internal/models"
	"Network-control-api/internal/repositories"
	"Network-control-api/internal/websocket"
)

type mockIncidentRepo struct {
	mu        sync.Mutex
	incidents map[uuid.UUID]*models.Incident
	links     map[uuid.UUID]uuid.UUID
}

func newMockIncidentRepo() *mockIncidentRepo {
	return &mockIncidentRepo{
		incidents: make(map[uuid.UUID]*models.Incident),
		links:     make(map[uuid.UUID]uuid.UUID),
	}
}

func (m *mockIncidentRepo) FindActiveByDevice(_ context.Context, deviceID uuid.UUID) (*models.Incident, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, incident := range m.incidents {
		if incident.DeviceID == deviceID && models.IsActiveIncidentStatus(incident.Status) {
			return incident, nil
		}
	}
	return nil, repositories.ErrNotFound
}

func (m *mockIncidentRepo) CreateWithAlert(_ context.Context, incident *models.Incident, alertID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, existing := range m.incidents {
		if existing.DeviceID == incident.DeviceID && models.IsActiveIncidentStatus(existing.Status) {
			return repositories.ErrDuplicateActiveIncident
		}
	}

	m.incidents[incident.ID] = incident
	m.links[alertID] = incident.ID
	return nil
}

func (m *mockIncidentRepo) LinkAlert(_ context.Context, incidentID, alertID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.links[alertID] = incidentID
	return nil
}

func (m *mockIncidentRepo) FindByID(_ context.Context, id uuid.UUID) (*models.Incident, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	inc, ok := m.incidents[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	return inc, nil
}

func (m *mockIncidentRepo) List(_ context.Context, status string, deviceID *uuid.UUID, page, limit int) (*repositories.PaginatedResult[models.Incident], error) {
	return nil, nil
}

func (m *mockIncidentRepo) UpdateStatus(_ context.Context, id uuid.UUID, status string, resolvedAt *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	inc, ok := m.incidents[id]
	if !ok {
		return repositories.ErrNotFound
	}
	inc.Status = status
	inc.ResolvedAt = resolvedAt
	inc.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *mockIncidentRepo) Escalate(_ context.Context, id uuid.UUID, escalatedTitle string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	inc, ok := m.incidents[id]
	if !ok {
		return repositories.ErrNotFound
	}
	inc.Escalated = true
	inc.Title = escalatedTitle
	inc.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *mockIncidentRepo) GetLinkedAlerts(_ context.Context, incidentID uuid.UUID) ([]models.Alert, error) {
	return nil, nil
}

func (m *mockIncidentRepo) CreateLog(_ context.Context, log *models.IncidentLog) error {
	return nil
}

func (m *mockIncidentRepo) GetLogs(_ context.Context, incidentID uuid.UUID) ([]models.IncidentLog, error) {
	return nil, nil
}

func (m *mockIncidentRepo) FindAllActive(_ context.Context) ([]models.Incident, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var active []models.Incident
	for _, inc := range m.incidents {
		if models.IsActiveIncidentStatus(inc.Status) {
			active = append(active, *inc)
		}
	}
	return active, nil
}

type mockDeviceRepo struct {
	mu      sync.Mutex
	devices map[uuid.UUID]*models.Device
}

func newMockDeviceRepo() *mockDeviceRepo {
	return &mockDeviceRepo{
		devices: make(map[uuid.UUID]*models.Device),
	}
}

func (m *mockDeviceRepo) Create(_ context.Context, device *models.Device) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.devices[device.ID] = device
	return nil
}

func (m *mockDeviceRepo) FindByID(_ context.Context, id uuid.UUID) (*models.Device, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.devices[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	return d, nil
}

func (m *mockDeviceRepo) Update(_ context.Context, device *models.Device) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.devices[device.ID] = device
	return nil
}

func (m *mockDeviceRepo) UpdateStatus(_ context.Context, id uuid.UUID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.devices[id]
	if !ok {
		m.devices[id] = &models.Device{ID: id, Status: status}
		return nil
	}
	d.Status = status
	return nil
}

func (m *mockDeviceRepo) Delete(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.devices, id)
	return nil
}

func (m *mockDeviceRepo) List(_ context.Context, filter repositories.DeviceListFilter) (*repositories.PaginatedResult[models.Device], error) {
	return nil, nil
}

func TestIncidentEngine_AvoidsDuplicateActiveIncidents(t *testing.T) {
	t.Parallel()

	repo := newMockIncidentRepo()
	deviceRepo := newMockDeviceRepo()
	engine := NewIncidentEngine(slog.Default(), repo, deviceRepo, 8, 30, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := engine.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer engine.Stop()

	deviceID := uuid.New()
	alert1 := models.Alert{
		ID:         uuid.New(),
		DeviceID:   deviceID,
		DeviceName: "router-1",
		Severity:   models.AlertSeverityCritical,
		Metric:     "packet_loss",
		Message:    "packet_loss exceeded threshold",
		Value:      40,
		Threshold:  30,
		CreatedAt:  time.Now().UTC(),
	}
	alert2 := alert1
	alert2.ID = uuid.New()

	engine.CriticalAlertsSink() <- alert1
	engine.CriticalAlertsSink() <- alert2

	time.Sleep(100 * time.Millisecond)

	repo.mu.Lock()
	defer repo.mu.Unlock()

	if len(repo.incidents) != 1 {
		t.Fatalf("expected 1 active incident, got %d", len(repo.incidents))
	}
	if len(repo.links) != 2 {
		t.Fatalf("expected 2 linked alerts, got %d", len(repo.links))
	}
}

func TestIncidentEngine_EscalationWorker(t *testing.T) {
	t.Parallel()

	repo := newMockIncidentRepo()
	deviceRepo := newMockDeviceRepo()
	eventSink := make(chan websocket.Event, 10)

	// Set escalation time to 1 second
	engine := NewIncidentEngine(slog.Default(), repo, deviceRepo, 8, 1, eventSink)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	deviceID := uuid.New()
	incidentID := uuid.New()
	now := time.Now().UTC()

	// Pre-populate mock repo with an active incident created 2 seconds ago
	incident := &models.Incident{
		ID:          incidentID,
		DeviceID:    deviceID,
		DeviceName:  "switch-1",
		Title:       "High CPU Usage",
		Description: "CPU has been above 90% for a long time",
		Status:      models.IncidentStatusOpen,
		Escalated:   false,
		CreatedAt:   now.Add(-2 * time.Second),
		UpdatedAt:   now.Add(-2 * time.Second),
	}
	repo.incidents[incidentID] = incident

	// Also make sure device is ONLINE in mock device repo
	deviceRepo.devices[deviceID] = &models.Device{
		ID:     deviceID,
		Name:   "switch-1",
		Status: "online",
	}

	if err := engine.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer engine.Stop()

	// The ticker runs every 5 seconds. Let's wait 6 seconds to ensure at least one tick has completed.
	time.Sleep(6 * time.Second)

	repo.mu.Lock()
	escalatedInc, exists := repo.incidents[incidentID]
	repo.mu.Unlock()

	if !exists {
		t.Fatalf("expected incident to exist")
	}

	if !escalatedInc.Escalated {
		t.Errorf("expected incident to be escalated")
	}

	expectedPrefix := "[ESCALATED] "
	if !strings.HasPrefix(escalatedInc.Title, expectedPrefix) {
		t.Errorf("expected incident title to start with %q, got %q", expectedPrefix, escalatedInc.Title)
	}

	// Verify device status in mock device repository changed to "offline"
	deviceRepo.mu.Lock()
	dev, devExists := deviceRepo.devices[deviceID]
	deviceRepo.mu.Unlock()

	if !devExists {
		t.Fatalf("expected device to exist in mock repo")
	}
	if dev.Status != "offline" {
		t.Errorf("expected device status to be 'offline', got %q", dev.Status)
	}
}
