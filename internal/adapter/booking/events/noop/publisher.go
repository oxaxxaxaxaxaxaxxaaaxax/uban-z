package noop

import (
	"context"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/port"
)

// Publisher discards every event. Used when EVENTS_ENABLED is false.
type Publisher struct{}

func (Publisher) Publish(_ context.Context, _ port.Event) error {
	return nil
}
