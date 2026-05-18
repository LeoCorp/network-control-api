package migrate

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const usersTableSQL = `
CREATE TABLE IF NOT EXISTS users (
	id UUID PRIMARY KEY,
	email VARCHAR(255) NOT NULL UNIQUE,
	password_hash VARCHAR(255) NOT NULL,
	role VARCHAR(50) NOT NULL DEFAULT 'viewer',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	CONSTRAINT users_role_check CHECK (role IN ('admin', 'operator', 'viewer'))
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);
`

const devicesTableSQL = `
CREATE TABLE IF NOT EXISTS devices (
	id UUID PRIMARY KEY,
	name VARCHAR(255) NOT NULL UNIQUE,
	type VARCHAR(50) NOT NULL,
	status VARCHAR(50) NOT NULL DEFAULT 'offline',
	location VARCHAR(255) NOT NULL DEFAULT '',
	ip_address VARCHAR(45) NOT NULL DEFAULT '',
	description TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	CONSTRAINT devices_type_check CHECK (type IN ('router', 'tower', 'switch', 'core_node', 'link', 'service')),
	CONSTRAINT devices_status_check CHECK (status IN ('online', 'offline', 'degraded', 'maintenance'))
);

CREATE INDEX IF NOT EXISTS idx_devices_type ON devices (type);
CREATE INDEX IF NOT EXISTS idx_devices_status ON devices (status);
CREATE INDEX IF NOT EXISTS idx_devices_name ON devices (name);
`

const alertsTableSQL = `
CREATE TABLE IF NOT EXISTS alerts (
	id UUID PRIMARY KEY,
	device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
	device_name VARCHAR(255) NOT NULL,
	severity VARCHAR(20) NOT NULL,
	metric VARCHAR(50) NOT NULL,
	message TEXT NOT NULL,
	value DOUBLE PRECISION NOT NULL,
	threshold DOUBLE PRECISION NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	CONSTRAINT alerts_severity_check CHECK (severity IN ('INFO', 'WARNING', 'CRITICAL'))
);

CREATE INDEX IF NOT EXISTS idx_alerts_device_id ON alerts (device_id);
CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON alerts (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_severity ON alerts (severity);
`

const incidentsTableSQL = `
CREATE TABLE IF NOT EXISTS incidents (
	id UUID PRIMARY KEY,
	device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
	device_name VARCHAR(255) NOT NULL,
	title VARCHAR(255) NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	status VARCHAR(20) NOT NULL DEFAULT 'OPEN',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	resolved_at TIMESTAMPTZ,
	CONSTRAINT incidents_status_check CHECK (status IN ('OPEN', 'INVESTIGATING', 'RESOLVED'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_incidents_one_active_per_device
	ON incidents (device_id)
	WHERE status IN ('OPEN', 'INVESTIGATING');

CREATE INDEX IF NOT EXISTS idx_incidents_device_id ON incidents (device_id);
CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents (status);
CREATE INDEX IF NOT EXISTS idx_incidents_created_at ON incidents (created_at DESC);

CREATE TABLE IF NOT EXISTS incident_alerts (
	incident_id UUID NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
	alert_id UUID NOT NULL UNIQUE REFERENCES alerts(id) ON DELETE CASCADE,
	linked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	PRIMARY KEY (incident_id, alert_id)
);

CREATE INDEX IF NOT EXISTS idx_incident_alerts_incident_id ON incident_alerts (incident_id);
CREATE INDEX IF NOT EXISTS idx_incident_alerts_alert_id ON incident_alerts (alert_id);
`

func Run(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, usersTableSQL); err != nil {
		return fmt.Errorf("run users migration: %w", err)
	}
	if _, err := pool.Exec(ctx, devicesTableSQL); err != nil {
		return fmt.Errorf("run devices migration: %w", err)
	}
	if _, err := pool.Exec(ctx, alertsTableSQL); err != nil {
		return fmt.Errorf("run alerts migration: %w", err)
	}
	if _, err := pool.Exec(ctx, incidentsTableSQL); err != nil {
		return fmt.Errorf("run incidents migration: %w", err)
	}
	return nil
}
