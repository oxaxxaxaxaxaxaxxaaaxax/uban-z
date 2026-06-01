-- +goose Up
-- +goose StatementBegin
-- Dev rooms used to be seeded here. Real rooms are now imported from NSU on
-- booking-service startup, so fresh databases should not get fake buildings.
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- intentionally no-op
-- +goose StatementEnd
