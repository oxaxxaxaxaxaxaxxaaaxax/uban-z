-- name: ListRooms :many
SELECT id, name, capacity, building, created_at
FROM rooms
ORDER BY id;

-- name: GetRoomByID :one
SELECT id, name, capacity, building, created_at
FROM rooms
WHERE id = $1;

-- name: ListBookingsByRoomID :many
SELECT id, room_id, start_time, end_time, created_at
FROM bookings
WHERE room_id = $1
ORDER BY start_time, id;

-- name: CreateBooking :one
INSERT INTO bookings (room_id, start_time, end_time)
VALUES ($1, $2, $3)
RETURNING id, room_id, start_time, end_time, created_at;

-- name: DeleteBooking :execrows
DELETE FROM bookings
WHERE id = $1;
