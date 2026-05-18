package monitoring

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"Network-control-api/internal/models"
	"Network-control-api/internal/repositories"
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

func TestIncidentEngine_AvoidsDuplicateActiveIncidents(t *testing.T) {
	t.Parallel()

	repo := newMockIncidentRepo()
	engine := NewIncidentEngine(slog.Default(), repo, 8, nil)

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
