package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Network-control-api/internal/monitoring"
)

type LiveStateProvider interface {
	IsRunning() bool
	GetRuntimeState(deviceID uuid.UUID) (monitoring.DeviceRuntimeState, bool)
	ListRuntimeStates() []monitoring.DeviceRuntimeState
}

type MonitoringHandler struct {
	engine LiveStateProvider
}

func NewMonitoringHandler(engine LiveStateProvider) *MonitoringHandler {
	return &MonitoringHandler{engine: engine}
}

type liveStateListResponse struct {
	EngineRunning bool                           `json:"engine_running"`
	Data          []monitoring.DeviceRuntimeState `json:"data"`
}

// ListLive godoc
//
//	@Summary		List live device metrics and runtime status
//	@Description	Returns in-memory telemetry and derived ONLINE/WARNING/DOWN status
//	@Tags			monitoring
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	liveStateListResponse
//	@Failure		401	{object}	ErrorResponse
//	@Router			/api/v1/monitoring/live [get]
func (h *MonitoringHandler) ListLive(c *gin.Context) {
	c.JSON(http.StatusOK, liveStateListResponse{
		EngineRunning: h.engine.IsRunning(),
		Data:          h.engine.ListRuntimeStates(),
	})
}

// GetLiveByID godoc
//
//	@Summary		Get live metrics and runtime status for a device
//	@Tags			monitoring
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Device ID"
//	@Success		200	{object}	monitoring.DeviceRuntimeState
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Router			/api/v1/monitoring/live/{id} [get]
func (h *MonitoringHandler) GetLiveByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		Error(c, http.StatusBadRequest, "invalid device id")
		return
	}

	state, ok := h.engine.GetRuntimeState(id)
	if !ok {
		Error(c, http.StatusNotFound, "live state not found for device")
		return
	}

	c.JSON(http.StatusOK, state)
}
