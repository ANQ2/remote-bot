package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"remote-bot/internal/domain"
)

type DialogStateRepo struct {
	pool *pgxpool.Pool
}

func NewDialogStateRepo(pool *pgxpool.Pool) *DialogStateRepo {
	return &DialogStateRepo{pool: pool}
}

func (r *DialogStateRepo) Get(ctx context.Context, telegramID int64) (*domain.DialogState, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT telegram_id, step, payload, updated_at
		FROM dialog_states
		WHERE telegram_id = $1
	`, telegramID)

	var st domain.DialogState
	var payloadRaw []byte
	err := row.Scan(&st.TelegramID, &st.Step, &payloadRaw, &st.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return &domain.DialogState{
			TelegramID: telegramID,
			Step:       domain.StepNone,
			Payload:    map[string]string{},
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan dialog state: %w", err)
	}

	if err := json.Unmarshal(payloadRaw, &st.Payload); err != nil {
		return nil, fmt.Errorf("unmarshal dialog payload: %w", err)
	}
	return &st, nil
}

func (r *DialogStateRepo) Set(ctx context.Context, telegramID int64, step domain.DialogStep, payload map[string]string) error {
	if payload == nil {
		payload = map[string]string{}
	}
	payloadRaw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal dialog payload: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO dialog_states (telegram_id, step, payload, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (telegram_id)
		DO UPDATE SET step = $2, payload = $3, updated_at = now()
	`, telegramID, step, payloadRaw)
	if err != nil {
		return fmt.Errorf("upsert dialog state: %w", err)
	}
	return nil
}

func (r *DialogStateRepo) Reset(ctx context.Context, telegramID int64) error {
	return r.Set(ctx, telegramID, domain.StepNone, nil)
}
