-- +goose Up
CREATE TABLE IF NOT EXISTS users
(
    tg_id BIGINT PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS groups
(
    id         BIGSERIAL PRIMARY KEY,
    group_name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS students
(
    id       BIGSERIAL PRIMARY KEY,
    user_id  BIGINT NOT NULL UNIQUE,
    group_id BIGINT NOT NULL,
    login    TEXT,
    password TEXT,
    CONSTRAINT fk_students_user
        FOREIGN KEY (user_id) REFERENCES users (tg_id)
            ON DELETE CASCADE,
    CONSTRAINT fk_students_group
        FOREIGN KEY (group_id) REFERENCES groups (id)
            ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS teachers
(
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT NOT NULL UNIQUE,
    teacher_name TEXT UNIQUE,
    login        TEXT,
    password     TEXT,
    CONSTRAINT fk_teachers_user
        FOREIGN KEY (user_id) REFERENCES users (tg_id)
            ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS auditoriums
(
    id        BIGSERIAL PRIMARY KEY,
    room_name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS lessons
(
    id          BIGSERIAL PRIMARY KEY,
    subject     TEXT     NOT NULL,
    lesson_type TEXT,
    weekday     SMALLINT NOT NULL CHECK (weekday BETWEEN 1 AND 7),
    start_time  TIME     NOT NULL,
    week        SMALLINT CHECK (week IN (1, 2) OR week IS NULL),
    teacher_id  BIGINT   NOT NULL,
    room_id     BIGINT   NOT NULL,
    CONSTRAINT fk_lessons_teacher
        FOREIGN KEY (teacher_id) REFERENCES teachers (id)
            ON DELETE CASCADE,
    CONSTRAINT fk_lessons_room
        FOREIGN KEY (room_id) REFERENCES auditoriums (id)
            ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS lesson_groups
(
    lesson_id BIGINT NOT NULL,
    group_id  BIGINT NOT NULL,
    PRIMARY KEY (lesson_id, group_id),
    CONSTRAINT fk_lesson_groups_lesson
        FOREIGN KEY (lesson_id) REFERENCES lessons (id)
            ON DELETE CASCADE,
    CONSTRAINT fk_lesson_groups_group
        FOREIGN KEY (group_id) REFERENCES groups (id)
            ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS homeworks
(
    id         BIGSERIAL PRIMARY KEY,
    text       TEXT NOT NULL,
    status     TEXT,
    teacher_id BIGINT,
    CONSTRAINT fk_homeworks_teacher
        FOREIGN KEY (teacher_id) REFERENCES teachers (id)
            ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS group_homeworks
(
    group_id    BIGINT NOT NULL,
    homework_id BIGINT NOT NULL,
    PRIMARY KEY (group_id, homework_id),
    CONSTRAINT fk_group_homeworks_group
        FOREIGN KEY (group_id) REFERENCES groups (id)
            ON DELETE CASCADE,
    CONSTRAINT fk_group_homeworks_homework
        FOREIGN KEY (homework_id) REFERENCES homeworks (id)
            ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS student_homework_statuses
(
    student_id  BIGINT NOT NULL,
    homework_id BIGINT NOT NULL,
    status      TEXT   NOT NULL,
    PRIMARY KEY (student_id, homework_id),
    CONSTRAINT fk_student_homework_statuses_student
        FOREIGN KEY (student_id) REFERENCES students (id)
            ON DELETE CASCADE,
    CONSTRAINT fk_student_homework_statuses_homework
        FOREIGN KEY (homework_id) REFERENCES homeworks (id)
            ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS notifications
(
    id          BIGSERIAL PRIMARY KEY,
    "timestamp" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id     BIGINT      NOT NULL,
    homework_id BIGINT      NOT NULL,
    CONSTRAINT fk_notifications_user
        FOREIGN KEY (user_id) REFERENCES users (tg_id)
            ON DELETE CASCADE,
    CONSTRAINT fk_notifications_homework
        FOREIGN KEY (homework_id) REFERENCES homeworks (id)
            ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_students_group_id
    ON students (group_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_students_login
    ON students (login)
    WHERE login IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_teachers_login
    ON teachers (login)
    WHERE login IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_lesson_groups_group_id
    ON lesson_groups (group_id);

CREATE INDEX IF NOT EXISTS idx_lessons_teacher_id
    ON lessons (teacher_id);

CREATE INDEX IF NOT EXISTS idx_lessons_room_id
    ON lessons (room_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_lessons_unique_lookup
    ON lessons (subject, COALESCE(lesson_type, ''), weekday, start_time, COALESCE(week, 0), teacher_id, room_id);

CREATE INDEX IF NOT EXISTS idx_homeworks_teacher_id
    ON homeworks (teacher_id);

CREATE INDEX IF NOT EXISTS idx_student_homework_statuses_homework_id
    ON student_homework_statuses (homework_id);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id
    ON notifications (user_id);

CREATE INDEX IF NOT EXISTS idx_notifications_homework_id
    ON notifications (homework_id);

-- +goose Down
DROP TABLE IF EXISTS notifications CASCADE;
DROP TABLE IF EXISTS student_homework_statuses CASCADE;
DROP TABLE IF EXISTS group_homeworks CASCADE;
DROP TABLE IF EXISTS homeworks CASCADE;
DROP TABLE IF EXISTS lesson_groups CASCADE;
DROP TABLE IF EXISTS lessons CASCADE;
DROP TABLE IF EXISTS auditoriums CASCADE;
DROP TABLE IF EXISTS teachers CASCADE;
DROP TABLE IF EXISTS students CASCADE;
DROP TABLE IF EXISTS groups CASCADE;
DROP TABLE IF EXISTS users CASCADE;
