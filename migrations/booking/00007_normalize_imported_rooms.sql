-- +goose Up
-- +goose StatementBegin
UPDATE rooms
SET capacity = 30
WHERE capacity = 1;

DELETE FROM bookings
WHERE user_id = 0
  AND creator_role = 'admin'
  AND (
      lower(coalesce(subject, '')) LIKE 'забронировано%'
      OR (
          coalesce(subject, '') <> ''
          AND coalesce(lesson_type, '') = ''
          AND coalesce(teacher, '') = ''
          AND cardinality(coalesce(group_numbers, ARRAY[]::text[])) = 0
      )
  );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- intentionally no-op: this only normalizes imported parser data
-- +goose StatementEnd
