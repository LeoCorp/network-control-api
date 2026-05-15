package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"Network-control-api/internal/models"
	"Network-control-api/internal/repositories"
)

var (
	ErrInvalidDeviceType   = errors.New("invalid device type")
	ErrInvalidDeviceStatus = errors.New("invalid device status")
)

type DeviceService struct {
	devices repositories.DeviceRepository
}

func NewDeviceService(devices repositories.DeviceRepository) *DeviceService {
	return &DeviceService{devices: devices}
}

type CreateDeviceInput struct {
	Name        string
	Type        string
	Status      string
	Location    string
	IPAddress   string
	Description string
}

type UpdateDeviceInput struct {
	Name        *string
	Type        *string
	Status      *string
	Location    *string
	IPAddress   *string
	Description *string
}

func (s *DeviceService) Create(ctx context.Context, input CreateDeviceInput) (*models.Device, error) {
	status := input.Status
	if status == "" {
		status = models.DeviceStatusOffline
	}

	if err := validateDeviceFields(input.Name, input.Type, status); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	device := &models.Device{
		ID:          uuid.New(),
		Name:        strings.TrimSpace(input.Name),
		Type:        input.Type,
		Status:      status,
		Location:    strings.TrimSpace(input.Location),
		IPAddress:   strings.TrimSpace(input.IPAddress),
		Description: strings.TrimSpace(input.Description),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.devices.Create(ctx, device); err != nil {
		return nil, err
	}

	return device, nil
}

func (s *DeviceService) GetByID(ctx context.Context, id uuid.UUID) (*models.Device, error) {
	return s.devices.FindByID(ctx, id)
}

func (s *DeviceService) Update(ctx context.Context, id uuid.UUID, input UpdateDeviceInput) (*models.Device, error) {
	device, err := s.devices.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		device.Name = strings.TrimSpace(*input.Name)
	}
	if input.Type != nil {
		device.Type = *input.Type
	}
	if input.Status != nil {
		device.Status = *input.Status
	}
	if input.Location != nil {
		device.Location = strings.TrimSpace(*input.Location)
	}
	if input.IPAddress != nil {
		device.IPAddress = strings.TrimSpace(*input.IPAddress)
	}
	if input.Description != nil {
		device.Description = strings.TrimSpace(*input.Description)
	}

	if err := validateDeviceFields(device.Name, device.Type, device.Status); err != nil {
		return nil, err
	}

	device.UpdatedAt = time.Now().UTC()

	if err := s.devices.Update(ctx, device); err != nil {
		return nil, err
	}

	return device, nil
}

func (s *DeviceService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.devices.Delete(ctx, id)
}

func (s *DeviceService) List(ctx context.Context, filter repositories.DeviceListFilter) (*repositories.PaginatedResult[models.Device], error) {
	return s.devices.List(ctx, filter)
}

func validateDeviceFields(name, deviceType, status string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("name is required")
	}
	if !models.IsValidDeviceType(deviceType) {
		return ErrInvalidDeviceType
	}
	if !models.IsValidDeviceStatus(status) {
		return ErrInvalidDeviceStatus
	}
	return nil
}
