//	@title			Network Control API
//	@version		1.0
//	@description	Telecom NOC Monitoring Backend API
//	@host			localhost:8080
//	@BasePath		/
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Type "Bearer" followed by a space and JWT token.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"Network-control-api/internal/auth"
	"Network-control-api/internal/config"
	"Network-control-api/internal/infrastructure/database"
	"Network-control-api/internal/infrastructure/migrate"
	"Network-control-api/internal/logger"
	"Network-control-api/internal/repositories"
	"Network-control-api/internal/router"
	"Network-control-api/internal/server"
	"Network-control-api/internal/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	log := logger.New(cfg.Log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.NewPostgres(ctx, cfg.Database)
	if err != nil {
		log.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	if err := migrate.Run(ctx, db.Pool); err != nil {
		log.Error("failed to run migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("database migrations applied")

	userRepo := repositories.NewUserRepository(db.Pool)
	deviceRepo := repositories.NewDeviceRepository(db.Pool)
	jwtService := auth.NewJWTService(cfg.JWT)
	authService := services.NewAuthService(userRepo, jwtService)
	deviceService := services.NewDeviceService(deviceRepo)

	engine := router.New(router.Dependencies{
		Config:        cfg,
		Log:           log,
		DB:            db,
		AuthService:   authService,
		DeviceService: deviceService,
		JWTService:    jwtService,
	})

	srv := server.New(cfg.Server, log, engine)

	if err := srv.Run(ctx); err != nil {
		log.Error("server stopped with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
