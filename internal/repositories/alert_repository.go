package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"Network-control-api/internal/models"
)

type AlertRepository interface {
	Create(ctx context.Context, alert *models.Alert) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Alert, error)
	List(ctx context.Context, filter AlertListFilter) (*PaginatedResult[models.Alert], error)
}

type AlertListFilter struct {
	Page     int
	Limit    int
	DeviceID uuid.UUID
	Severity string
}

type PostgresAlertRepository struct {
	pool *pgxpool.Pool
}

func NewAlertRepository(pool *pgxpool.Pool) *PostgresAlertRepository {
	return &PostgresAlertRepository{pool: pool}
}

func (r *PostgresAlertRepository) Create(ctx context.Context, alert *models.Alert) error {
	query := `
		INSERT INTO alerts (
			id, device_id, device_name, severity, metric, message, value, threshold, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.pool.Exec(ctx, query,
		alert.ID,
		alert.DeviceID,
		alert.DeviceName,
		alert.Severity,
		alert.Metric,
		alert.Message,
		alert.Value,
		alert.Threshold,
		alert.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert alert: %w", err)
	}

	return nil
}

func (r *PostgresAlertRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Alert, error) {
	query := `
		SELECT id, device_id, device_name, severity, metric, message, value, threshold, created_at
		FROM alerts
		WHERE id = $1
	`

	var alert models.Alert
	err := r.pool.QueryRow(ctx, query, id).Scan(
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("query alert: %w", err)
	}

	return &alert, nil
}

func (r *PostgresAlertRepository) List(ctx context.Context, filter AlertListFilter) (*PaginatedResult[models.Alert], error) {
	page := filter.Page
	if page <= 0 {
		page = 1
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	where, args := buildAlertListWhere(filter)
	args = append(args, limit, offset)

	countQuery := `SELECT COUNT(*) FROM alerts` + where
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args[:len(args)-2]...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count alerts: %w", err)
	}

	listQuery := fmt.Sprintf(`
		SELECT id, device_id, device_name, severity, metric, message, value, threshold, created_at
		FROM alerts%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("list alerts: %w", err)
	}
	defer rows.Close()

	alerts := make([]models.Alert, 0)
	for rows.Next() {
		var alert models.Alert
		if err := rows.Scan(
			&alert.ID,
			&alert.DeviceID,
			&alert.DeviceName,
			&alert.Severity,
			&alert.Metric,
			&alert.Message,
			&alert.Value,
			&alert.Threshold,
			&alert.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}
		alerts = append(alerts, alert)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate alerts: %w", err)
	}

	return &PaginatedResult[models.Alert]{
		Items: alerts,
		Meta:  NewPaginationMeta(total, page, limit),
	}, nil
}

func buildAlertListWhere(filter AlertListFilter) (string, []any) {
	clauses := make([]string, 0)
	args := make([]any, 0)
	argPos := 1

	if filter.DeviceID != uuid.Nil {
		clauses = append(clauses, fmt.Sprintf("device_id = $%d", argPos))
		args = append(args, filter.DeviceID)
		argPos++
	}

	if filter.Severity != "" {
		clauses = append(clauses, fmt.Sprintf("severity = $%d", argPos))
		args = append(args, filter.Severity)
	}

	if len(clauses) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(clauses, " AND "), args
}
