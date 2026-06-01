package domain

import "time"

type RoomSelector struct {
	Name     string
	Building string
	Capacity int
	FullURL  string
}

type LessonRoom struct {
	Subject      string
	LessonType   string
	Teacher      string
	StartTime    string
	Weekday      string
	Room         string
	Week         string
	GroupNumbers []string
}

type ScheduleSlot struct {
	RoomName     string
	Building     string
	Capacity     int
	Subject      string
	LessonType   string
	Teacher      string
	GroupNumbers []string
	Week         string
	StartTime    time.Time
	EndTime      time.Time
}

type ImportStats struct {
	RoomsSeen       int
	RoomsImported   int
	LessonsSeen     int
	LessonsExpanded int
	LessonsImported int
	LessonsSkipped  int
}
