//go:build integration

package rabbitmq_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/testcontainers/testcontainers-go"
	tcrabbitmq "github.com/testcontainers/testcontainers-go/modules/rabbitmq"
	"github.com/testcontainers/testcontainers-go/wait"

	bookingrmq "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/events/rabbitmq"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/port"
)

func bootRabbitMQ(t *testing.T) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	container, err := tcrabbitmq.Run(ctx,
		"rabbitmq:3-management-alpine",
		testcontainers.WithWaitStrategy(
			wait.ForLog("Server startup complete").WithStartupTimeout(2*time.Minute),
		),
	)
	if err != nil {
		t.Fatalf("rabbitmq container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("terminate rabbitmq: %v", err)
		}
	})

	url, err := container.AmqpURL(ctx)
	if err != nil {
		t.Fatalf("amqp url: %v", err)
	}
	return url
}

func TestRabbitMQ_PublishAndConsume(t *testing.T) {
	url := bootRabbitMQ(t)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	const exchange = "booking.events.test"
	pub, err := bookingrmq.NewPublisher(url, exchange, logger)
	if err != nil {
		t.Fatalf("NewPublisher: %v", err)
	}
	t.Cleanup(func() {
		if err := pub.Close(); err != nil {
			t.Logf("publisher close: %v", err)
		}
	})

	// Consumer: declare an exclusive queue bound to the exchange.
	conn, err := amqp.Dial(url)
	if err != nil {
		t.Fatalf("consumer dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("consumer channel: %v", err)
	}
	t.Cleanup(func() { _ = ch.Close() })

	queue, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		t.Fatalf("queue declare: %v", err)
	}
	if err := ch.QueueBind(queue.Name, "booking.*", exchange, false, nil); err != nil {
		t.Fatalf("queue bind: %v", err)
	}

	deliveries, err := ch.Consume(queue.Name, "", true, true, false, false, nil)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}

	// Publish a created event.
	occurred := time.Date(2026, time.September, 1, 9, 0, 0, 0, time.UTC)
	if err := pub.Publish(context.Background(), port.Event{
		Type:       port.EventBookingCreated,
		BookingID:  42,
		RoomID:     1,
		StartTime:  occurred,
		EndTime:    occurred.Add(time.Hour),
		OccurredAt: occurred,
	}); err != nil {
		t.Fatalf("publish created: %v", err)
	}

	// Publish a cancelled event.
	if err := pub.Publish(context.Background(), port.Event{
		Type:       port.EventBookingCancelled,
		BookingID:  42,
		OccurredAt: occurred.Add(time.Hour),
	}); err != nil {
		t.Fatalf("publish cancelled: %v", err)
	}

	// Consume both deliveries with a per-message timeout.
	for i, wantType := range []string{port.EventBookingCreated, port.EventBookingCancelled} {
		select {
		case d := <-deliveries:
			if d.RoutingKey != wantType {
				t.Fatalf("message %d routing key = %q, want %q", i, d.RoutingKey, wantType)
			}
			if d.ContentType != "application/json" {
				t.Fatalf("message %d content type = %q", i, d.ContentType)
			}
			var ev port.Event
			if err := json.Unmarshal(d.Body, &ev); err != nil {
				t.Fatalf("message %d unmarshal: %v; body=%s", i, err, d.Body)
			}
			if ev.Type != wantType {
				t.Fatalf("message %d event.Type = %q, want %q", i, ev.Type, wantType)
			}
			if ev.BookingID != 42 {
				t.Fatalf("message %d booking_id = %d, want 42", i, ev.BookingID)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for message %d (%q)", i, wantType)
		}
	}

	// No more messages expected.
	select {
	case d := <-deliveries:
		t.Fatalf("unexpected extra message: routingKey=%q body=%s", d.RoutingKey, d.Body)
	case <-time.After(500 * time.Millisecond):
		// good
	}
}
