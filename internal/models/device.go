package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	DeviceTypeRouter   = "router"
	DeviceTypeTower    = "tower"
	DeviceTypeSwitch   = "switch"
	DeviceTypeCoreNode = "core_node"
	DeviceTypeLink     = "link"
	DeviceTypeService  = "service"

	DeviceStatusOnline      = "online"
	DeviceStatusOffline     = "offline"
	DeviceStatusDegraded    = "degraded"
	DeviceStatusMaintenance = "maintenance"
)

var ValidDeviceTypes = map[string]bool{
	DeviceTypeRouter:   true,
	DeviceTypeTower:    true,
	DeviceTypeSwitch:   true,
	DeviceTypeCoreNode: true,
	DeviceTypeLink:     true,
	DeviceTypeService:  true,
}

var ValidDeviceStatuses = map[string]bool{
	DeviceStatusOnline:      true,
	DeviceStatusOffline:     true,
	DeviceStatusDegraded:    true,
	DeviceStatusMaintenance: true,
}

type Device struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	Location    string    `json:"location,omitempty"`
	IPAddress   string    `json:"ip_address,omitempty"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func IsValidDeviceType(deviceType string) bool {
	return ValidDeviceTypes[deviceType]
}

func IsValidDeviceStatus(status string) bool {
	return ValidDeviceStatuses[status]
}
