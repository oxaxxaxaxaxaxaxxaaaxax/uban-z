-- +goose Up
-- +goose StatementBegin
INSERT INTO rooms (name, capacity, building) VALUES
    ('A101', 12, 'North'),
    ('B204', 24, 'South'),
    ('C305',  8, 'West')
ON CONFLICT (building, name) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- intentionally no-op: keep dev seed idempotent and avoid
-- destroying rows that may have been edited locally.
-- +goose StatementEnd
