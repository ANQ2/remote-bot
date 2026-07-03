-- Состояние FSM-диалога с пользователем.
-- Хранится в БД, а не в памяти, чтобы рестарт бота не сбрасывал
-- незавершённый ввод сотрудника.
CREATE TABLE dialog_states (
    telegram_id BIGINT PRIMARY KEY,
    step        TEXT NOT NULL DEFAULT '',
    payload     JSONB NOT NULL DEFAULT '{}',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);