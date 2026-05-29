package port

import (
	"context"
	"time"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/domain"
)

const (
	EventBookingCreated   = "booking.created"
	EventBookingCancelled = "booking.cancelled"
)

// Actor identifies who performed the action that produced an event.
type Actor struct {
	UserID int         `json:"user_id"`
	Login  string      `json:"login"`
	Role   domain.Role `json:"role"`
}

// Event is a booking-lifecycle event emitted to downstream consumers.
//
// Created events carry the creator (OwnerID/OwnerRole). Cancelled events
// carry both the owner and the actor who cancelled (CancelledBy), plus a
// SelfCancel flag so notification consumers can suppress messages to users
// cancelling their own bookings.
type Event struct {
	Type        string      `json:"type"`
	BookingID   int         `json:"booking_id"`
	RoomID      int         `json:"room_id"`
	OwnerID     int         `json:"owner_id"`
	OwnerRole   domain.Role `json:"owner_role"`
	StartTime   time.Time   `json:"start_time"`
	EndTime     time.Time   `json:"end_time"`
	CancelledBy *Actor      `json:"cancelled_by,omitempty"`
	SelfCancel  bool        `json:"self_cancel,omitempty"`
	OccurredAt  time.Time   `json:"occurred_at"`
}

// EventPublisher emits booking-lifecycle events to a broker.
// Implementations should treat transport errors as observable but non-fatal;
// the booking-service is the source of truth, the broker is best-effort.
type EventPublisher interface {
	Publish(ctx context.Context, event Event) error
}
