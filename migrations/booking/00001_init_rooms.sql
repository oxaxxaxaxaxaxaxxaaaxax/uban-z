-- +goose Up
-- +goose StatementBegin
CREATE TABLE rooms (
    id         BIGSERIAL    PRIMARY KEY,
    name       TEXT         NOT NULL,
    capacity   INT          NOT NULL CHECK (capacity > 0),
    building   TEXT         NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX rooms_building_name_uniq ON rooms (building, name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS rooms;
-- +goose StatementEnd
