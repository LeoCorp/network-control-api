package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Network-control-api/internal/models"
	"Network-control-api/internal/repositories"
	"Network-control-api/internal/services"
)

type DeviceHandler struct {
	devices *services.DeviceService
}

func NewDeviceHandler(devices *services.DeviceService) *DeviceHandler {
	return &DeviceHandler{devices: devices}
}

// Create godoc
//
//	@Summary		Create device
//	@Description	Create a new network device (admin, operator)
//	@Tags			devices
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateDeviceRequest	true	"Device payload"
//	@Success		201		{object}	DeviceResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/api/v1/devices [post]
func (h *DeviceHandler) Create(c *gin.Context) {
	var req CreateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	device, err := h.devices.Create(c.Request.Context(), services.CreateDeviceInput{
		Name:        req.Name,
		Type:        req.Type,
		Status:      req.Status,
		Location:    req.Location,
		IPAddress:   req.IPAddress,
		Description: req.Description,
	})
	if err != nil {
		handleDeviceError(c, err, "failed to create device")
		return
	}

	c.JSON(http.StatusCreated, toDeviceResponse(device))
}

// GetByID godoc
//
//	@Summary		Get device by ID
//	@Description	Retrieve a single device by UUID
//	@Tags			devices
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Device ID"
//	@Success		200	{object}	DeviceResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/api/v1/devices/{id} [get]
func (h *DeviceHandler) GetByID(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		Error(c, http.StatusBadRequest, "invalid device id")
		return
	}

	device, err := h.devices.GetByID(c.Request.Context(), id)
	if err != nil {
		handleDeviceError(c, err, "failed to get device")
		return
	}

	c.JSON(http.StatusOK, toDeviceResponse(device))
}

// Update godoc
//
//	@Summary		Update device
//	@Description	Partially update a device (admin, operator)
//	@Tags			devices
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string				true	"Device ID"
//	@Param			request	body		UpdateDeviceRequest	true	"Device update payload"
//	@Success		200		{object}	DeviceResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/api/v1/devices/{id} [patch]
func (h *DeviceHandler) Update(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		Error(c, http.StatusBadRequest, "invalid device id")
		return
	}

	var req UpdateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.Name == nil && req.Type == nil && req.Status == nil &&
		req.Location == nil && req.IPAddress == nil && req.Description == nil {
		Error(c, http.StatusBadRequest, "at least one field is required")
		return
	}

	device, err := h.devices.Update(c.Request.Context(), id, services.UpdateDeviceInput{
		Name:        req.Name,
		Type:        req.Type,
		Status:      req.Status,
		Location:    req.Location,
		IPAddress:   req.IPAddress,
		Description: req.Description,
	})
	if err != nil {
		handleDeviceError(c, err, "failed to update device")
		return
	}

	c.JSON(http.StatusOK, toDeviceResponse(device))
}

// Delete godoc
//
//	@Summary		Delete device
//	@Description	Delete a device by UUID (admin only)
//	@Tags			devices
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Device ID"
//	@Success		204	"No Content"
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		403	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/api/v1/devices/{id} [delete]
func (h *DeviceHandler) Delete(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		Error(c, http.StatusBadRequest, "invalid device id")
		return
	}

	if err := h.devices.Delete(c.Request.Context(), id); err != nil {
		handleDeviceError(c, err, "failed to delete device")
		return
	}

	c.Status(http.StatusNoContent)
}

// List godoc
//
//	@Summary		List devices
//	@Description	List devices with pagination, search, and filters
//	@Tags			devices
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int		false	"Page number"		default(1)
//	@Param			limit		query		int		false	"Items per page"	default(10)
//	@Param			search		query		string	false	"Search name, location, IP, description"
//	@Param			type		query		string	false	"Device type"		Enums(router, tower, switch, core_node, link, service)
//	@Param			status		query		string	false	"Device status"		Enums(online, offline, degraded, maintenance)
//	@Param			sort_by		query		string	false	"Sort field"		Enums(name, type, status, created_at, updated_at)
//	@Param			sort_order	query		string	false	"Sort order"		Enums(asc, desc)
//	@Success		200			{object}	DeviceListResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/api/v1/devices [get]
func (h *DeviceHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	deviceType := c.Query("type")
	if deviceType != "" && !models.IsValidDeviceType(deviceType) {
		Error(c, http.StatusBadRequest, "invalid type filter")
		return
	}

	status := c.Query("status")
	if status != "" && !models.IsValidDeviceStatus(status) {
		Error(c, http.StatusBadRequest, "invalid status filter")
		return
	}

	result, err := h.devices.List(c.Request.Context(), repositories.DeviceListFilter{
		Page:      page,
		Limit:     limit,
		Search:    c.Query("search"),
		Type:      deviceType,
		Status:    status,
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
	})
	if err != nil {
		Error(c, http.StatusInternalServerError, "failed to list devices")
		return
	}

	items := make([]DeviceResponse, 0, len(result.Items))
	for i := range result.Items {
		items = append(items, toDeviceResponse(&result.Items[i]))
	}

	c.JSON(http.StatusOK, DeviceListResponse{
		Data: items,
		Meta: PaginationMetaResponse{
			Total:       result.Meta.Total,
			CurrentPage: result.Meta.CurrentPage,
			TotalPages:  result.Meta.TotalPages,
			Limit:       result.Meta.Limit,
		},
	})
}

func handleDeviceError(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, repositories.ErrNotFound):
		Error(c, http.StatusNotFound, "device not found")
	case errors.Is(err, repositories.ErrDuplicateName):
		Error(c, http.StatusConflict, "device name already exists")
	case errors.Is(err, services.ErrInvalidDeviceType):
		Error(c, http.StatusBadRequest, "invalid device type")
	case errors.Is(err, services.ErrInvalidDeviceStatus):
		Error(c, http.StatusBadRequest, "invalid device status")
	default:
		if err.Error() == "name is required" {
			Error(c, http.StatusBadRequest, err.Error())
			return
		}
		Error(c, http.StatusInternalServerError, fallback)
	}
}

func toDeviceResponse(device *models.Device) DeviceResponse {
	return DeviceResponse{
		ID:          device.ID.String(),
		Name:        device.Name,
		Type:        device.Type,
		Status:      device.Status,
		Location:    device.Location,
		IPAddress:   device.IPAddress,
		Description: device.Description,
		CreatedAt:   device.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   device.UpdatedAt.Format(time.RFC3339),
	}
}

func parseUUIDParam(c *gin.Context, name string) (uuid.UUID, error) {
	return uuid.Parse(c.Param(name))
}
