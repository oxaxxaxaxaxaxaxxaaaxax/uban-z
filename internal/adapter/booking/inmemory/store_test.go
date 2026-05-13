package inmemory_test

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/inmemory"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/domain"
)

func TestStore_List_returnsSeedRoomsSortedByID(t *testing.T) {
	t.Parallel()

	store := inmemory.NewStore()

	rooms, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List err = %v", err)
	}
	if len(rooms) == 0 {
		t.Fatal("expected at least one seeded room")
	}
	for i := 1; i < len(rooms); i++ {
		if rooms[i-1].ID > rooms[i].ID {
			t.Fatalf("rooms not sorted by ID at index %d: %+v", i, rooms)
		}
	}
}

func TestStore_GetByID(t *testing.T) {
	t.Parallel()

	store := inmemory.NewStore()

	t.Run("returns seeded room", func(t *testing.T) {
		t.Parallel()
		room, err := store.GetByID(context.Background(), 1)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if room.ID != 1 {
			t.Fatalf("ID = %d, want 1", room.ID)
		}
	})

	t.Run("returns ErrRoomNotFound for missing ID", func(t *testing.T) {
		t.Parallel()
		_, err := store.GetByID(context.Background(), 999_999)
		if !errors.Is(err, domain.ErrRoomNotFound) {
			t.Fatalf("err = %v, want ErrRoomNotFound", err)
		}
	})
}

func TestStore_ListByRoomID_isFilteredAndSorted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := inmemory.NewStore()

	base := time.Date(2026, time.May, 1, 9, 0, 0, 0, time.UTC)
	want := []time.Time{
		base.Add(2 * time.Hour),
		base.Add(4 * time.Hour),
		base.Add(6 * time.Hour),
	}

	for _, start := range []time.Time{want[2], want[0], want[1]} {
		if _, err := store.Create(ctx, domain.Booking{
			RoomID:    1,
			StartTime: start,
			EndTime:   start.Add(time.Hour),
		}); err != nil {
			t.Fatalf("Create err = %v", err)
		}
	}

	got, err := store.ListByRoomID(ctx, 1)
	if err != nil {
		t.Fatalf("ListByRoomID err = %v", err)
	}

	starts := make([]time.Time, 0, len(got))
	for _, b := range got {
		if b.RoomID != 1 {
			t.Fatalf("got booking for wrong room: %+v", b)
		}
		starts = append(starts, b.StartTime)
	}
	if !sort.SliceIsSorted(starts, func(i, j int) bool { return starts[i].Before(starts[j]) }) {
		t.Fatalf("bookings not sorted by start time: %v", starts)
	}
}

func TestStore_DeleteByID_returnsErrBookingNotFound(t *testing.T) {
	t.Parallel()

	store := inmemory.NewStore()

	err := store.DeleteByID(context.Background(), 999_999)
	if !errors.Is(err, domain.ErrBookingNotFound) {
		t.Fatalf("err = %v, want ErrBookingNotFound", err)
	}
}

func TestStore_Create_assignsUniqueIDsUnderConcurrency(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := inmemory.NewStore()

	const workers = 50
	const perWorker = 20

	base := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)

	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)
	ids := make(map[int]struct{}, workers*perWorker)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerIdx int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				start := base.Add(time.Duration(workerIdx*perWorker+i) * time.Hour)
				b, err := store.Create(ctx, domain.Booking{
					RoomID:    1 + (workerIdx % 3),
					StartTime: start,
					EndTime:   start.Add(time.Minute),
				})
				if err != nil {
					t.Errorf("Create err = %v", err)
					return
				}
				mu.Lock()
				if _, dup := ids[b.ID]; dup {
					t.Errorf("duplicate ID %d", b.ID)
				}
				ids[b.ID] = struct{}{}
				mu.Unlock()
			}
		}(w)
	}

	wg.Wait()

	if len(ids) != workers*perWorker {
		t.Fatalf("got %d unique IDs, want %d", len(ids), workers*perWorker)
	}
}
