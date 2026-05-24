//go:build integration

package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	bookingpostgres "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/postgres"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/domain"
)

// migrationsDir is resolved relative to this test file.
var migrationsDir = filepath.Join("..", "..", "..", "..", "migrations", "booking")

func bootPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	container, err := tcpostgres.Run(ctx,
		"postgres:17-alpine",
		tcpostgres.WithDatabase("test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("terminate container: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	// Run goose against a database/sql handle, then close it. Tests use pgxpool directly.
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("goose dialect: %v", err)
	}
	if err := goose.UpContext(ctx, db, migrationsDir); err != nil {
		_ = db.Close()
		t.Fatalf("goose up: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close sql.DB: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)

	return pool
}

func TestPostgresStore_ListRooms_returnsSeeded(t *testing.T) {
	pool := bootPostgres(t)
	store := bookingpostgres.NewStoreFromPool(pool)

	rooms, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rooms) != 3 {
		t.Fatalf("got %d rooms, want 3 (seeded)", len(rooms))
	}
	names := map[string]bool{}
	for _, r := range rooms {
		names[r.Name] = true
	}
	for _, want := range []string{"A101", "B204", "C305"} {
		if !names[want] {
			t.Errorf("missing seed room %q", want)
		}
	}
}

func TestPostgresStore_GetByID_missingRoom(t *testing.T) {
	pool := bootPostgres(t)
	store := bookingpostgres.NewStoreFromPool(pool)

	_, err := store.GetByID(context.Background(), 999_999)
	if !errors.Is(err, domain.ErrRoomNotFound) {
		t.Fatalf("err = %v, want ErrRoomNotFound", err)
	}
}

func TestPostgresStore_CreateBooking_persistsAndConflicts(t *testing.T) {
	pool := bootPostgres(t)
	store := bookingpostgres.NewStoreFromPool(pool)
	ctx := context.Background()

	base := time.Date(2026, time.September, 1, 9, 0, 0, 0, time.UTC)

	created, err := store.Create(ctx, domain.Booking{
		RoomID:    1,
		StartTime: base,
		EndTime:   base.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	// Overlap on same room must be rejected.
	_, err = store.Create(ctx, domain.Booking{
		RoomID:    1,
		StartTime: base.Add(30 * time.Minute),
		EndTime:   base.Add(90 * time.Minute),
	})
	if !errors.Is(err, domain.ErrScheduleConflict) {
		t.Fatalf("overlap err = %v, want ErrScheduleConflict", err)
	}

	// Boundary-touch (existing ends at 10:00, new starts at 10:00) must succeed.
	if _, err := store.Create(ctx, domain.Booking{
		RoomID:    1,
		StartTime: base.Add(time.Hour),
		EndTime:   base.Add(2 * time.Hour),
	}); err != nil {
		t.Fatalf("boundary Create: %v", err)
	}

	// Different room, same time must succeed.
	if _, err := store.Create(ctx, domain.Booking{
		RoomID:    2,
		StartTime: base,
		EndTime:   base.Add(time.Hour),
	}); err != nil {
		t.Fatalf("other-room Create: %v", err)
	}
}

func TestPostgresStore_CreateBooking_rejectsInvalidRange(t *testing.T) {
	pool := bootPostgres(t)
	store := bookingpostgres.NewStoreFromPool(pool)

	start := time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC)
	_, err := store.Create(context.Background(), domain.Booking{
		RoomID:    1,
		StartTime: start,
		EndTime:   start.Add(-time.Hour),
	})
	if !errors.Is(err, domain.ErrInvalidTimeRange) {
		t.Fatalf("err = %v, want ErrInvalidTimeRange", err)
	}
}

func TestPostgresStore_DeleteByID(t *testing.T) {
	pool := bootPostgres(t)
	store := bookingpostgres.NewStoreFromPool(pool)
	ctx := context.Background()

	start := time.Date(2026, time.October, 1, 12, 0, 0, 0, time.UTC)
	created, err := store.Create(ctx, domain.Booking{
		RoomID:    1,
		StartTime: start,
		EndTime:   start.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.DeleteByID(ctx, created.ID); err != nil {
		t.Fatalf("DeleteByID first: %v", err)
	}

	if err := store.DeleteByID(ctx, created.ID); !errors.Is(err, domain.ErrBookingNotFound) {
		t.Fatalf("second DeleteByID = %v, want ErrBookingNotFound", err)
	}
}

func TestPostgresStore_ParallelInserts_exactlyOneWins(t *testing.T) {
	pool := bootPostgres(t)
	store := bookingpostgres.NewStoreFromPool(pool)
	ctx := context.Background()

	start := time.Date(2026, time.November, 1, 9, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	const attempts = 30

	var (
		wg        sync.WaitGroup
		successes int64
		conflicts int64
	)
	wg.Add(attempts)
	for i := 0; i < attempts; i++ {
		go func() {
			defer wg.Done()
			_, err := store.Create(ctx, domain.Booking{
				RoomID:    3,
				StartTime: start,
				EndTime:   end,
			})
			switch {
			case err == nil:
				atomic.AddInt64(&successes, 1)
			case errors.Is(err, domain.ErrScheduleConflict):
				atomic.AddInt64(&conflicts, 1)
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	if successes != 1 {
		t.Fatalf("successes = %d, want 1 (GiST constraint should serialize)", successes)
	}
	if conflicts != attempts-1 {
		t.Fatalf("conflicts = %d, want %d", conflicts, attempts-1)
	}

	got, err := store.ListByRoomID(ctx, 3)
	if err != nil {
		t.Fatalf("ListByRoomID: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("persisted = %d, want 1", len(got))
	}
}
