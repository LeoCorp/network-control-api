package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App        AppConfig
	Server     ServerConfig
	Database   DatabaseConfig
	JWT        JWTConfig
	Monitoring MonitoringConfig
	Log        LogConfig
}

type AppConfig struct {
	Env  string
	Name string
}

type ServerConfig struct {
	Host            string
	Port            string
	ShutdownTimeout time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type JWTConfig struct {
	Secret     string
	Expiration time.Duration
}

type MonitoringConfig struct {
	Interval      time.Duration
	DeviceRefresh time.Duration
	ChannelBuffer int
}

type LogConfig struct {
	Level  string
	Format string
}

func (s ServerConfig) Address() string {
	return s.Host + ":" + s.Port
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	shutdownTimeout, err := parseDurationSeconds(getEnv("SERVER_SHUTDOWN_TIMEOUT", "10"), 10)
	if err != nil {
		return nil, fmt.Errorf("invalid SERVER_SHUTDOWN_TIMEOUT: %w", err)
	}

	cfg := &Config{
		App: AppConfig{
			Env:  getEnv("APP_ENV", "development"),
			Name: getEnv("APP_NAME", "network-control-api"),
		},
		Server: ServerConfig{
			Host:            getEnv("SERVER_HOST", "0.0.0.0"),
			Port:            getEnv("SERVER_PORT", "8080"),
			ShutdownTimeout: shutdownTimeout,
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "network_control"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", ""),
			Expiration: parseJWTExpiration(getEnv("JWT_EXPIRATION_HOURS", "24")),
		},
		Monitoring: MonitoringConfig{
			Interval:      parseMonitoringDuration(getEnv("MONITORING_INTERVAL_SECONDS", "5"), 5),
			DeviceRefresh: parseMonitoringDuration(getEnv("MONITORING_DEVICE_REFRESH_SECONDS", "30"), 30),
			ChannelBuffer: parseMonitoringBuffer(getEnv("MONITORING_CHANNEL_BUFFER", "64"), 64),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "text"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("SERVER_PORT is required")
	}
	if c.Database.Host == "" || c.Database.User == "" || c.Database.Name == "" {
		return fmt.Errorf("database configuration is incomplete")
	}
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if c.JWT.Expiration <= 0 {
		return fmt.Errorf("JWT_EXPIRATION_HOURS must be positive")
	}
	return nil
}

func parseMonitoringDuration(seconds string, fallback int) time.Duration {
	parsed, err := strconv.Atoi(seconds)
	if err != nil || parsed <= 0 {
		return time.Duration(fallback) * time.Second
	}
	return time.Duration(parsed) * time.Second
}

func parseMonitoringBuffer(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseJWTExpiration(hours string) time.Duration {
	parsed, err := strconv.Atoi(hours)
	if err != nil || parsed <= 0 {
		return 24 * time.Hour
	}
	return time.Duration(parsed) * time.Hour
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func parseDurationSeconds(value string, fallback int) (time.Duration, error) {
	seconds, err := strconv.Atoi(value)
	if err != nil {
		return time.Duration(fallback) * time.Second, err
	}
	if seconds <= 0 {
		return time.Duration(fallback) * time.Second, fmt.Errorf("must be positive")
	}
	return time.Duration(seconds) * time.Second, nil
}
