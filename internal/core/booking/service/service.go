package service

import (
	"context"
	"time"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/domain"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/port"
)

// CreateBookingInput contains data required to create a booking.
type CreateBookingInput struct {
	RoomID    int
	StartTime time.Time
	EndTime   time.Time
}

// UseCase exposes booking operations for inbound adapters.
type UseCase interface {
	ListRooms(ctx context.Context) ([]domain.Room, error)
	GetRoomSchedule(ctx context.Context, roomID int) ([]domain.ScheduleItem, error)
	CreateBooking(ctx context.Context, input CreateBookingInput) (domain.Booking, error)
	CancelBooking(ctx context.Context, bookingID int) error
}

// Service implements booking use cases.
type Service struct {
	roomRepository    port.RoomRepository
	bookingRepository port.BookingRepository
	publisher         port.EventPublisher
	now               func() time.Time
}

// New builds the booking service. A nil publisher is treated as no-op.
func New(roomRepository port.RoomRepository, bookingRepository port.BookingRepository, publisher port.EventPublisher) *Service {
	return &Service{
		roomRepository:    roomRepository,
		bookingRepository: bookingRepository,
		publisher:         publisher,
		now:               time.Now,
	}
}

func (s *Service) ListRooms(ctx context.Context) ([]domain.Room, error) {
	return s.roomRepository.List(ctx)
}

func (s *Service) GetRoomSchedule(ctx context.Context, roomID int) ([]domain.ScheduleItem, error) {
	if _, err := s.roomRepository.GetByID(ctx, roomID); err != nil {
		return nil, err
	}

	bookings, err := s.bookingRepository.ListByRoomID(ctx, roomID)
	if err != nil {
		return nil, err
	}

	schedule := make([]domain.ScheduleItem, 0, len(bookings))
	for _, booking := range bookings {
		schedule = append(schedule, domain.ScheduleItem{
			StartTime: booking.StartTime,
			EndTime:   booking.EndTime,
			Type:      "booking",
		})
	}

	return schedule, nil
}

func (s *Service) CreateBooking(ctx context.Context, input CreateBookingInput) (domain.Booking, error) {
	if err := domain.ValidateTimeRange(input.StartTime, input.EndTime); err != nil {
		return domain.Booking{}, err
	}

	if _, err := s.roomRepository.GetByID(ctx, input.RoomID); err != nil {
		return domain.Booking{}, err
	}

	existingBookings, err := s.bookingRepository.ListByRoomID(ctx, input.RoomID)
	if err != nil {
		return domain.Booking{}, err
	}

	for _, existingBooking := range existingBookings {
		if overlaps(input.StartTime, input.EndTime, existingBooking.StartTime, existingBooking.EndTime) {
			return domain.Booking{}, domain.ErrScheduleConflict
		}
	}

	booking, err := s.bookingRepository.Create(ctx, domain.Booking{
		RoomID:    input.RoomID,
		StartTime: input.StartTime,
		EndTime:   input.EndTime,
	})
	if err != nil {
		return domain.Booking{}, err
	}

	s.publish(ctx, port.Event{
		Type:       port.EventBookingCreated,
		BookingID:  booking.ID,
		RoomID:     booking.RoomID,
		StartTime:  booking.StartTime,
		EndTime:    booking.EndTime,
		OccurredAt: s.now().UTC(),
	})

	return booking, nil
}

func (s *Service) CancelBooking(ctx context.Context, bookingID int) error {
	if err := s.bookingRepository.DeleteByID(ctx, bookingID); err != nil {
		return err
	}

	s.publish(ctx, port.Event{
		Type:       port.EventBookingCancelled,
		BookingID:  bookingID,
		OccurredAt: s.now().UTC(),
	})

	return nil
}

// publish is fire-and-forget: the publisher implementation handles its own
// transport-error logging; the service treats the broker as best-effort.
func (s *Service) publish(ctx context.Context, event port.Event) {
	if s.publisher == nil {
		return
	}
	_ = s.publisher.Publish(ctx, event)
}

func overlaps(startA, endA, startB, endB time.Time) bool {
	return startA.Before(endB) && startB.Before(endA)
}
