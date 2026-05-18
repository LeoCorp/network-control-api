package monitoring

import (
	"context"

	"Network-control-api/internal/models"
	"Network-control-api/internal/repositories"
)

// DeviceProvider supplies devices to the monitoring engine.
type DeviceProvider interface {
	ListDevices(ctx context.Context) ([]models.Device, error)
}

type RepositoryDeviceProvider struct {
	devices repositories.DeviceRepository
}

func NewRepositoryDeviceProvider(devices repositories.DeviceRepository) *RepositoryDeviceProvider {
	return &RepositoryDeviceProvider{devices: devices}
}

func (p *RepositoryDeviceProvider) ListDevices(ctx context.Context) ([]models.Device, error) {
	result, err := p.devices.List(ctx, repositories.DeviceListFilter{
		Page:  1,
		Limit: 1000,
	})
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}
