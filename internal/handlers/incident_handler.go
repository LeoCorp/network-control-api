package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Network-control-api/internal/middleware"
	"Network-control-api/internal/models"
	"Network-control-api/internal/repositories"
	"Network-control-api/internal/services"
)

type IncidentHandler struct {
	service *services.IncidentService
}

func NewIncidentHandler(service *services.IncidentService) *IncidentHandler {
	return &IncidentHandler{service: service}
}

type UpdateIncidentStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=OPEN INVESTIGATING RESOLVED" example:"INVESTIGATING"`
}

type IncidentLogResponse struct {
	ID         string         `json:"id"`
	IncidentID string         `json:"incident_id"`
	UserID     *string        `json:"user_id,omitempty"`
	Action     string         `json:"action"`
	Message    string         `json:"message"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  string         `json:"created_at"`
}

type IncidentResponse struct {
	ID          string     `json:"id"`
	DeviceID    string     `json:"device_id"`
	DeviceName  string     `json:"device_name"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status"`
	Escalated   bool       `json:"escalated"`
	CreatedAt   string     `json:"created_at"`
	UpdatedAt   string     `json:"updated_at"`
	ResolvedAt  *string    `json:"resolved_at,omitempty"`
}

type IncidentDetailsResponse struct {
	Incident IncidentResponse      `json:"incident"`
	Alerts   []DeviceAlertResponse `json:"alerts"`
	Logs     []IncidentLogResponse `json:"logs"`
}

type DeviceAlertResponse struct {
	ID         string  `json:"id"`
	DeviceID   string  `json:"device_id"`
	DeviceName string  `json:"device_name"`
	Severity   string  `json:"severity"`
	Metric     string  `json:"metric"`
	Message    string  `json:"message"`
	Value      float64 `json:"value"`
	Threshold  float64 `json:"threshold"`
	CreatedAt  string  `json:"created_at"`
}

type IncidentListResponse struct {
	Data []IncidentResponse     `json:"data"`
	Meta PaginationMetaResponse `json:"meta"`
}

// List godoc
//
//	@Summary		List incidents
//	@Description	Retrieve a paginated list of incidents, optionally filtered by status or device_id.
//	@Tags			incidents
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int		false	"Page number (default 1)"
//	@Param			limit		query		int		false	"Page limit (default 10, max 100)"
//	@Param			status		query		string	false	"Filter by incident status (OPEN, INVESTIGATING, RESOLVED)"
//	@Param			device_id	query		string	false	"Filter by device UUID"
//	@Success		200			{object}	IncidentListResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/api/v1/incidents [get]
func (h *IncidentHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	status := c.Query("status")
	deviceIDStr := c.Query("device_id")

	var deviceID *uuid.UUID
	if deviceIDStr != "" {
		id, err := uuid.Parse(deviceIDStr)
		if err != nil {
			Error(c, http.StatusBadRequest, "invalid device_id parameter")
			return
		}
		deviceID = &id
	}

	result, err := h.service.ListIncidents(c.Request.Context(), status, deviceID, page, limit)
	if err != nil {
		Error(c, http.StatusInternalServerError, "failed to list incidents")
		return
	}

	data := make([]IncidentResponse, 0, len(result.Items))
	for i := range result.Items {
		data = append(data, toIncidentResponse(&result.Items[i]))
	}

	c.JSON(http.StatusOK, IncidentListResponse{
		Data: data,
		Meta: PaginationMetaResponse{
			Total:       result.Meta.Total,
			CurrentPage: result.Meta.CurrentPage,
			TotalPages:  result.Meta.TotalPages,
			Limit:       result.Meta.Limit,
		},
	})
}

// GetByID godoc
//
//	@Summary		Get incident details
//	@Description	Retrieve detailed information for a specific incident, including linked alerts and audit logs.
//	@Tags			incidents
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Incident UUID"
//	@Success		200	{object}	IncidentDetailsResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/api/v1/incidents/{id} [get]
func (h *IncidentHandler) GetByID(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		Error(c, http.StatusBadRequest, "invalid incident id")
		return
	}

	details, err := h.service.GetIncidentDetails(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			Error(c, http.StatusNotFound, "incident not found")
			return
		}
		Error(c, http.StatusInternalServerError, "failed to get incident details")
		return
	}

	alertsRes := make([]DeviceAlertResponse, 0, len(details.Alerts))
	for _, alert := range details.Alerts {
		alertsRes = append(alertsRes, DeviceAlertResponse{
			ID:         alert.ID.String(),
			DeviceID:   alert.DeviceID.String(),
			DeviceName: alert.DeviceName,
			Severity:   alert.Severity,
			Metric:     alert.Metric,
			Message:    alert.Message,
			Value:      alert.Value,
			Threshold:  alert.Threshold,
			CreatedAt:  alert.CreatedAt.Format(time.RFC3339),
		})
	}

	logsRes := make([]IncidentLogResponse, 0, len(details.Logs))
	for _, log := range details.Logs {
		var userIDStr *string
		if log.UserID != nil {
			s := log.UserID.String()
			userIDStr = &s
		}
		logsRes = append(logsRes, IncidentLogResponse{
			ID:         log.ID.String(),
			IncidentID: log.IncidentID.String(),
			UserID:     userIDStr,
			Action:     log.Action,
			Message:    log.Message,
			Metadata:   log.Metadata,
			CreatedAt:  log.CreatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, IncidentDetailsResponse{
		Incident: toIncidentResponse(details.Incident),
		Alerts:   alertsRes,
		Logs:     logsRes,
	})
}

// UpdateStatus godoc
//
//	@Summary		Update incident status
//	@Description	Update the status of an incident (requires admin or operator role).
//	@Tags			incidents
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string						true	"Incident UUID"
//	@Param			request	body		UpdateIncidentStatusRequest	true	"New status payload"
//	@Success		200		{object}	IncidentResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		403		{object}	ErrorResponse
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/api/v1/incidents/{id}/status [patch]
func (h *IncidentHandler) UpdateStatus(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		Error(c, http.StatusBadRequest, "invalid incident id")
		return
	}

	var req UpdateIncidentStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, _, _, ok := middleware.GetAuthUser(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "user authentication not found")
		return
	}

	incident, err := h.service.UpdateIncidentStatus(c.Request.Context(), id, req.Status, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			Error(c, http.StatusNotFound, "incident not found")
			return
		}
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, toIncidentResponse(incident))
}

// GetLogs godoc
//
//	@Summary		Get incident audit logs
//	@Description	Retrieve the list of historical audit logs for a specific incident.
//	@Tags			incidents
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Incident UUID"
//	@Success		200	{array}		IncidentLogResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/api/v1/incidents/{id}/logs [get]
func (h *IncidentHandler) GetLogs(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		Error(c, http.StatusBadRequest, "invalid incident id")
		return
	}

	details, err := h.service.GetIncidentDetails(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			Error(c, http.StatusNotFound, "incident not found")
			return
		}
		Error(c, http.StatusInternalServerError, "failed to get logs")
		return
	}

	logsRes := make([]IncidentLogResponse, 0, len(details.Logs))
	for _, log := range details.Logs {
		var userIDStr *string
		if log.UserID != nil {
			s := log.UserID.String()
			userIDStr = &s
		}
		logsRes = append(logsRes, IncidentLogResponse{
			ID:         log.ID.String(),
			IncidentID: log.IncidentID.String(),
			UserID:     userIDStr,
			Action:     log.Action,
			Message:    log.Message,
			Metadata:   log.Metadata,
			CreatedAt:  log.CreatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, logsRes)
}

func toIncidentResponse(inc *models.Incident) IncidentResponse {
	var resolvedAtStr *string
	if inc.ResolvedAt != nil {
		s := inc.ResolvedAt.Format(time.RFC3339)
		resolvedAtStr = &s
	}
	return IncidentResponse{
		ID:          inc.ID.String(),
		DeviceID:    inc.DeviceID.String(),
		DeviceName:  inc.DeviceName,
		Title:       inc.Title,
		Description: inc.Description,
		Status:      inc.Status,
		Escalated:   inc.Escalated,
		CreatedAt:   inc.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   inc.UpdatedAt.Format(time.RFC3339),
		ResolvedAt:  resolvedAtStr,
	}
}
