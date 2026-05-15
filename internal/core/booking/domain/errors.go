package domain

import "errors"

var (
	ErrRoomNotFound     = errors.New("room not found")
	ErrBookingNotFound  = errors.New("booking not found")
	ErrInvalidTimeRange = errors.New("invalid time range")
	ErrScheduleConflict = errors.New("schedule conflict")
)
