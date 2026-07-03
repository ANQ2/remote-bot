package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"remote-bot/internal/domain"
)

type RequestRepo struct {
	pool *pgxpool.Pool
}

func NewRequestRepo(pool *pgxpool.Pool) *RequestRepo {
	return &RequestRepo{pool: pool}
}

func (r *RequestRepo) CreateRemote(ctx context.Context, employeeID int64, date time.Time) (*domain.Request, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO requests (employee_id, type, date)
		VALUES ($1, 'remote', $2)
		RETURNING id, employee_id, type, date, date_from, date_to, notified, created_at
	`, employeeID, date)
	return scanRequest(row)
}

func (r *RequestRepo) CreateSick(ctx context.Context, employeeID int64, dateFrom, dateTo time.Time) (*domain.Request, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO requests (employee_id, type, date, date_from, date_to)
		VALUES ($1, 'sick', $2, $2, $3)
		RETURNING id, employee_id, type, date, date_from, date_to, notified, created_at
	`, employeeID, dateFrom, dateTo)
	return scanRequest(row)
}

func (r *RequestRepo) PendingForDate(ctx context.Context, date time.Time) ([]domain.RequestWithEmployee, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT r.id, r.employee_id, r.type, r.date, r.date_from, r.date_to, r.notified, r.created_at,
		       e.full_name, e.telegram_id, e.username
		FROM requests r
		JOIN employees e ON e.id = r.employee_id
		WHERE (
			(r.type = 'remote' AND r.date = $1 AND r.notified = false) OR
			(r.type = 'sick' AND r.date_from <= $1 AND r.date_to >= $1)
		)
	`, date)
	if err != nil {
		return nil, fmt.Errorf("query pending requests: %w", err)
	}
	defer rows.Close()

	var result []domain.RequestWithEmployee
	for rows.Next() {
		var rw domain.RequestWithEmployee
		if err := rows.Scan(
			&rw.ID, &rw.EmployeeID, &rw.Type, &rw.Date, &rw.DateFrom, &rw.DateTo, &rw.Notified, &rw.CreatedAt,
			&rw.EmployeeFullName, &rw.EmployeeTelegramID, &rw.EmployeeUsername,
		); err != nil {
			return nil, fmt.Errorf("scan pending request: %w", err)
		}
		result = append(result, rw)
	}
	return result, rows.Err()
}

func (r *RequestRepo) MarkNotified(ctx context.Context, id int64, reqType domain.RequestType) error {
	if reqType == domain.RequestSick {
		// Для больничного не помечаем notified — уведомляем каждый день
		return nil
	}
	_, err := r.pool.Exec(ctx, `UPDATE requests SET notified = true WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("mark request notified: %w", err)
	}
	return nil
}

func scanRequest(row pgx.Row) (*domain.Request, error) {
	var req domain.Request
	err := row.Scan(&req.ID, &req.EmployeeID, &req.Type, &req.Date, &req.DateFrom, &req.DateTo, &req.Notified, &req.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan request: %w", err)
	}
	return &req, nil
}
