package port

import (
	"context"
	"time"
)

const (
	EventBookingCreated   = "booking.created"
	EventBookingCancelled = "booking.cancelled"
)

// Event is a booking-lifecycle event emitted to downstream consumers.
type Event struct {
	Type       string    `json:"type"`
	BookingID  int       `json:"booking_id"`
	RoomID     int       `json:"room_id"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	OccurredAt time.Time `json:"occurred_at"`
}

// EventPublisher emits booking-lifecycle events to a broker.
// Implementations should treat transport errors as observable but non-fatal;
// the booking-service is the source of truth, the broker is best-effort.
type EventPublisher interface {
	Publish(ctx context.Context, event Event) error
}
