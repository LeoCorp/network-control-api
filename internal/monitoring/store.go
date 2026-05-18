package monitoring

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"Network-control-api/internal/models"
)

// Store keeps live runtime state per device in memory.
type Store struct {
	mu         sync.RWMutex
	states     map[uuid.UUID]DeviceRuntimeState
	thresholds Thresholds
}

func NewStore() *Store {
	return &Store{
		states:     make(map[uuid.UUID]DeviceRuntimeState),
		thresholds: DefaultThresholds(),
	}
}

func (s *Store) SyncDevices(devices []models.Device) {
	s.mu.Lock()
	defer s.mu.Unlock()

	seen := make(map[uuid.UUID]struct{}, len(devices))
	now := time.Now().UTC()

	for _, device := range devices {
		seen[device.ID] = struct{}{}

		state, ok := s.states[device.ID]
		if !ok {
			s.states[device.ID] = DeviceRuntimeState{
				DeviceID:      device.ID,
				DeviceName:    device.Name,
				RuntimeStatus: RuntimeStatusUnknown,
				UpdatedAt:     now,
			}
			continue
		}

		state.DeviceName = device.Name
		s.states[device.ID] = state
	}

	for id := range s.states {
		if _, ok := seen[id]; !ok {
			delete(s.states, id)
		}
	}
}

func (s *Store) UpdateFromMetric(event MetricEvent) (previousStatus string, state DeviceRuntimeState) {
	status := EvaluateStatus(event, s.thresholds)
	metrics := event

	s.mu.Lock()
	defer s.mu.Unlock()

	previousStatus = RuntimeStatusUnknown
	if prev, ok := s.states[event.DeviceID]; ok {
		previousStatus = prev.RuntimeStatus
	}

	state = DeviceRuntimeState{
		DeviceID:      event.DeviceID,
		DeviceName:    event.DeviceName,
		RuntimeStatus: status,
		Metrics:       &metrics,
		UpdatedAt:     event.Timestamp,
	}
	s.states[event.DeviceID] = state

	return previousStatus, state
}

func (s *Store) Get(deviceID uuid.UUID) (DeviceRuntimeState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.states[deviceID]
	return state, ok
}

func (s *Store) List() []DeviceRuntimeState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	states := make([]DeviceRuntimeState, 0, len(s.states))
	for _, state := range s.states {
		states = append(states, state)
	}
	return states
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.states)
}
