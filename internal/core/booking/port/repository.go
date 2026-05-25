package port

import (
	"context"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/domain"
)

// RoomRepository provides room data to the booking core.
type RoomRepository interface {
	List(ctx context.Context) ([]domain.Room, error)
	GetByID(ctx context.Context, id int) (domain.Room, error)
}

// BookingRepository provides booking data to the booking core.
type BookingRepository interface {
	ListByRoomID(ctx context.Context, roomID int) ([]domain.Booking, error)
	Create(ctx context.Context, booking domain.Booking) (domain.Booking, error)
	DeleteByID(ctx context.Context, id int) error
}
