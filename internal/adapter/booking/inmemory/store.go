package inmemory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/domain"
)

// Package inmemory contains a temporary stub outbound adapter used until a real database-backed implementation is added.

// Store keeps stub booking data in memory and implements booking repository ports.
// It is intended only for the bootstrap stage of the booking service and should later
// be replaced with a persistent storage adapter.
type Store struct {
	mu       sync.RWMutex
	rooms    map[int]domain.Room
	bookings map[int]domain.Booking
	nextID   int
}

func NewStore() *Store {
	rooms := []domain.Room{
		{ID: 1, Name: "A101", Capacity: 12, Building: "North"},
		{ID: 2, Name: "B204", Capacity: 24, Building: "South"},
		{ID: 3, Name: "C305", Capacity: 8, Building: "West"},
	}

	bookings := []domain.Booking{
		{
			ID:        1,
			RoomID:    1,
			StartTime: time.Date(2026, time.April, 17, 9, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2026, time.April, 17, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:        2,
			RoomID:    2,
			StartTime: time.Date(2026, time.April, 17, 11, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2026, time.April, 17, 12, 30, 0, 0, time.UTC),
		},
	}

	store := &Store{
		rooms:    make(map[int]domain.Room, len(rooms)),
		bookings: make(map[int]domain.Booking, len(bookings)),
		nextID:   3,
	}

	for _, room := range rooms {
		store.rooms[room.ID] = room
	}

	for _, booking := range bookings {
		store.bookings[booking.ID] = booking
	}

	return store
}

func (s *Store) List(ctx context.Context) ([]domain.Room, error) {
	_ = ctx

	s.mu.RLock()
	defer s.mu.RUnlock()

	rooms := make([]domain.Room, 0, len(s.rooms))
	for _, room := range s.rooms {
		rooms = append(rooms, room)
	}

	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].ID < rooms[j].ID
	})

	return rooms, nil
}

func (s *Store) GetByID(ctx context.Context, id int) (domain.Room, error) {
	_ = ctx

	s.mu.RLock()
	defer s.mu.RUnlock()

	room, ok := s.rooms[id]
	if !ok {
		return domain.Room{}, domain.ErrRoomNotFound
	}

	return room, nil
}

func (s *Store) ListByRoomID(ctx context.Context, roomID int) ([]domain.Booking, error) {
	_ = ctx

	s.mu.RLock()
	defer s.mu.RUnlock()

	bookings := make([]domain.Booking, 0)
	for _, booking := range s.bookings {
		if booking.RoomID == roomID {
			bookings = append(bookings, booking)
		}
	}

	sort.Slice(bookings, func(i, j int) bool {
		if bookings[i].StartTime.Equal(bookings[j].StartTime) {
			return bookings[i].ID < bookings[j].ID
		}

		return bookings[i].StartTime.Before(bookings[j].StartTime)
	})

	return bookings, nil
}

func (s *Store) Create(ctx context.Context, booking domain.Booking) (domain.Booking, error) {
	_ = ctx

	s.mu.Lock()
	defer s.mu.Unlock()

	booking.ID = s.nextID
	s.nextID++
	s.bookings[booking.ID] = booking

	return booking, nil
}

func (s *Store) DeleteByID(ctx context.Context, id int) error {
	_ = ctx

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.bookings[id]; !ok {
		return domain.ErrBookingNotFound
	}

	delete(s.bookings, id)

	return nil
}
