//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	bookingpostgres "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/postgres"
	parserdomain "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/parser/domain"
)

func TestScheduleImportLockConcurrentStartupImportsOnce(t *testing.T) {
	pool := bootPostgres(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var imports atomic.Int32
	results := make(chan error, 6)
	start := make(chan struct{})
	for i := 0; i < cap(results); i++ {
		go func() {
			<-start
			store := bookingpostgres.NewStoreFromPool(pool)
			results <- store.WithScheduleImportLock(ctx, func() error {
				exists, err := store.HasParsedSchedule(ctx)
				if err != nil || exists {
					return err
				}
				imports.Add(1)
				startTime := time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC)
				_, err = store.ReplaceParsedSchedule(ctx, nil, []parserdomain.ScheduleSlot{{
					RoomName: "3107", Building: "НГУ", Capacity: 30,
					StartTime: startTime, EndTime: startTime.Add(95 * time.Minute),
				}})
				return err
			})
		}()
	}
	close(start)
	for i := 0; i < cap(results); i++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	if got := imports.Load(); got != 1 {
		t.Fatalf("imports = %d, want 1", got)
	}
}

func TestScheduleImportLockReleasedAfterFailure(t *testing.T) {
	store := bookingpostgres.NewStoreFromPool(bootPostgres(t))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	want := errors.New("parser failed")
	if err := store.WithScheduleImportLock(ctx, func() error { return want }); !errors.Is(err, want) {
		t.Fatalf("error = %v, want parser failure", err)
	}
	called := false
	if err := store.WithScheduleImportLock(ctx, func() error { called = true; return nil }); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("lock was not released")
	}
}

func TestScheduleImportLockWaitingReplicaCanBeCanceled(t *testing.T) {
	store := bookingpostgres.NewStoreFromPool(bootPostgres(t))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	locked := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	done := make(chan error, 1)
	go func() {
		done <- store.WithScheduleImportLock(ctx, func() error {
			close(locked)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	select {
	case <-locked:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	waitCtx, stopWaiting := context.WithTimeout(ctx, 100*time.Millisecond)
	defer stopWaiting()
	if err := store.WithScheduleImportLock(waitCtx, func() error {
		t.Error("waiting replica must not enter import")
		return nil
	}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want waiting canceled", err)
	}
	release <- struct{}{}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestScheduleImportLockReleasedWhenConnectionDies(t *testing.T) {
	pool := bootPostgres(t)
	store := bookingpostgres.NewStoreFromPool(pool)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	locked := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	done := make(chan error, 1)
	go func() {
		done <- store.WithScheduleImportLock(ctx, func() error {
			close(locked)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	select {
	case <-locked:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	var terminated bool
	err := pool.QueryRow(ctx, `
		SELECT pg_terminate_backend(pid)
		FROM pg_locks
		WHERE locktype = 'advisory' AND database = (SELECT oid FROM pg_database WHERE datname = current_database())
		AND granted
	`).Scan(&terminated)
	if err != nil || !terminated {
		t.Fatalf("terminate lock connection: terminated=%v, err=%v", terminated, err)
	}
	if err := store.WithScheduleImportLock(ctx, func() error { return nil }); err != nil {
		t.Fatalf("lock after connection loss: %v", err)
	}
	release <- struct{}{}
	if err := <-done; err == nil {
		t.Fatal("lost connection must fail the original transaction")
	}
}
