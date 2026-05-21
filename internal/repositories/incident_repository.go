package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"Network-control-api/internal/models"
)

type IncidentRepository interface {
	FindActiveByDevice(ctx context.Context, deviceID uuid.UUID) (*models.Incident, error)
	CreateWithAlert(ctx context.Context, incident *models.Incident, alertID uuid.UUID) error
	LinkAlert(ctx context.Context, incidentID, alertID uuid.UUID) error

	// Phase 3 additions:
	FindByID(ctx context.Context, id uuid.UUID) (*models.Incident, error)
	List(ctx context.Context, status string, deviceID *uuid.UUID, page, limit int) (*PaginatedResult[models.Incident], error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, resolvedAt *time.Time) error
	Escalate(ctx context.Context, id uuid.UUID, escalatedTitle string) error
	GetLinkedAlerts(ctx context.Context, incidentID uuid.UUID) ([]models.Alert, error)
	CreateLog(ctx context.Context, log *models.IncidentLog) error
	GetLogs(ctx context.Context, incidentID uuid.UUID) ([]models.IncidentLog, error)
	FindAllActive(ctx context.Context) ([]models.Incident, error)
}

type PostgresIncidentRepository struct {
	pool *pgxpool.Pool
}

func NewIncidentRepository(pool *pgxpool.Pool) *PostgresIncidentRepository {
	return &PostgresIncidentRepository{pool: pool}
}

func (r *PostgresIncidentRepository) FindActiveByDevice(ctx context.Context, deviceID uuid.UUID) (*models.Incident, error) {
	query := `
		SELECT id, device_id, device_name, title, description, status, escalated, created_at, updated_at, resolved_at
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
		&incident.Escalated,
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
			id, device_id, device_name, title, description, status, escalated, created_at, updated_at, resolved_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = tx.Exec(ctx, insertIncident,
		incident.ID,
		incident.DeviceID,
		incident.DeviceName,
		incident.Title,
		incident.Description,
		incident.Status,
		incident.Escalated,
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

	// Create initial audit log for creation inside the same transaction
	logID := uuid.New()
	initialLogQuery := `
		INSERT INTO incident_logs (id, incident_id, user_id, action, message, metadata, created_at)
		VALUES ($1, $2, NULL, 'created', 'Incident automatically created from critical alert.', NULL, NOW())
	`
	if _, err := tx.Exec(ctx, initialLogQuery, logID, incident.ID); err != nil {
		return fmt.Errorf("insert initial incident log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit incident transaction: %w", err)
	}

	return nil
}

func (r *PostgresIncidentRepository) LinkAlert(ctx context.Context, incidentID, alertID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO incident_alerts (incident_id, alert_id, linked_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (alert_id) DO NOTHING
	`, incidentID, alertID)
	if err != nil {
		return fmt.Errorf("link alert to incident: %w", err)
	}

	// Create audit log for linked alert inside transaction
	logID := uuid.New()
	msg := fmt.Sprintf("Alert %s linked to incident.", alertID.String())
	metadataMap := map[string]any{"alert_id": alertID.String()}
	metadataBytes, _ := json.Marshal(metadataMap)

	logQuery := `
		INSERT INTO incident_logs (id, incident_id, user_id, action, message, metadata, created_at)
		VALUES ($1, $2, NULL, 'alert_linked', $3, $4, NOW())
	`
	if _, err := tx.Exec(ctx, logQuery, logID, incidentID, msg, metadataBytes); err != nil {
		return fmt.Errorf("insert linked alert incident log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit link alert transaction: %w", err)
	}
	return nil
}

func (r *PostgresIncidentRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Incident, error) {
	query := `
		SELECT id, device_id, device_name, title, description, status, escalated, created_at, updated_at, resolved_at
		FROM incidents
		WHERE id = $1
	`
	var incident models.Incident
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&incident.ID,
		&incident.DeviceID,
		&incident.DeviceName,
		&incident.Title,
		&incident.Description,
		&incident.Status,
		&incident.Escalated,
		&incident.CreatedAt,
		&incident.UpdatedAt,
		&incident.ResolvedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("query incident by id: %w", err)
	}
	return &incident, nil
}

func (r *PostgresIncidentRepository) List(ctx context.Context, status string, deviceID *uuid.UUID, page, limit int) (*PaginatedResult[models.Incident], error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	whereClauses := make([]string, 0)
	args := make([]any, 0)
	argPos := 1

	if status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argPos))
		args = append(args, status)
		argPos++
	}
	if deviceID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("device_id = $%d", argPos))
		args = append(args, *deviceID)
		argPos++
	}

	where := ""
	if len(whereClauses) > 0 {
		where = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	countQuery := `SELECT COUNT(*) FROM incidents` + where
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count incidents: %w", err)
	}

	listQuery := fmt.Sprintf(`
		SELECT id, device_id, device_name, title, description, status, escalated, created_at, updated_at, resolved_at
		FROM incidents%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argPos, argPos+1)

	listArgs := append(args, limit, offset)
	rows, err := r.pool.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, fmt.Errorf("query incidents: %w", err)
	}
	defer rows.Close()

	incidents := make([]models.Incident, 0)
	for rows.Next() {
		var incident models.Incident
		err := rows.Scan(
			&incident.ID,
			&incident.DeviceID,
			&incident.DeviceName,
			&incident.Title,
			&incident.Description,
			&incident.Status,
			&incident.Escalated,
			&incident.CreatedAt,
			&incident.UpdatedAt,
			&incident.ResolvedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan incident: %w", err)
		}
		incidents = append(incidents, incident)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate incidents: %w", err)
	}

	return &PaginatedResult[models.Incident]{
		Items: incidents,
		Meta:  NewPaginationMeta(total, page, limit),
	}, nil
}

func (r *PostgresIncidentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, resolvedAt *time.Time) error {
	query := `
		UPDATE incidents
		SET status = $2, resolved_at = $3, updated_at = NOW()
		WHERE id = $1
	`
	result, err := r.pool.Exec(ctx, query, id, status, resolvedAt)
	if err != nil {
		return fmt.Errorf("update incident status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresIncidentRepository) Escalate(ctx context.Context, id uuid.UUID, escalatedTitle string) error {
	query := `
		UPDATE incidents
		SET escalated = TRUE, title = $2, updated_at = NOW()
		WHERE id = $1 AND escalated = FALSE
	`
	result, err := r.pool.Exec(ctx, query, id, escalatedTitle)
	if err != nil {
		return fmt.Errorf("escalate incident: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresIncidentRepository) GetLinkedAlerts(ctx context.Context, incidentID uuid.UUID) ([]models.Alert, error) {
	query := `
		SELECT a.id, a.device_id, a.device_name, a.severity, a.metric, a.message, a.value, a.threshold, a.created_at
		FROM alerts a
		JOIN incident_alerts ia ON a.id = ia.alert_id
		WHERE ia.incident_id = $1
		ORDER BY a.created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, incidentID)
	if err != nil {
		return nil, fmt.Errorf("query linked alerts: %w", err)
	}
	defer rows.Close()

	alerts := make([]models.Alert, 0)
	for rows.Next() {
		var alert models.Alert
		err := rows.Scan(
			&alert.ID,
			&alert.DeviceID,
			&alert.DeviceName,
			&alert.Severity,
			&alert.Metric,
			&alert.Message,
			&alert.Value,
			&alert.Threshold,
			&alert.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}
		alerts = append(alerts, alert)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate alerts: %w", err)
	}

	return alerts, nil
}

func (r *PostgresIncidentRepository) CreateLog(ctx context.Context, log *models.IncidentLog) error {
	var metadataBytes []byte
	var err error
	if log.Metadata != nil {
		metadataBytes, err = json.Marshal(log.Metadata)
		if err != nil {
			return fmt.Errorf("marshal log metadata: %w", err)
		}
	}

	query := `
		INSERT INTO incident_logs (id, incident_id, user_id, action, message, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = r.pool.Exec(ctx, query,
		log.ID,
		log.IncidentID,
		log.UserID,
		log.Action,
		log.Message,
		metadataBytes,
		log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert incident log: %w", err)
	}
	return nil
}

func (r *PostgresIncidentRepository) GetLogs(ctx context.Context, incidentID uuid.UUID) ([]models.IncidentLog, error) {
	query := `
		SELECT id, incident_id, user_id, action, message, metadata, created_at
		FROM incident_logs
		WHERE incident_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, incidentID)
	if err != nil {
		return nil, fmt.Errorf("query incident logs: %w", err)
	}
	defer rows.Close()

	logs := make([]models.IncidentLog, 0)
	for rows.Next() {
		var log models.IncidentLog
		var metadataBytes []byte
		err := rows.Scan(
			&log.ID,
			&log.IncidentID,
			&log.UserID,
			&log.Action,
			&log.Message,
			&metadataBytes,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan incident log: %w", err)
		}

		if len(metadataBytes) > 0 {
			if err := json.Unmarshal(metadataBytes, &log.Metadata); err != nil {
				return nil, fmt.Errorf("unmarshal log metadata: %w", err)
			}
		}
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate incident logs: %w", err)
	}

	return logs, nil
}

func (r *PostgresIncidentRepository) FindAllActive(ctx context.Context) ([]models.Incident, error) {
	query := `
		SELECT id, device_id, device_name, title, description, status, escalated, created_at, updated_at, resolved_at
		FROM incidents
		WHERE status IN ('OPEN', 'INVESTIGATING')
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query all active incidents: %w", err)
	}
	defer rows.Close()

	incidents := make([]models.Incident, 0)
	for rows.Next() {
		var incident models.Incident
		err := rows.Scan(
			&incident.ID,
			&incident.DeviceID,
			&incident.DeviceName,
			&incident.Title,
			&incident.Description,
			&incident.Status,
			&incident.Escalated,
			&incident.CreatedAt,
			&incident.UpdatedAt,
			&incident.ResolvedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan active incident: %w", err)
		}
		incidents = append(incidents, incident)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active incidents: %w", err)
	}

	return incidents, nil
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
