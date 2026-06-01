-- +goose Up
-- +goose StatementBegin
ALTER TABLE bookings
    ADD COLUMN subject     TEXT,
    ADD COLUMN lesson_type TEXT,
    ADD COLUMN week        TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE bookings
    DROP COLUMN IF EXISTS week,
    DROP COLUMN IF EXISTS lesson_type,
    DROP COLUMN IF EXISTS subject;
-- +goose StatementEnd
