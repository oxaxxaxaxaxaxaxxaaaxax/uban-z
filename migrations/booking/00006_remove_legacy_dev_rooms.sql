-- +goose Up
-- +goose StatementBegin
DELETE FROM bookings
WHERE room_id IN (
    SELECT id
    FROM rooms
    WHERE (building, name) IN (('North', 'A101'), ('South', 'B204'), ('West', 'C305'))
);

DELETE FROM rooms
WHERE (building, name) IN (('North', 'A101'), ('South', 'B204'), ('West', 'C305'));

DELETE FROM rooms r
WHERE r.building = 'НГУ'
  AND r.name LIKE '%(%)'
  AND NOT EXISTS (
      SELECT 1
      FROM bookings b
      WHERE b.room_id = r.id
  );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- intentionally no-op: legacy fake rooms should not be restored automatically
-- +goose StatementEnd
