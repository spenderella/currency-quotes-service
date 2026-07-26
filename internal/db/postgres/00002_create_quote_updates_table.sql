-- +goose Up
CREATE TYPE task_status AS ENUM('pending', 'in_progress', 'done', 'failed');

CREATE TABLE IF NOT EXISTS quote_updates (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  base_currency    CHAR(3) NOT NULL REFERENCES currencies(code),
  quote_currency   CHAR(3) NOT NULL REFERENCES currencies(code),
  status           task_status NOT NULL DEFAULT 'pending',
  rate             NUMERIC(18,6),
  provider_time    TIMESTAMPTZ,
  fetched_at       TIMESTAMPTZ,
  error            TEXT,
  idempotency_key  TEXT NOT NULL,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_quote_updates_idempotency_key ON quote_updates (idempotency_key);

CREATE INDEX idx_quote_updates_latest ON quote_updates (base_currency, quote_currency, fetched_at DESC)
  WHERE status = 'done';

-- +goose Down
DROP TABLE IF EXISTS quote_updates;
DROP TYPE IF EXISTS task_status;
