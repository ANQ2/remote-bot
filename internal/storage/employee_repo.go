package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"remote-bot/internal/domain"
)

type EmployeeRepo struct {
	pool *pgxpool.Pool
}

func NewEmployeeRepo(pool *pgxpool.Pool) *EmployeeRepo {
	return &EmployeeRepo{pool: pool}
}

func (r *EmployeeRepo) GetOrCreate(ctx context.Context, telegramID int64, username, fullName string) (*domain.Employee, error) {
	emp, err := r.GetByTelegramID(ctx, telegramID)
	if err == nil {
		return emp, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	row := r.pool.QueryRow(ctx, `
	INSERT INTO employees (telegram_id, username, full_name)
	VALUES ($1, $2, $3)
	RETURNING id, telegram_id, username, full_name, is_pm, created_at
	`, telegramID, username, fullName)

	return scanEmployee(row)
}

func (r *EmployeeRepo) GetByTelegramID(ctx context.Context, telegramID int64) (*domain.Employee, error) {
	row := r.pool.QueryRow(ctx, `
	SELECT id, telegram_id, username, full_name, is_pm, created_at
	FROM employees
	WHERE telegram_id = $1
	`, telegramID)

	return scanEmployee(row)
}

func scanEmployee(row pgx.Row) (*domain.Employee, error) {
	var e domain.Employee
	err := row.Scan(&e.ID, &e.TelegramID, &e.Username, &e.FullName, &e.IsPM, &e.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan employee: %w", err)
	}
	return &e, nil
}
