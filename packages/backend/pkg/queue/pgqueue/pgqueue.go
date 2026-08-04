package pgqueue

import (
	"context"
	"log/slog"
	"time"

	"github.com/uptrace/bun"
)

type Queue struct {
	logger *slog.Logger
}

type Option interface {
	apply(*Queue)
}

type optionFunc func(*Queue)

func (f optionFunc) apply(r *Queue) { f(r) }

func WithLogger(logger *slog.Logger) Option {
	return optionFunc(func(s *Queue) {
		s.logger = logger
	})
}

func New(opts ...Option) (*Queue, error) {
	q := new(Queue)

	for _, opt := range opts {
		opt.apply(q)
	}

	return q, nil
}

func (q *Queue) Send(ctx context.Context, db bun.IDB, eventType string, payload []byte) error {
	q.logger.InfoContext(ctx, "sending message to queue", slog.String("event_type", eventType))
	return nil
}

func (q *Queue) Consume(ctx context.Context, handlers map[string]func(ctx context.Context, payload []byte, attempt int) error) error {
	q.logger.InfoContext(ctx, "starting consume handler")
	return nil
}

func (q *Queue) RunTicker(ctx context.Context, interval time.Duration) {
	q.logger.InfoContext(ctx, "starting ticker")
}
