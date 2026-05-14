package service_test

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/domain"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/port"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/service"
)

type fakePublisher struct {
	mu       sync.Mutex
	events   []port.Event
	errOn    string // event type to fail on
	failWith error
}

func (p *fakePublisher) Publish(_ context.Context, event port.Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.errOn != "" && event.Type == p.errOn {
		return p.failWith
	}
	p.events = append(p.events, event)
	return nil
}

func (p *fakePublisher) Events() []port.Event {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]port.Event, len(p.events))
	copy(out, p.events)
	return out
}

type fakeRepo struct {
	mu       sync.Mutex
	rooms    map[int]domain.Room
	bookings map[int]domain.Booking
	nextID   int

	listFn         func(ctx context.Context) ([]domain.Room, error)
	getByIDFn      func(ctx context.Context, id int) (domain.Room, error)
	listByRoomFn   func(ctx context.Context, roomID int) ([]domain.Booking, error)
	createFn       func(ctx context.Context, b domain.Booking) (domain.Booking, error)
	deleteByIDFn   func(ctx context.Context, id int) error
	createCalls    int
	createReceived []domain.Booking
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		rooms:    map[int]domain.Room{},
		bookings: map[int]domain.Booking{},
		nextID:   1,
	}
}

func (f *fakeRepo) List(ctx context.Context) ([]domain.Room, error) {
	if f.listFn != nil {
		return f.listFn(ctx)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	rooms := make([]domain.Room, 0, len(f.rooms))
	for _, r := range f.rooms {
		rooms = append(rooms, r)
	}
	sort.Slice(rooms, func(i, j int) bool { return rooms[i].ID < rooms[j].ID })

	return rooms, nil
}

func (f *fakeRepo) GetByID(ctx context.Context, id int) (domain.Room, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	r, ok := f.rooms[id]
	if !ok {
		return domain.Room{}, domain.ErrRoomNotFound
	}

	return r, nil
}

func (f *fakeRepo) ListByRoomID(ctx context.Context, roomID int) ([]domain.Booking, error) {
	if f.listByRoomFn != nil {
		return f.listByRoomFn(ctx, roomID)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]domain.Booking, 0)
	for _, b := range f.bookings {
		if b.RoomID == roomID {
			out = append(out, b)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartTime.Before(out[j].StartTime) })

	return out, nil
}

func (f *fakeRepo) Create(ctx context.Context, b domain.Booking) (domain.Booking, error) {
	if f.createFn != nil {
		return f.createFn(ctx, b)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.createCalls++
	f.createReceived = append(f.createReceived, b)
	b.ID = f.nextID
	f.nextID++
	f.bookings[b.ID] = b

	return b, nil
}

func (f *fakeRepo) DeleteByID(ctx context.Context, id int) error {
	if f.deleteByIDFn != nil {
		return f.deleteByIDFn(ctx, id)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.bookings[id]; !ok {
		return domain.ErrBookingNotFound
	}
	delete(f.bookings, id)

	return nil
}

func TestService_ListRooms_delegatesToRepository(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo()
	repo.rooms[1] = domain.Room{ID: 1, Name: "A"}
	repo.rooms[2] = domain.Room{ID: 2, Name: "B"}

	svc := service.New(repo, repo, nil)

	got, err := svc.ListRooms(context.Background())
	if err != nil {
		t.Fatalf("ListRooms err = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListRooms returned %d rooms, want 2", len(got))
	}
}

func TestService_GetRoomSchedule(t *testing.T) {
	t.Parallel()

	t.Run("returns ErrRoomNotFound when room is missing", func(t *testing.T) {
		t.Parallel()

		repo := newFakeRepo()
		svc := service.New(repo, repo, nil)

		_, err := svc.GetRoomSchedule(context.Background(), 42)
		if !errors.Is(err, domain.ErrRoomNotFound) {
			t.Fatalf("err = %v, want ErrRoomNotFound", err)
		}
	})

	t.Run("maps bookings to schedule items with type=booking", func(t *testing.T) {
		t.Parallel()

		repo := newFakeRepo()
		repo.rooms[1] = domain.Room{ID: 1, Name: "A"}
		start := time.Date(2026, time.April, 17, 9, 0, 0, 0, time.UTC)
		repo.bookings[10] = domain.Booking{
			ID:        10,
			RoomID:    1,
			StartTime: start,
			EndTime:   start.Add(time.Hour),
		}
		svc := service.New(repo, repo, nil)

		got, err := svc.GetRoomSchedule(context.Background(), 1)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1", len(got))
		}
		if got[0].Type != "booking" {
			t.Fatalf("type = %q, want %q", got[0].Type, "booking")
		}
		if !got[0].StartTime.Equal(start) {
			t.Fatalf("start = %v, want %v", got[0].StartTime, start)
		}
	})

	t.Run("returns empty slice for room with no bookings", func(t *testing.T) {
		t.Parallel()

		repo := newFakeRepo()
		repo.rooms[1] = domain.Room{ID: 1, Name: "A"}
		svc := service.New(repo, repo, nil)

		got, err := svc.GetRoomSchedule(context.Background(), 1)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if got == nil {
			t.Fatal("expected non-nil empty slice")
		}
		if len(got) != 0 {
			t.Fatalf("len = %d, want 0", len(got))
		}
	})
}

func TestService_CreateBooking(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, time.April, 17, 9, 0, 0, 0, time.UTC)

	t.Run("rejects invalid time range without touching repository", func(t *testing.T) {
		t.Parallel()

		repo := newFakeRepo()
		repo.rooms[1] = domain.Room{ID: 1}
		svc := service.New(repo, repo, nil)

		_, err := svc.CreateBooking(context.Background(), service.CreateBookingInput{
			RoomID:    1,
			StartTime: base,
			EndTime:   base,
		})
		if !errors.Is(err, domain.ErrInvalidTimeRange) {
			t.Fatalf("err = %v, want ErrInvalidTimeRange", err)
		}
		if repo.createCalls != 0 {
			t.Fatalf("Create called %d times, want 0", repo.createCalls)
		}
	})

	t.Run("returns ErrRoomNotFound when room is missing", func(t *testing.T) {
		t.Parallel()

		repo := newFakeRepo()
		svc := service.New(repo, repo, nil)

		_, err := svc.CreateBooking(context.Background(), service.CreateBookingInput{
			RoomID:    99,
			StartTime: base,
			EndTime:   base.Add(time.Hour),
		})
		if !errors.Is(err, domain.ErrRoomNotFound) {
			t.Fatalf("err = %v, want ErrRoomNotFound", err)
		}
	})

	t.Run("returns ErrScheduleConflict on overlap", func(t *testing.T) {
		t.Parallel()

		repo := newFakeRepo()
		repo.rooms[1] = domain.Room{ID: 1}
		repo.bookings[1] = domain.Booking{
			ID:        1,
			RoomID:    1,
			StartTime: base,
			EndTime:   base.Add(time.Hour),
		}
		svc := service.New(repo, repo, nil)

		_, err := svc.CreateBooking(context.Background(), service.CreateBookingInput{
			RoomID:    1,
			StartTime: base.Add(30 * time.Minute),
			EndTime:   base.Add(90 * time.Minute),
		})
		if !errors.Is(err, domain.ErrScheduleConflict) {
			t.Fatalf("err = %v, want ErrScheduleConflict", err)
		}
		if repo.createCalls != 0 {
			t.Fatalf("Create called %d times, want 0", repo.createCalls)
		}
	})

	t.Run("allows boundary-touching booking", func(t *testing.T) {
		t.Parallel()

		repo := newFakeRepo()
		repo.rooms[1] = domain.Room{ID: 1}
		repo.bookings[1] = domain.Booking{
			ID:        1,
			RoomID:    1,
			StartTime: base,
			EndTime:   base.Add(time.Hour),
		}
		svc := service.New(repo, repo, nil)

		got, err := svc.CreateBooking(context.Background(), service.CreateBookingInput{
			RoomID:    1,
			StartTime: base.Add(time.Hour),
			EndTime:   base.Add(2 * time.Hour),
		})
		if err != nil {
			t.Fatalf("unexpected err = %v", err)
		}
		if got.ID == 0 {
			t.Fatal("expected non-zero ID on created booking")
		}
	})

	t.Run("persists booking on success", func(t *testing.T) {
		t.Parallel()

		repo := newFakeRepo()
		repo.rooms[1] = domain.Room{ID: 1}
		svc := service.New(repo, repo, nil)

		input := service.CreateBookingInput{
			RoomID:    1,
			StartTime: base,
			EndTime:   base.Add(time.Hour),
		}

		got, err := svc.CreateBooking(context.Background(), input)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if got.RoomID != 1 || !got.StartTime.Equal(input.StartTime) || !got.EndTime.Equal(input.EndTime) {
			t.Fatalf("unexpected booking returned: %+v", got)
		}
		if repo.createCalls != 1 {
			t.Fatalf("Create called %d times, want 1", repo.createCalls)
		}
	})
}

func TestService_CancelBooking(t *testing.T) {
	t.Parallel()

	t.Run("forwards ErrBookingNotFound", func(t *testing.T) {
		t.Parallel()

		repo := newFakeRepo()
		svc := service.New(repo, repo, nil)

		err := svc.CancelBooking(context.Background(), 1)
		if !errors.Is(err, domain.ErrBookingNotFound) {
			t.Fatalf("err = %v, want ErrBookingNotFound", err)
		}
	})

	t.Run("deletes booking on success", func(t *testing.T) {
		t.Parallel()

		repo := newFakeRepo()
		repo.bookings[1] = domain.Booking{ID: 1, RoomID: 1}
		svc := service.New(repo, repo, nil)

		if err := svc.CancelBooking(context.Background(), 1); err != nil {
			t.Fatalf("err = %v", err)
		}
		if _, ok := repo.bookings[1]; ok {
			t.Fatal("booking 1 still present after CancelBooking")
		}
	})
}

func TestService_PublishesEvents(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, time.April, 17, 9, 0, 0, 0, time.UTC)

	t.Run("CreateBooking publishes booking.created on success", func(t *testing.T) {
		t.Parallel()

		repo := newFakeRepo()
		repo.rooms[1] = domain.Room{ID: 1}
		pub := &fakePublisher{}
		svc := service.New(repo, repo, pub)

		got, err := svc.CreateBooking(context.Background(), service.CreateBookingInput{
			RoomID:    1,
			StartTime: base,
			EndTime:   base.Add(time.Hour),
		})
		if err != nil {
			t.Fatalf("err = %v", err)
		}

		events := pub.Events()
		if len(events) != 1 {
			t.Fatalf("events = %d, want 1", len(events))
		}
		ev := events[0]
		if ev.Type != port.EventBookingCreated {
			t.Fatalf("type = %q, want %q", ev.Type, port.EventBookingCreated)
		}
		if ev.BookingID != got.ID || ev.RoomID != got.RoomID {
			t.Fatalf("event ids mismatch: %+v vs booking %+v", ev, got)
		}
		if !ev.StartTime.Equal(got.StartTime) || !ev.EndTime.Equal(got.EndTime) {
			t.Fatalf("event times mismatch: %+v vs %+v", ev, got)
		}
		if ev.OccurredAt.IsZero() {
			t.Fatal("OccurredAt should be set")
		}
	})

	t.Run("CreateBooking does not publish on conflict", func(t *testing.T) {
		t.Parallel()

		repo := newFakeRepo()
		repo.rooms[1] = domain.Room{ID: 1}
		repo.bookings[1] = domain.Booking{ID: 1, RoomID: 1, StartTime: base, EndTime: base.Add(time.Hour)}
		pub := &fakePublisher{}
		svc := service.New(repo, repo, pub)

		_, err := svc.CreateBooking(context.Background(), service.CreateBookingInput{
			RoomID:    1,
			StartTime: base.Add(30 * time.Minute),
			EndTime:   base.Add(90 * time.Minute),
		})
		if !errors.Is(err, domain.ErrScheduleConflict) {
			t.Fatalf("err = %v, want ErrScheduleConflict", err)
		}
		if len(pub.Events()) != 0 {
			t.Fatalf("expected no events, got %d", len(pub.Events()))
		}
	})

	t.Run("CreateBooking publish error does not fail HTTP success", func(t *testing.T) {
		t.Parallel()

		repo := newFakeRepo()
		repo.rooms[1] = domain.Room{ID: 1}
		pub := &fakePublisher{errOn: port.EventBookingCreated, failWith: errors.New("broker down")}
		svc := service.New(repo, repo, pub)

		got, err := svc.CreateBooking(context.Background(), service.CreateBookingInput{
			RoomID:    1,
			StartTime: base,
			EndTime:   base.Add(time.Hour),
		})
		if err != nil {
			t.Fatalf("CreateBooking err = %v, want success despite broker failure", err)
		}
		if got.ID == 0 {
			t.Fatal("expected created booking")
		}
	})

	t.Run("CancelBooking publishes booking.cancelled on success", func(t *testing.T) {
		t.Parallel()

		repo := newFakeRepo()
		repo.bookings[7] = domain.Booking{ID: 7, RoomID: 2}
		pub := &fakePublisher{}
		svc := service.New(repo, repo, pub)

		if err := svc.CancelBooking(context.Background(), 7); err != nil {
			t.Fatalf("err = %v", err)
		}

		events := pub.Events()
		if len(events) != 1 || events[0].Type != port.EventBookingCancelled || events[0].BookingID != 7 {
			t.Fatalf("unexpected events: %+v", events)
		}
	})

	t.Run("CancelBooking does not publish when booking missing", func(t *testing.T) {
		t.Parallel()

		repo := newFakeRepo()
		pub := &fakePublisher{}
		svc := service.New(repo, repo, pub)

		_ = svc.CancelBooking(context.Background(), 999)

		if len(pub.Events()) != 0 {
			t.Fatalf("expected no events, got %d", len(pub.Events()))
		}
	})
}
