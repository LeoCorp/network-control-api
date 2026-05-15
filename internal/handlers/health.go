package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"Network-control-api/internal/infrastructure/database"
)

type HealthHandler struct {
	appName string
	log     *slog.Logger
	db      *database.Postgres
}

func NewHealthHandler(appName string, log *slog.Logger, db *database.Postgres) *HealthHandler {
	return &HealthHandler{
		appName: appName,
		log:     log,
		db:      db,
	}
}

type healthResponse struct {
	Status   string `json:"status"`
	Service  string `json:"service"`
	Database string `json:"database"`
}

func (h *HealthHandler) Check(c *gin.Context) {
	dbStatus := "up"
	if err := h.db.Ping(c.Request.Context()); err != nil {
		h.log.Warn("health check database ping failed", slog.String("error", err.Error()))
		dbStatus = "down"
	}

	status := http.StatusOK
	overall := "ok"
	if dbStatus == "down" {
		status = http.StatusServiceUnavailable
		overall = "degraded"
	}

	c.JSON(status, healthResponse{
		Status:   overall,
		Service:  h.appName,
		Database: dbStatus,
	})
}
