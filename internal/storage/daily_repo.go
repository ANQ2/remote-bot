package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"remote-bot/internal/domain"
)

type DailyRepo struct {
	pool *pgxpool.Pool
}

func NewDailyRepo(pool *pgxpool.Pool) *DailyRepo {
	return &DailyRepo{pool: pool}
}

func (r *DailyRepo) Create(ctx context.Context, date time.Time, timeStr string, mode domain.DailyMode, location string, createdBy int64) (*domain.Daily, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO dailies (date, "time", mode, location, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, date, "time", mode, location, created_by, notified, created_at
	`, date, timeStr, mode, location, createdBy)
	return scanDaily(row)
}

func (r *DailyRepo) PendingForDateTime(ctx context.Context, date time.Time, timeStr string) ([]domain.Daily, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, date, "time", mode, location, created_by, notified, created_at
		FROM dailies
		WHERE date = $1 AND "time" = $2 AND notified = false
	`, date, timeStr)
	if err != nil {
		return nil, fmt.Errorf("query pending dailies: %w", err)
	}
	defer rows.Close()

	var result []domain.Daily
	for rows.Next() {
		var d domain.Daily
		if err := rows.Scan(&d.ID, &d.Date, &d.Time, &d.Mode, &d.Location, &d.CreatedBy, &d.Notified, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan daily: %w", err)
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func (r *DailyRepo) GetLastByMode(ctx context.Context, mode domain.DailyMode) (*domain.Daily, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, date, "time", mode, location, created_by, notified, created_at
		FROM dailies
		WHERE mode = $1 AND location != ''
		ORDER BY created_at DESC
		LIMIT 1
	`, mode)
	d, err := scanDaily(row)
	if err != nil {
		return nil, nil
	}
	return d, nil
}

func (r *DailyRepo) MarkNotified(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `UPDATE dailies SET notified = true WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("mark daily notified: %w", err)
	}
	return nil
}

func (r *DailyRepo) DeleteLastByCreator(ctx context.Context, createdBy int64) (*domain.Daily, error) {
	row := r.pool.QueryRow(ctx, `
		DELETE FROM dailies
		WHERE id = (
			SELECT id FROM dailies
			WHERE created_by = $1
			ORDER BY created_at DESC
			LIMIT 1
		)
		RETURNING id, date, "time", mode, location, created_by, notified, created_at
	`, createdBy)
	return scanDaily(row)
}

func scanDaily(row pgx.Row) (*domain.Daily, error) {
	var d domain.Daily
	err := row.Scan(&d.ID, &d.Date, &d.Time, &d.Mode, &d.Location, &d.CreatedBy, &d.Notified, &d.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan daily: %w", err)
	}
	return &d, nil
}
