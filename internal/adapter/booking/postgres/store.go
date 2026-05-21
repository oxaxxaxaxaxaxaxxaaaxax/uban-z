package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/postgres/sqlcgen"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/domain"
)

const (
	constraintNoOverlap  = "bookings_no_overlap_excl"
	constraintTimeRange  = "bookings_time_range_chk"
	pgCodeExclusionViol  = "23P01"
	pgCodeCheckViolation = "23514"
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
	rows, err := s.queries.ListBookingsByRoomID(ctx, int64(roomID))
	if err != nil {
		return nil, fmt.Errorf("list bookings by room id: %w", err)
	}

	bookings := make([]domain.Booking, 0, len(rows))
	for _, b := range rows {
		bookings = append(bookings, toDomainBooking(b))
	}
	return bookings, nil
}

func (s *Store) Create(ctx context.Context, booking domain.Booking) (domain.Booking, error) {
	row, err := s.queries.CreateBooking(ctx, sqlcgen.CreateBookingParams{
		RoomID:    int64(booking.RoomID),
		StartTime: pgtype.Timestamptz{Time: booking.StartTime, Valid: true},
		EndTime:   pgtype.Timestamptz{Time: booking.EndTime, Valid: true},
	})
	if err != nil {
		return domain.Booking{}, translateInsertError(err)
	}
	return toDomainBooking(row), nil
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

func toDomainBooking(b sqlcgen.Booking) domain.Booking {
	return domain.Booking{
		ID:        int(b.ID),
		RoomID:    int(b.RoomID),
		StartTime: b.StartTime.Time,
		EndTime:   b.EndTime.Time,
	}
}
