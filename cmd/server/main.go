package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"Network-control-api/internal/config"
	"Network-control-api/internal/infrastructure/database"
	"Network-control-api/internal/logger"
	"Network-control-api/internal/router"
	"Network-control-api/internal/server"
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

	engine := router.New(cfg, log, db)
	srv := server.New(cfg.Server, log, engine)

	if err := srv.Run(ctx); err != nil {
		log.Error("server stopped with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
