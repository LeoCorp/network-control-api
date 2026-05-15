package router

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"Network-control-api/internal/config"

	docs "Network-control-api/docs"
)

func registerSwaggerRoutes(r *gin.Engine, cfg config.ServerConfig) {
	host := cfg.Host
	if host == "0.0.0.0" || host == "" {
		host = "localhost"
	}

	docs.SwaggerInfo.Host = host + ":" + cfg.Port
	docs.SwaggerInfo.BasePath = "/"

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
