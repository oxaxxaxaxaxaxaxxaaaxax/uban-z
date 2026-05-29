-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE bookings (
    id           BIGSERIAL    PRIMARY KEY,
    room_id      BIGINT       NOT NULL REFERENCES rooms (id) ON DELETE RESTRICT,
    user_id      BIGINT       NOT NULL,
    creator_role TEXT         NOT NULL,
    start_time   TIMESTAMPTZ  NOT NULL,
    end_time     TIMESTAMPTZ  NOT NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT bookings_time_range_chk CHECK (start_time < end_time),
    CONSTRAINT bookings_creator_role_chk CHECK (
        creator_role IN ('student_b', 'student_m', 'student_a', 'teacher', 'admin')
    ),
    CONSTRAINT bookings_no_overlap_excl EXCLUDE USING gist (
        room_id WITH =,
        tstzrange(start_time, end_time, '[)') WITH &&
    )
);

CREATE INDEX bookings_room_start_idx ON bookings (room_id, start_time);
CREATE INDEX bookings_user_start_idx ON bookings (user_id, start_time);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS bookings;
DROP EXTENSION IF EXISTS btree_gist;
-- +goose StatementEnd
