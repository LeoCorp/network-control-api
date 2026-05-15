package handlers

// ErrorResponse represents a standard API error.
type ErrorResponse struct {
	Error string `json:"error" example:"invalid credentials"`
}

// RegisterRequest is the payload for user registration.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email" example:"operator@noc.local"`
	Password string `json:"password" binding:"required,min=8" example:"password123"`
	Role     string `json:"role" binding:"omitempty,oneof=admin operator viewer" example:"operator"`
}

// LoginRequest is the payload for user login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"operator@noc.local"`
	Password string `json:"password" binding:"required" example:"password123"`
}

// AuthResponse is returned after register or login.
type AuthResponse struct {
	Token string       `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	User  UserResponse `json:"user"`
}

// UserResponse represents a user without sensitive fields.
type UserResponse struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email     string `json:"email" example:"operator@noc.local"`
	Role      string `json:"role" example:"operator"`
	CreatedAt string `json:"created_at" example:"2026-05-15T12:00:00Z"`
	UpdatedAt string `json:"updated_at" example:"2026-05-15T12:00:00Z"`
}

// CreateDeviceRequest is the payload for creating a device.
type CreateDeviceRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=255" example:"Core Router 01"`
	Type        string `json:"type" binding:"required,oneof=router tower switch core_node link service" example:"router"`
	Status      string `json:"status" binding:"omitempty,oneof=online offline degraded maintenance" example:"online"`
	Location    string `json:"location" binding:"omitempty,max=255" example:"DC-East"`
	IPAddress   string `json:"ip_address" binding:"omitempty,max=45" example:"10.0.0.1"`
	Description string `json:"description" binding:"omitempty,max=2000" example:"Primary edge router"`
}

// UpdateDeviceRequest is the payload for partial device updates.
type UpdateDeviceRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=2,max=255" example:"Core Router 01"`
	Type        *string `json:"type" binding:"omitempty,oneof=router tower switch core_node link service" example:"router"`
	Status      *string `json:"status" binding:"omitempty,oneof=online offline degraded maintenance" example:"maintenance"`
	Location    *string `json:"location" binding:"omitempty,max=255" example:"DC-West"`
	IPAddress   *string `json:"ip_address" binding:"omitempty,max=45" example:"10.0.0.2"`
	Description *string `json:"description" binding:"omitempty,max=2000" example:"Under maintenance"`
}

// DeviceResponse represents a network device.
type DeviceResponse struct {
	ID          string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string `json:"name" example:"Core Router 01"`
	Type        string `json:"type" example:"router"`
	Status      string `json:"status" example:"online"`
	Location    string `json:"location,omitempty" example:"DC-East"`
	IPAddress   string `json:"ip_address,omitempty" example:"10.0.0.1"`
	Description string `json:"description,omitempty" example:"Primary edge router"`
	CreatedAt   string `json:"created_at" example:"2026-05-15T12:00:00Z"`
	UpdatedAt   string `json:"updated_at" example:"2026-05-15T12:00:00Z"`
}

// PaginationMetaResponse contains list pagination metadata.
type PaginationMetaResponse struct {
	Total       int64 `json:"total" example:"42"`
	CurrentPage int   `json:"current_page" example:"1"`
	TotalPages  int   `json:"total_pages" example:"5"`
	Limit       int   `json:"limit" example:"10"`
}

// DeviceListResponse is the paginated device list response.
type DeviceListResponse struct {
	Data []DeviceResponse       `json:"data"`
	Meta PaginationMetaResponse `json:"meta"`
}
