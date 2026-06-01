package service_test

import (
	"context"
	"testing"
	"time"

	parserdomain "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/parser/domain"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/parser/service"
)

type fakeSource struct {
	rooms   []parserdomain.RoomSelector
	lessons map[string][]parserdomain.LessonRoom
}

func (f fakeSource) ParseRooms(context.Context) ([]parserdomain.RoomSelector, error) {
	return f.rooms, nil
}

func (f fakeSource) ParseLessonsRoom(_ context.Context, roomURL string) ([]parserdomain.LessonRoom, error) {
	return f.lessons[roomURL], nil
}

type captureRepo struct {
	rooms []parserdomain.RoomSelector
	slots []parserdomain.ScheduleSlot
}

func (r *captureRepo) ReplaceParsedSchedule(_ context.Context, rooms []parserdomain.RoomSelector, slots []parserdomain.ScheduleSlot) (parserdomain.ImportStats, error) {
	r.rooms = append([]parserdomain.RoomSelector(nil), rooms...)
	r.slots = append([]parserdomain.ScheduleSlot(nil), slots...)
	return parserdomain.ImportStats{
		RoomsImported:   len(rooms),
		LessonsImported: len(slots),
	}, nil
}

func TestService_RunExpandsRoomLessons(t *testing.T) {
	t.Parallel()

	location := time.FixedZone("NOVT", 7*60*60)
	repo := &captureRepo{}
	source := fakeSource{
		rooms: []parserdomain.RoomSelector{{Name: "3107 (Новый корпус)", FullURL: "room-3107"}},
		lessons: map[string][]parserdomain.LessonRoom{
			"room-3107": {
				{
					Subject:      "Мат анализ",
					LessonType:   "Лекция",
					Teacher:      "Иванов И.И.",
					StartTime:    "09:00",
					Weekday:      "Понедельник",
					Week:         "",
					GroupNumbers: []string{"22201"},
				},
			},
		},
	}

	parser, err := service.New(source, repo, service.Config{
		WeeksAhead:      2,
		DefaultBuilding: "НГУ",
		DefaultCapacity: 30,
		Location:        location,
		Now: func() time.Time {
			return time.Date(2026, time.September, 2, 12, 0, 0, 0, location)
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	stats, err := parser.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if stats.RoomsSeen != 1 || stats.LessonsSeen != 1 || stats.LessonsExpanded != 2 || stats.LessonsImported != 2 {
		t.Fatalf("stats = %+v", stats)
	}
	if len(repo.slots) != 2 {
		t.Fatalf("slots len = %d, want 2", len(repo.slots))
	}

	first := repo.slots[0]
	if first.RoomName != "3107" || first.Building != "Новый корпус" {
		t.Fatalf("room fields = %+v", first)
	}
	if first.Capacity != 30 {
		t.Fatalf("capacity = %d, want 30", first.Capacity)
	}
	if first.Subject != "Мат анализ" || first.LessonType != "Лекция" || first.Teacher != "Иванов И.И." {
		t.Fatalf("lesson fields = %+v", first)
	}
	if first.StartTime.Format(time.RFC3339) != "2026-08-31T09:00:00+07:00" {
		t.Fatalf("start = %s", first.StartTime.Format(time.RFC3339))
	}
	if first.EndTime.Sub(first.StartTime) != 95*time.Minute {
		t.Fatalf("duration = %s, want 95m", first.EndTime.Sub(first.StartTime))
	}
}
