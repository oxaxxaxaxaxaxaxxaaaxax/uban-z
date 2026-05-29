-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
    ADD COLUMN full_name TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users
    DROP COLUMN IF EXISTS full_name;
-- +goose StatementEnd
