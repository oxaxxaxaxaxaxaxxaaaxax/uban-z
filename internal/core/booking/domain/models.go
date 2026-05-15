package domain

import "time"

// Room describes a bookable room inside the booking domain.
type Room struct {
	ID       int
	Name     string
	Capacity int
	Building string
}

// Booking represents a reserved time slot for a room.
type Booking struct {
	ID        int
	RoomID    int
	StartTime time.Time
	EndTime   time.Time
}

// ScheduleItem describes an occupied room interval returned by schedule queries.
type ScheduleItem struct {
	StartTime time.Time
	EndTime   time.Time
	Type      string
}
