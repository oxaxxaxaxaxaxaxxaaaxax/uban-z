-- +goose Up
-- +goose StatementBegin
CREATE TYPE user_role AS ENUM (
    'student_b',
    'student_m',
    'student_a',
    'teacher',
    'admin'
);

CREATE TABLE users (
    id            BIGSERIAL    PRIMARY KEY,
    login         TEXT         NOT NULL,
    password_hash TEXT         NOT NULL,
    role          user_role    NOT NULL DEFAULT 'student_b',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX users_login_uniq ON users (login);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS user_role;
-- +goose StatementEnd
