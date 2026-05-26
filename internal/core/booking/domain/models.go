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
//
// Teacher and GroupNumbers carry class-schedule metadata for parser-imported
// rows and are zero-valued for user-created bookings. They are not exposed on
// the Booking HTTP response — only on the ScheduleItem projection.
type Booking struct {
	ID           int
	RoomID       int
	UserID       int
	CreatorRole  Role
	StartTime    time.Time
	EndTime      time.Time
	Teacher      string
	GroupNumbers []string
}

// ScheduleItem describes an occupied room interval returned by schedule queries.
//
// Teacher and GroupNumbers are populated only for parser-imported class rows;
// user-created bookings leave them zero-valued.
type ScheduleItem struct {
	StartTime    time.Time
	EndTime      time.Time
	Type         string
	Teacher      string
	GroupNumbers []string
}
