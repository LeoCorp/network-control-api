package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"Network-control-api/internal/config"
	"Network-control-api/internal/handlers"
	"Network-control-api/internal/infrastructure/database"
	"Network-control-api/internal/middleware"
)

func New(cfg *config.Config, log *slog.Logger, db *database.Postgres) *gin.Engine {
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger(log))

	healthHandler := handlers.NewHealthHandler(cfg.App.Name, log, db)
	r.GET("/health", healthHandler.Check)

	return r
}
