package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/port"
)

const (
	exchangeKind = "topic"
)

// Publisher publishes booking-lifecycle events to a RabbitMQ topic exchange.
//
// Errors during publish are logged at WARN level (so events can be recovered
// from logs if the broker is down) but are not returned to the caller — the
// service treats event publication as best-effort.
type Publisher struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
	logger   *slog.Logger
}

func NewPublisher(url, exchange string, logger *slog.Logger) (*Publisher, error) {
	if logger == nil {
		logger = slog.Default()
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq channel: %w", err)
	}

	if err := channel.ExchangeDeclare(
		exchange,
		exchangeKind,
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,
	); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("declare exchange %q: %w", exchange, err)
	}

	if err := channel.Confirm(false); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("enable publisher confirms: %w", err)
	}

	return &Publisher{
		conn:     conn,
		channel:  channel,
		exchange: exchange,
		logger:   logger,
	}, nil
}

// Close releases the broker connection and channel.
func (p *Publisher) Close() error {
	var errs []error
	if p.channel != nil {
		if err := p.channel.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if p.conn != nil {
		if err := p.conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (p *Publisher) Publish(ctx context.Context, event port.Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		p.logger.Warn("event marshal failed",
			slog.String("event_type", event.Type),
			slog.Int("booking_id", event.BookingID),
			slog.Any("err", err),
		)
		return err
	}

	if err := p.channel.PublishWithContext(
		ctx,
		p.exchange,
		event.Type, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
			Timestamp:    event.OccurredAt,
		},
	); err != nil {
		p.logger.Warn("event publish failed",
			slog.String("event_type", event.Type),
			slog.Int("booking_id", event.BookingID),
			slog.Int("room_id", event.RoomID),
			slog.Any("err", err),
		)
		return err
	}

	return nil
}
