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

type DeviceListFilter struct {
	Page      int
	Limit     int
	Search    string
	Type      string
	Status    string
	SortBy    string
	SortOrder string
}

type DeviceRepository interface {
	Create(ctx context.Context, device *models.Device) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Device, error)
	Update(ctx context.Context, device *models.Device) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter DeviceListFilter) (*PaginatedResult[models.Device], error)
}

type PostgresDeviceRepository struct {
	pool *pgxpool.Pool
}

func NewDeviceRepository(pool *pgxpool.Pool) *PostgresDeviceRepository {
	return &PostgresDeviceRepository{pool: pool}
}

func (r *PostgresDeviceRepository) Create(ctx context.Context, device *models.Device) error {
	query := `
		INSERT INTO devices (id, name, type, status, location, ip_address, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.pool.Exec(ctx, query,
		device.ID,
		device.Name,
		device.Type,
		device.Status,
		device.Location,
		device.IPAddress,
		device.Description,
		device.CreatedAt,
		device.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateName
		}
		return fmt.Errorf("insert device: %w", err)
	}

	return nil
}

func (r *PostgresDeviceRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Device, error) {
	query := `
		SELECT id, name, type, status, location, ip_address, description, created_at, updated_at
		FROM devices
		WHERE id = $1
	`

	return r.scanDevice(ctx, query, id)
}

func (r *PostgresDeviceRepository) Update(ctx context.Context, device *models.Device) error {
	query := `
		UPDATE devices
		SET name = $2, type = $3, status = $4, location = $5, ip_address = $6,
		    description = $7, updated_at = $8
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query,
		device.ID,
		device.Name,
		device.Type,
		device.Status,
		device.Location,
		device.IPAddress,
		device.Description,
		device.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateName
		}
		return fmt.Errorf("update device: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *PostgresDeviceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM devices WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *PostgresDeviceRepository) List(ctx context.Context, filter DeviceListFilter) (*PaginatedResult[models.Device], error) {
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
	sortColumn := resolveDeviceSortColumn(filter.SortBy)
	sortOrder := resolveSortOrder(filter.SortOrder)

	where, args := buildDeviceListWhere(filter)
	args = append(args, limit, offset)

	countQuery := `SELECT COUNT(*) FROM devices` + where
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args[:len(args)-2]...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count devices: %w", err)
	}

	listQuery := fmt.Sprintf(`
		SELECT id, name, type, status, location, ip_address, description, created_at, updated_at
		FROM devices%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, where, sortColumn, sortOrder, len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	defer rows.Close()

	devices := make([]models.Device, 0)
	for rows.Next() {
		var device models.Device
		if err := rows.Scan(
			&device.ID,
			&device.Name,
			&device.Type,
			&device.Status,
			&device.Location,
			&device.IPAddress,
			&device.Description,
			&device.CreatedAt,
			&device.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}
		devices = append(devices, device)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate devices: %w", err)
	}

	return &PaginatedResult[models.Device]{
		Items: devices,
		Meta:  NewPaginationMeta(total, page, limit),
	}, nil
}

func buildDeviceListWhere(filter DeviceListFilter) (string, []any) {
	clauses := make([]string, 0)
	args := make([]any, 0)
	argPos := 1

	if search := strings.TrimSpace(filter.Search); search != "" {
		clauses = append(clauses, fmt.Sprintf(
			"(name ILIKE $%d OR location ILIKE $%d OR ip_address ILIKE $%d OR description ILIKE $%d)",
			argPos, argPos, argPos, argPos,
		))
		args = append(args, "%"+search+"%")
		argPos++
	}

	if deviceType := strings.TrimSpace(filter.Type); deviceType != "" {
		clauses = append(clauses, fmt.Sprintf("type = $%d", argPos))
		args = append(args, deviceType)
		argPos++
	}

	if status := strings.TrimSpace(filter.Status); status != "" {
		clauses = append(clauses, fmt.Sprintf("status = $%d", argPos))
		args = append(args, status)
	}

	if len(clauses) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(clauses, " AND "), args
}

func resolveDeviceSortColumn(sortBy string) string {
	switch strings.ToLower(strings.TrimSpace(sortBy)) {
	case "name":
		return "name"
	case "type":
		return "type"
	case "status":
		return "status"
	case "updated_at":
		return "updated_at"
	default:
		return "created_at"
	}
}

func resolveSortOrder(sortOrder string) string {
	if strings.EqualFold(strings.TrimSpace(sortOrder), "asc") {
		return "ASC"
	}
	return "DESC"
}

func (r *PostgresDeviceRepository) scanDevice(ctx context.Context, query string, arg any) (*models.Device, error) {
	var device models.Device

	err := r.pool.QueryRow(ctx, query, arg).Scan(
		&device.ID,
		&device.Name,
		&device.Type,
		&device.Status,
		&device.Location,
		&device.IPAddress,
		&device.Description,
		&device.CreatedAt,
		&device.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("query device: %w", err)
	}

	return &device, nil
}
