package port

import (
	"context"

	parserdomain "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/parser/domain"
)

type Source interface {
	ParseRooms(ctx context.Context) ([]parserdomain.RoomSelector, error)
	ParseLessonsRoom(ctx context.Context, roomURL string) ([]parserdomain.LessonRoom, error)
}

type ScheduleRepository interface {
	ReplaceParsedSchedule(ctx context.Context, rooms []parserdomain.RoomSelector, slots []parserdomain.ScheduleSlot) (parserdomain.ImportStats, error)
}
