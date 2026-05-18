package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"Network-control-api/internal/models"
)

type IncidentRepository interface {
	FindActiveByDevice(ctx context.Context, deviceID uuid.UUID) (*models.Incident, error)
	CreateWithAlert(ctx context.Context, incident *models.Incident, alertID uuid.UUID) error
	LinkAlert(ctx context.Context, incidentID, alertID uuid.UUID) error
}

type PostgresIncidentRepository struct {
	pool *pgxpool.Pool
}

func NewIncidentRepository(pool *pgxpool.Pool) *PostgresIncidentRepository {
	return &PostgresIncidentRepository{pool: pool}
}

func (r *PostgresIncidentRepository) FindActiveByDevice(ctx context.Context, deviceID uuid.UUID) (*models.Incident, error) {
	query := `
		SELECT id, device_id, device_name, title, description, status, created_at, updated_at, resolved_at
		FROM incidents
		WHERE device_id = $1 AND status IN ('OPEN', 'INVESTIGATING')
		ORDER BY created_at DESC
		LIMIT 1
	`

	var incident models.Incident
	err := r.pool.QueryRow(ctx, query, deviceID).Scan(
		&incident.ID,
		&incident.DeviceID,
		&incident.DeviceName,
		&incident.Title,
		&incident.Description,
		&incident.Status,
		&incident.CreatedAt,
		&incident.UpdatedAt,
		&incident.ResolvedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("query active incident: %w", err)
	}

	return &incident, nil
}

func (r *PostgresIncidentRepository) CreateWithAlert(ctx context.Context, incident *models.Incident, alertID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	insertIncident := `
		INSERT INTO incidents (
			id, device_id, device_name, title, description, status, created_at, updated_at, resolved_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = tx.Exec(ctx, insertIncident,
		incident.ID,
		incident.DeviceID,
		incident.DeviceName,
		incident.Title,
		incident.Description,
		incident.Status,
		incident.CreatedAt,
		incident.UpdatedAt,
		incident.ResolvedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateActiveIncident
		}
		return fmt.Errorf("insert incident: %w", err)
	}

	if err := insertIncidentAlert(ctx, tx, incident.ID, alertID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit incident transaction: %w", err)
	}

	return nil
}

func (r *PostgresIncidentRepository) LinkAlert(ctx context.Context, incidentID, alertID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO incident_alerts (incident_id, alert_id, linked_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (alert_id) DO NOTHING
	`, incidentID, alertID)
	if err != nil {
		return fmt.Errorf("link alert to incident: %w", err)
	}
	return nil
}

func insertIncidentAlert(ctx context.Context, tx pgx.Tx, incidentID, alertID uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO incident_alerts (incident_id, alert_id, linked_at)
		VALUES ($1, $2, NOW())
	`, incidentID, alertID)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("alert already linked to an incident")
		}
		return fmt.Errorf("insert incident alert link: %w", err)
	}
	return nil
}
