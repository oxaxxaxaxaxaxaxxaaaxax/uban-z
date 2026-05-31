package service

import (
	"context"
	"time"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/domain"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/port"
)

// Caller identifies the authenticated user making a request.
type Caller struct {
	UserID int
	Login  string
	Role   domain.Role
}

// CreateBookingInput contains data required to create a booking.
type CreateBookingInput struct {
	Caller    Caller
	RoomID    int
	StartTime time.Time
	EndTime   time.Time
}

// UseCase exposes booking operations for inbound adapters.
type UseCase interface {
	ListRooms(ctx context.Context) ([]domain.Room, error)
	GetRoomSchedule(ctx context.Context, roomID int) ([]domain.ScheduleItem, error)
	ListMyBookings(ctx context.Context, caller Caller) ([]domain.Booking, error)
	CreateBooking(ctx context.Context, input CreateBookingInput) (domain.Booking, error)
	CancelBooking(ctx context.Context, bookingID int, caller Caller) error
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
			StartTime:    booking.StartTime,
			EndTime:      booking.EndTime,
			Type:         "booking",
			Teacher:      booking.Teacher,
			GroupNumbers: booking.GroupNumbers,
		})
	}

	return schedule, nil
}

// ListMyBookings returns bookings owned by the caller, ordered by start_time.
func (s *Service) ListMyBookings(ctx context.Context, caller Caller) ([]domain.Booking, error) {
	if !caller.Role.IsKnown() {
		return nil, domain.ErrForbidden
	}
	return s.bookingRepository.ListByUserID(ctx, caller.UserID)
}

func (s *Service) CreateBooking(ctx context.Context, input CreateBookingInput) (domain.Booking, error) {
	if !input.Caller.Role.IsKnown() {
		return domain.Booking{}, domain.ErrForbidden
	}

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
		RoomID:      input.RoomID,
		UserID:      input.Caller.UserID,
		CreatorRole: input.Caller.Role,
		StartTime:   input.StartTime,
		EndTime:     input.EndTime,
	})
	if err != nil {
		return domain.Booking{}, err
	}

	s.publish(ctx, port.Event{
		Type:       port.EventBookingCreated,
		BookingID:  booking.ID,
		RoomID:     booking.RoomID,
		OwnerID:    booking.UserID,
		OwnerRole:  booking.CreatorRole,
		StartTime:  booking.StartTime,
		EndTime:    booking.EndTime,
		OccurredAt: s.now().UTC(),
	})

	return booking, nil
}

func (s *Service) CancelBooking(ctx context.Context, bookingID int, caller Caller) error {
	if !caller.Role.IsKnown() {
		return domain.ErrForbidden
	}

	booking, err := s.bookingRepository.GetBookingByID(ctx, bookingID)
	if err != nil {
		return err
	}

	selfCancel := booking.UserID == caller.UserID
	if !selfCancel && !caller.Role.CanCancelOther(booking.CreatorRole) {
		return domain.ErrForbidden
	}

	if err := s.bookingRepository.DeleteByID(ctx, bookingID); err != nil {
		return err
	}

	s.publish(ctx, port.Event{
		Type:       port.EventBookingCancelled,
		BookingID:  booking.ID,
		RoomID:     booking.RoomID,
		OwnerID:    booking.UserID,
		OwnerRole:  booking.CreatorRole,
		StartTime:  booking.StartTime,
		EndTime:    booking.EndTime,
		CancelledBy: &port.Actor{
			UserID: caller.UserID,
			Login:  caller.Login,
			Role:   caller.Role,
		},
		SelfCancel: selfCancel,
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
