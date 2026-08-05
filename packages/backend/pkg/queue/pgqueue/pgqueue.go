// Package pgqueue wraps pgque-go for Dockzilla's job engine.
package pgqueue

import (
	"context"
	"dockzilla/pkg/domain"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/NikolayS/pgque-go"
	"github.com/uptrace/bun"
)

// Queue wraps pgque-go for Dockzilla's job engine. The zero value is not
// usable; build one with New.
type Queue struct {
	logger   *slog.Logger
	client   *pgque.Client
	name     string
	consumer string
	tick     time.Duration
}

// Option configures a Queue during construction.
type Option interface {
	apply(q *Queue)
}

type optionFunc func(*Queue)

func (f optionFunc) apply(r *Queue) { f(r) }

// WithLogger sets the structured logger.
func WithLogger(logger *slog.Logger) Option {
	return optionFunc(func(s *Queue) {
		s.logger = logger
	})
}

// WithQueue sets the queue name (e.g., "dckz"). Required.
func WithQueue(name string) Option {
	return optionFunc(func(s *Queue) {
		s.name = name
	})
}

// WithConsumer sets the consumer name (e.g., "dckz-worker"). Required.
func WithConsumer(name string) Option {
	return optionFunc(func(s *Queue) {
		s.consumer = name
	})
}

// WithTick sets the ticker interval. Defaults to 500ms.
func WithTick(d time.Duration) Option {
	return optionFunc(func(s *Queue) {
		s.tick = d
	})
}

// New builds a Queue from opts, returning an error when a required option is
// missing.
func New(opts ...Option) (*Queue, error) {
	q := &Queue{
		tick: 500 * time.Millisecond,
	}

	for _, opt := range opts {
		opt.apply(q)
	}

	if q.logger == nil {
		return nil, errors.New("pgqueue: logger is required")
	}
	if q.name == "" {
		return nil, errors.New("pgqueue: queue name is required")
	}
	if q.consumer == "" {
		return nil, errors.New("pgqueue: consumer name is required")
	}

	return q, nil
}

// Send appends an event to the queue on db. db must be the caller's bun.Tx to
// join their transaction — the pgque-go client owns a separate pool and cannot
// join a transaction.
func (q *Queue) Send(ctx context.Context, db bun.IDB, eventType string, payload []byte) error {
	q.logger.DebugContext(ctx, "sending message to queue",
		"event_type", eventType,
		"payload_size", len(payload),
	)

	ev := newEvent(payload, eventType)
	eventID, err := q.client.Send(ctx, q.name, *ev)
	if err != nil {
		q.logger.WarnContext(ctx, "failed to send event to queue", "error", err)
		return err
	}

	q.logger.DebugContext(ctx, "sent event to queue", "event_id", eventID)
	return nil
}

// Consume registers handlers and blocks in consumer.Start until ctx is
// cancelled or the substrate dies.
func (q *Queue) Consume(
	ctx context.Context,
	handlers map[domain.Kind]func(ctx context.Context, payload []byte, attempt int) error,
) error {
	q.logger.InfoContext(ctx, "starting consume loop",
		"queue", q.name,
		"consumer", q.consumer,
	)
	c := q.client.NewConsumer(q.consumer, q.name,
		pgque.WithPollInterval(q.tick),
	)

	for k, h := range handlers {
		c.Handle(string(k), func(ctx context.Context, msg pgque.Message) error {
			attempt := 0
			if msg.RetryCount != nil {
				attempt = *msg.RetryCount
			}
			return h(ctx, []byte(msg.Payload), attempt)
		})
	}

	return c.Start(ctx)
}

// RunTicker drives pgque.ticker() at q.tick intervals until ctx is cancelled.
// We drive it ourselves rather than requiring pg_cron.
func (q *Queue) RunTicker(ctx context.Context) error {
	t := time.NewTicker(q.tick)
	defer t.Stop()

	q.logger.InfoContext(ctx, "starting ticker",
		"interval", q.tick,
	)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("run ticker: %w", ctx.Err())
		case <-t.C:
			if _, err := q.client.Ticker(ctx, q.name); err != nil {
				q.logger.ErrorContext(ctx, "failed to run ticker", err)
			}
		}
	}
}

func newEvent(p domain.Payload, eventType string) *pgque.Event {
	return &pgque.Event{
		Payload: p,
		Type:    eventType,
	}
}
