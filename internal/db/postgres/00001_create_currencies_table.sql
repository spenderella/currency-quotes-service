-- +goose Up

CREATE TABLE IF NOT EXISTS currencies (
  code    CHAR(3) PRIMARY KEY,
  enabled BOOLEAN NOT NULL DEFAULT true
);

INSERT INTO currencies (code) VALUES
  ('CAD'),
  ('EUR'),
  ('GBP'),
  ('MXN'),
  ('USD');

-- +goose Down

DROP TABLE IF EXISTS currencies;
