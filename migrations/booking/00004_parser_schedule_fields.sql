-- +goose Up
-- +goose StatementBegin
ALTER TABLE bookings
    ADD COLUMN teacher       TEXT,
    ADD COLUMN group_numbers TEXT[];
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE bookings
    DROP COLUMN IF EXISTS group_numbers,
    DROP COLUMN IF EXISTS teacher;
-- +goose StatementEnd
