-- Сотрудники
CREATE TABLE employees (
    id          SERIAL PRIMARY KEY,
    telegram_id BIGINT NOT NULL UNIQUE,
    username    TEXT,
    full_name   TEXT NOT NULL,
    is_pm       BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Заявки на удалёнку / больничный
CREATE TABLE requests (
    id          SERIAL PRIMARY KEY,
    employee_id INTEGER NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    type        TEXT NOT NULL CHECK (type IN ('remote', 'sick')),
    date        DATE NOT NULL,
    notified    BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_requests_date_notified ON requests(date, notified);

-- Дэйлики
CREATE TABLE dailies (
    id          SERIAL PRIMARY KEY,
    date        DATE NOT NULL,
    time        TEXT NOT NULL, -- формат "HH:MM"
    mode        TEXT NOT NULL CHECK (mode IN ('online', 'offline')),
    created_by  BIGINT NOT NULL, -- telegram_id ПМа
    notified    BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_dailies_date_time_notified ON dailies(date, time, notified);