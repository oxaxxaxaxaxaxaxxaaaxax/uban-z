//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	bookingpostgres "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/postgres"
	bookingdomain "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/domain"
	parserdomain "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/parser/domain"
)

func TestPostgresStore_ReplaceParsedSchedule_importsAndReplacesParserRows(t *testing.T) {
	pool := bootPostgres(t)
	store := bookingpostgres.NewStoreFromPool(pool)
	ctx := context.Background()

	hasSchedule, err := store.HasParsedSchedule(ctx)
	if err != nil {
		t.Fatalf("HasParsedSchedule before import: %v", err)
	}
	if hasSchedule {
		t.Fatal("HasParsedSchedule before import = true, want false")
	}

	start := time.Date(2026, time.September, 7, 9, 0, 0, 0, time.UTC)
	rooms := []parserdomain.RoomSelector{{Name: "3107 (Новый корпус)", FullURL: "https://table.nsu.ru/room/3107"}}
	slots := []parserdomain.ScheduleSlot{
		{
			RoomName:     "3107",
			Building:     "Новый корпус",
			Capacity:     30,
			Subject:      "Мат анализ",
			LessonType:   "Лекция",
			Teacher:      "Иванов И.И.",
			GroupNumbers: []string{"22201", "22202"},
			Week:         "числ.",
			StartTime:    start,
			EndTime:      start.Add(95 * time.Minute),
		},
		{
			RoomName:     "3107",
			Building:     "Новый корпус",
			Capacity:     30,
			Subject:      "Физика",
			LessonType:   "Практика",
			Teacher:      "Петров П.П.",
			GroupNumbers: []string{"22203"},
			StartTime:    start.Add(2 * time.Hour),
			EndTime:      start.Add(2*time.Hour + 95*time.Minute),
		},
	}

	stats, err := store.ReplaceParsedSchedule(ctx, rooms, slots)
	if err != nil {
		t.Fatalf("ReplaceParsedSchedule: %v", err)
	}
	if stats.RoomsImported != 1 || stats.LessonsImported != 2 || stats.LessonsSkipped != 0 {
		t.Fatalf("stats = %+v", stats)
	}
	hasSchedule, err = store.HasParsedSchedule(ctx)
	if err != nil {
		t.Fatalf("HasParsedSchedule after import: %v", err)
	}
	if !hasSchedule {
		t.Fatal("HasParsedSchedule after import = false, want true")
	}

	roomID := findRoomID(t, store, "3107", "Новый корпус")
	roomsAfterImport, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List rooms after import: %v", err)
	}
	for _, room := range roomsAfterImport {
		if room.Name == "3107 (Новый корпус)" {
			t.Fatalf("raw parser room was stored separately: %+v", room)
		}
		if room.ID == roomID && room.Capacity != 30 {
			t.Fatalf("imported room capacity = %d, want 30", room.Capacity)
		}
	}

	got, err := store.ListByRoomID(ctx, roomID)
	if err != nil {
		t.Fatalf("ListByRoomID: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Subject != "Мат анализ" || got[0].LessonType != "Лекция" || got[0].Teacher != "Иванов И.И." {
		t.Fatalf("first imported booking = %+v", got[0])
	}
	if got[0].CreatorRole != bookingdomain.RoleAdmin || got[0].UserID != 0 {
		t.Fatalf("parser marker = role %q user %d", got[0].CreatorRole, got[0].UserID)
	}
	if len(got[0].GroupNumbers) != 2 || got[0].GroupNumbers[0] != "22201" {
		t.Fatalf("group_numbers = %v", got[0].GroupNumbers)
	}

	replacement := []parserdomain.ScheduleSlot{{
		RoomName:   "3107",
		Building:   "Новый корпус",
		Capacity:   30,
		Subject:    "Алгебра",
		LessonType: "Семинар",
		StartTime:  start.Add(24 * time.Hour),
		EndTime:    start.Add(24*time.Hour + 95*time.Minute),
	}}
	if _, err := store.ReplaceParsedSchedule(ctx, rooms, replacement); err != nil {
		t.Fatalf("ReplaceParsedSchedule replacement: %v", err)
	}

	got, err = store.ListByRoomID(ctx, roomID)
	if err != nil {
		t.Fatalf("ListByRoomID after replace: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len after replace = %d, want 1", len(got))
	}
	if got[0].Subject != "Алгебра" {
		t.Fatalf("subject after replace = %q", got[0].Subject)
	}
}

func TestPostgresStore_ReplaceParsedSchedule_skipsRowsConflictingWithUserBookings(t *testing.T) {
	pool := bootPostgres(t)
	store := bookingpostgres.NewStoreFromPool(pool)
	ctx := context.Background()

	rooms := []parserdomain.RoomSelector{{Name: "3108", FullURL: "https://table.nsu.ru/room/3108"}}
	if _, err := store.ReplaceParsedSchedule(ctx, rooms, nil); err != nil {
		t.Fatalf("ReplaceParsedSchedule rooms: %v", err)
	}
	roomID := findRoomID(t, store, "3108", "НГУ")

	start := time.Date(2026, time.September, 8, 9, 0, 0, 0, time.UTC)
	if _, err := store.Create(ctx, bookingdomain.Booking{
		RoomID:      roomID,
		UserID:      42,
		CreatorRole: bookingdomain.RoleStudentB,
		StartTime:   start,
		EndTime:     start.Add(95 * time.Minute),
	}); err != nil {
		t.Fatalf("Create user booking: %v", err)
	}

	stats, err := store.ReplaceParsedSchedule(ctx, rooms, []parserdomain.ScheduleSlot{{
		RoomName:   "3108",
		Building:   "НГУ",
		Capacity:   30,
		Subject:    "Конфликтующее занятие",
		LessonType: "Лекция",
		StartTime:  start,
		EndTime:    start.Add(95 * time.Minute),
	}})
	if err != nil {
		t.Fatalf("ReplaceParsedSchedule conflicting: %v", err)
	}
	if stats.LessonsImported != 0 || stats.LessonsSkipped != 1 {
		t.Fatalf("stats = %+v, want skipped conflict", stats)
	}

	got, err := store.ListByRoomID(ctx, roomID)
	if err != nil {
		t.Fatalf("ListByRoomID: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want only existing user booking", len(got))
	}
	if got[0].UserID != 42 || got[0].Subject != "" {
		t.Fatalf("booking after conflict import = %+v", got[0])
	}
}

func findRoomID(t *testing.T, store *bookingpostgres.Store, name string, building string) int {
	t.Helper()

	rooms, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List rooms: %v", err)
	}
	for _, room := range rooms {
		if room.Name == name && room.Building == building {
			return room.ID
		}
	}
	t.Fatalf("room %q/%q not found in %v", building, name, rooms)
	return 0
}
