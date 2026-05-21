package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"Network-control-api/internal/auth"
	"Network-control-api/internal/config"
	"Network-control-api/internal/handlers"
	"Network-control-api/internal/infrastructure/database"
	"Network-control-api/internal/middleware"
	"Network-control-api/internal/models"
	"Network-control-api/internal/monitoring"
	"Network-control-api/internal/services"
	"Network-control-api/internal/websocket"
)

type Dependencies struct {
	Config          *config.Config
	Log             *slog.Logger
	DB              *database.Postgres
	AuthService     *services.AuthService
	DeviceService   *services.DeviceService
	IncidentService *services.IncidentService
	MonitorEngine   *monitoring.Engine
	WSHub           *websocket.Hub
	JWTService      *auth.JWTService
}

func New(deps Dependencies) *gin.Engine {
	if deps.Config.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger(deps.Log))

	registerSwaggerRoutes(r, deps.Config.Server)

	healthHandler := handlers.NewHealthHandler(deps.Config.App.Name, deps.Log, deps.DB)
	r.GET("/health", healthHandler.Check)

	authHandler := handlers.NewAuthHandler(deps.AuthService)
	protectedHandler := handlers.NewProtectedHandler()
	deviceHandler := handlers.NewDeviceHandler(deps.DeviceService)
	incidentHandler := handlers.NewIncidentHandler(deps.IncidentService)
	monitoringHandler := handlers.NewMonitoringHandler(deps.MonitorEngine)
	wsHandler := websocket.NewHandler(deps.WSHub, deps.Log)

	v1 := r.Group("/api/v1")
	{
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/register", authHandler.Register)
			authGroup.POST("/login", authHandler.Login)
		}

		protected := v1.Group("/")
		protected.Use(middleware.JWTAuth(deps.JWTService))
		{
			protected.GET("/protected/test", protectedHandler.Test)

			devices := protected.Group("/devices")
			{
				devices.GET("", deviceHandler.List)
				devices.GET("/:id", deviceHandler.GetByID)
				devices.POST("", middleware.RequireRoles(models.RoleAdmin, models.RoleOperator), deviceHandler.Create)
				devices.PATCH("/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleOperator), deviceHandler.Update)
				devices.DELETE("/:id", middleware.RequireRoles(models.RoleAdmin), deviceHandler.Delete)
			}

			incidents := protected.Group("/incidents")
			{
				incidents.GET("", incidentHandler.List)
				incidents.GET("/:id", incidentHandler.GetByID)
				incidents.PATCH("/:id/status", middleware.RequireRoles(models.RoleAdmin, models.RoleOperator), incidentHandler.UpdateStatus)
				incidents.GET("/:id/logs", incidentHandler.GetLogs)
			}

			monitoringGroup := protected.Group("/monitoring")
			{
				monitoringGroup.GET("/live", monitoringHandler.ListLive)
				monitoringGroup.GET("/live/:id", monitoringHandler.GetLiveByID)
				monitoringGroup.GET("/ws", wsHandler.Serve)
			}
		}
	}

	return r
}
