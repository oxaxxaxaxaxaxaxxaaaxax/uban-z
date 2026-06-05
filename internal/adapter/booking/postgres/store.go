package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/postgres/sqlcgen"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/domain"
	parserdomain "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/parser/domain"
)

const (
	constraintNoOverlap         = "bookings_no_overlap_excl"
	constraintTimeRange         = "bookings_time_range_chk"
	defaultImportedRoomCapacity = 30
	pgCodeExclusionViol         = "23P01"
	pgCodeCheckViolation        = "23514"
)

type Store struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
}

func NewStoreFromPool(pool *pgxpool.Pool) *Store {
	return &Store{
		pool:    pool,
		queries: sqlcgen.New(pool),
	}
}

func (s *Store) List(ctx context.Context) ([]domain.Room, error) {
	rows, err := s.queries.ListRooms(ctx)
	if err != nil {
		return nil, fmt.Errorf("list rooms: %w", err)
	}

	rooms := make([]domain.Room, 0, len(rows))
	for _, r := range rows {
		rooms = append(rooms, toDomainRoom(r))
	}
	return rooms, nil
}

func (s *Store) GetByID(ctx context.Context, id int) (domain.Room, error) {
	row, err := s.queries.GetRoomByID(ctx, int64(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Room{}, domain.ErrRoomNotFound
		}
		return domain.Room{}, fmt.Errorf("get room by id: %w", err)
	}
	return toDomainRoom(row), nil
}

func (s *Store) ListByRoomID(ctx context.Context, roomID int) ([]domain.Booking, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, room_id, user_id, creator_role, start_time, end_time,
		       subject, lesson_type, teacher, group_numbers, week
		FROM bookings
		WHERE room_id = $1
		ORDER BY start_time, id
	`, int64(roomID))
	if err != nil {
		return nil, fmt.Errorf("list bookings by room id: %w", err)
	}
	defer rows.Close()

	bookings := make([]domain.Booking, 0)
	for rows.Next() {
		booking, err := scanBooking(rows)
		if err != nil {
			return nil, fmt.Errorf("scan booking by room id: %w", err)
		}
		bookings = append(bookings, booking)
	}
	return bookings, rows.Err()
}

func (s *Store) ListByUserID(ctx context.Context, userID int) ([]domain.Booking, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, room_id, user_id, creator_role, start_time, end_time,
		       subject, lesson_type, teacher, group_numbers, week
		FROM bookings
		WHERE user_id = $1
		ORDER BY start_time, id
	`, int64(userID))
	if err != nil {
		return nil, fmt.Errorf("list bookings by user id: %w", err)
	}
	defer rows.Close()

	bookings := make([]domain.Booking, 0)
	for rows.Next() {
		booking, err := scanBooking(rows)
		if err != nil {
			return nil, fmt.Errorf("scan booking by user id: %w", err)
		}
		bookings = append(bookings, booking)
	}
	return bookings, rows.Err()
}

func (s *Store) GetBookingByID(ctx context.Context, id int) (domain.Booking, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, room_id, user_id, creator_role, start_time, end_time,
		       subject, lesson_type, teacher, group_numbers, week
		FROM bookings
		WHERE id = $1
	`, int64(id))
	booking, err := scanBooking(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Booking{}, domain.ErrBookingNotFound
		}
		return domain.Booking{}, fmt.Errorf("get booking by id: %w", err)
	}
	return booking, nil
}

func (s *Store) Create(ctx context.Context, booking domain.Booking) (domain.Booking, error) {
	row, err := s.queries.CreateBooking(ctx, sqlcgen.CreateBookingParams{
		RoomID:      int64(booking.RoomID),
		UserID:      int64(booking.UserID),
		CreatorRole: string(booking.CreatorRole),
		StartTime:   pgtype.Timestamptz{Time: booking.StartTime, Valid: true},
		EndTime:     pgtype.Timestamptz{Time: booking.EndTime, Valid: true},
	})
	if err != nil {
		return domain.Booking{}, translateInsertError(err)
	}
	return domain.Booking{
		ID:           int(row.ID),
		RoomID:       int(row.RoomID),
		UserID:       int(row.UserID),
		CreatorRole:  domain.Role(row.CreatorRole),
		StartTime:    row.StartTime.Time,
		EndTime:      row.EndTime.Time,
		Subject:      textOr(row.Subject),
		LessonType:   textOr(row.LessonType),
		Teacher:      textOr(row.Teacher),
		GroupNumbers: row.GroupNumbers,
		Week:         textOr(row.Week),
	}, nil
}

func (s *Store) DeleteByID(ctx context.Context, id int) error {
	rows, err := s.queries.DeleteBooking(ctx, int64(id))
	if err != nil {
		return fmt.Errorf("delete booking: %w", err)
	}
	if rows == 0 {
		return domain.ErrBookingNotFound
	}
	return nil
}

func (s *Store) HasParsedSchedule(ctx context.Context) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM bookings
			WHERE user_id = 0 AND creator_role = $1
			LIMIT 1
		)
	`, string(domain.RoleAdmin)).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check parsed schedule: %w", err)
	}
	return exists, nil
}

func (s *Store) ReplaceParsedSchedule(ctx context.Context, rooms []parserdomain.RoomSelector, slots []parserdomain.ScheduleSlot) (parserdomain.ImportStats, error) {
	stats := parserdomain.ImportStats{
		RoomsSeen:       len(rooms),
		LessonsExpanded: len(slots),
	}

	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			DELETE FROM bookings
			WHERE user_id = 0 AND creator_role = $1
		`, string(domain.RoleAdmin)); err != nil {
			return fmt.Errorf("delete previous parser rows: %w", err)
		}

		if _, err := tx.Exec(ctx, `
			DELETE FROM rooms r
			WHERE (
					(r.building, r.name) IN (('North', 'A101'), ('South', 'B204'), ('West', 'C305'))
					OR (r.building = 'НГУ' AND r.name LIKE '%(%)')
				)
				AND NOT EXISTS (
					SELECT 1
					FROM bookings b
					WHERE b.room_id = r.id
				)
		`); err != nil {
			return fmt.Errorf("delete stale parser rooms: %w", err)
		}

		roomIDs := make(map[string]int64, len(rooms))
		for _, room := range rooms {
			name, building, capacity := normalizeParsedRoom(room.Name, room.Building, room.Capacity)
			id, err := upsertParsedRoom(ctx, tx, name, building, capacity)
			if err != nil {
				return err
			}
			roomIDs[roomKey(building, name)] = id
			stats.RoomsImported++
		}

		for _, slot := range slots {
			name, building, capacity := normalizeParsedRoom(slot.RoomName, slot.Building, slot.Capacity)
			key := roomKey(building, name)
			roomID, ok := roomIDs[key]
			if !ok {
				id, err := upsertParsedRoom(ctx, tx, name, building, capacity)
				if err != nil {
					return err
				}
				roomID = id
				roomIDs[key] = id
				stats.RoomsImported++
			}

			tag, err := tx.Exec(ctx, `
				INSERT INTO bookings (
					room_id, user_id, creator_role, start_time, end_time,
					subject, lesson_type, teacher, group_numbers, week
				)
				VALUES ($1, 0, $2, $3, $4, $5, $6, $7, $8, $9)
				ON CONFLICT DO NOTHING
			`,
				roomID,
				string(domain.RoleAdmin),
				pgtype.Timestamptz{Time: slot.StartTime, Valid: true},
				pgtype.Timestamptz{Time: slot.EndTime, Valid: true},
				nullableText(slot.Subject),
				nullableText(slot.LessonType),
				nullableText(slot.Teacher),
				slot.GroupNumbers,
				nullableText(slot.Week),
			)
			if err != nil {
				return translateInsertError(err)
			}
			if tag.RowsAffected() == 0 {
				stats.LessonsSkipped++
				continue
			}
			stats.LessonsImported++
		}

		return nil
	})
	if err != nil {
		return parserdomain.ImportStats{}, err
	}

	return stats, nil
}

func translateInsertError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgCodeExclusionViol:
			if pgErr.ConstraintName == constraintNoOverlap {
				return domain.ErrScheduleConflict
			}
		case pgCodeCheckViolation:
			if pgErr.ConstraintName == constraintTimeRange {
				return domain.ErrInvalidTimeRange
			}
		}
	}
	return fmt.Errorf("create booking: %w", err)
}

func toDomainRoom(r sqlcgen.Room) domain.Room {
	return domain.Room{
		ID:       int(r.ID),
		Name:     r.Name,
		Capacity: int(r.Capacity),
		Building: r.Building,
	}
}

type bookingScanner interface {
	Scan(dest ...any) error
}

func scanBooking(row bookingScanner) (domain.Booking, error) {
	var (
		id           int64
		roomID       int64
		userID       int64
		creatorRole  string
		startTime    pgtype.Timestamptz
		endTime      pgtype.Timestamptz
		subject      pgtype.Text
		lessonType   pgtype.Text
		teacher      pgtype.Text
		groupNumbers []string
		week         pgtype.Text
	)
	if err := row.Scan(
		&id,
		&roomID,
		&userID,
		&creatorRole,
		&startTime,
		&endTime,
		&subject,
		&lessonType,
		&teacher,
		&groupNumbers,
		&week,
	); err != nil {
		return domain.Booking{}, err
	}

	return domain.Booking{
		ID:           int(id),
		RoomID:       int(roomID),
		UserID:       int(userID),
		CreatorRole:  domain.Role(creatorRole),
		StartTime:    startTime.Time,
		EndTime:      endTime.Time,
		Subject:      textOr(subject),
		LessonType:   textOr(lessonType),
		Teacher:      textOr(teacher),
		GroupNumbers: groupNumbers,
		Week:         textOr(week),
	}, nil
}

func textOr(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

func nullableText(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func upsertParsedRoom(ctx context.Context, tx pgx.Tx, name string, building string, capacity int) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO rooms (name, capacity, building)
		VALUES ($1, $2, $3)
		ON CONFLICT (building, name) DO UPDATE
		SET capacity = EXCLUDED.capacity
		RETURNING id
	`, name, capacity, building).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("upsert parsed room %q/%q: %w", building, name, err)
	}
	return id, nil
}

func normalizeParsedRoom(name string, building string, capacity int) (string, string, int) {
	if building == "" {
		parsedName, parsedBuilding := splitParsedRoomName(name)
		if parsedName != "" {
			name = parsedName
		}
		if parsedBuilding != "" {
			building = parsedBuilding
		}
	}
	if name == "" {
		name = "unknown"
	}
	if building == "" {
		building = "НГУ"
	}
	if capacity <= 0 {
		capacity = defaultImportedRoomCapacity
	}
	return name, building, capacity
}

func splitParsedRoomName(raw string) (string, string) {
	room := strings.TrimSpace(raw)
	open := strings.LastIndex(room, "(")
	close := strings.LastIndex(room, ")")
	if open > 0 && close > open {
		name := strings.TrimSpace(room[:open])
		building := strings.TrimSpace(room[open+1 : close])
		if name != "" && building != "" {
			return name, building
		}
	}
	return room, ""
}

func roomKey(building string, name string) string {
	return building + "\x00" + name
}
