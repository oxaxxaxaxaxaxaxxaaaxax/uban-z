package bookinghttp

import (
	"time"

	bookingserver "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/bookingserver"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/domain"
)

func mapRooms(rooms []domain.Room) []bookingserver.Room {
	response := make([]bookingserver.Room, 0, len(rooms))
	for _, room := range rooms {
		response = append(response, mapRoom(room))
	}

	return response
}

func mapRoom(room domain.Room) bookingserver.Room {
	return bookingserver.Room{
		Building: ptr(room.Building),
		Capacity: ptr(room.Capacity),
		Id:       ptr(room.ID),
		Name:     ptr(room.Name),
	}
}

func mapSchedule(schedule []domain.ScheduleItem) []bookingserver.ScheduleItem {
	response := make([]bookingserver.ScheduleItem, 0, len(schedule))
	for _, item := range schedule {
		response = append(response, bookingserver.ScheduleItem{
			EndTime:   ptr(item.EndTime),
			StartTime: ptr(item.StartTime),
			Type:      ptr(item.Type),
		})
	}

	return response
}

func mapBooking(booking domain.Booking) bookingserver.Booking {
	startTime := booking.StartTime.Format(time.RFC3339)
	endTime := booking.EndTime.Format(time.RFC3339)

	return bookingserver.Booking{
		EndTime:   ptr(endTime),
		Id:        ptr(booking.ID),
		RoomId:    ptr(booking.RoomID),
		StartTime: ptr(startTime),
	}
}

func ptr[T any](value T) *T {
	return &value
}
