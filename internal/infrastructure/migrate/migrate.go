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

func Run(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, usersTableSQL); err != nil {
		return fmt.Errorf("run users migration: %w", err)
	}
	if _, err := pool.Exec(ctx, devicesTableSQL); err != nil {
		return fmt.Errorf("run devices migration: %w", err)
	}
	return nil
}
