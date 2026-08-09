package domain

import (
	"context"
	"encoding/json"
	"time"

	errs "dockzilla/pkg/domain/errors"
)

// JobsPayload is the job argument blob, bounded at MaxPayloadSize.
type JobsPayload = json.RawMessage

// NewPayload validates and returns a Payload, or an error if the size is invalid.
func NewPayload(b []byte) (JobsPayload, error) {
	if len(b) > MaxPayloadSize {
		return nil, errs.ErrPayloadTooLarge
	}
	if len(b) == 0 {
		return nil, errs.ErrPayloadEmpty
	}
	return b, nil
}

// Key is a serialization key for per-resource job uniqueness.
type Key string

// Kind identifies the job type and routes to a registered handler.
type Kind string

const (
	// RunDeployment pulls an image, creates a container, starts it, waits for
	// health, swaps the proxy, and stops the old container.
	RunDeployment Kind = "deployment.run"

	// StartApp starts an app container.
	StartApp Kind = "app.start"
	// StopApp stops an app container.
	StopApp Kind = "app.stop"
	// RestartApp restarts an app container.
	RestartApp Kind = "app.restart"
)

// Envelope is the wire shape written to the queue. pgque.send carries only a
// type and a payload, so the message identifier travels inside the payload to
// survive the round trip. Args holds the handler's arguments untouched, so
// Register[T] still unmarshals exactly what the producer sent.
type Envelope struct {
	ID   UUID        `json:"id"`
	Args JobsPayload `json:"args"`
}

// AllKinds returns every job kind the engine knows about. Substrates that
// route by message type need the full set up front to register one handler per
// kind, so a new Kind constant must be added here to ever be consumed.
func AllKinds() []Kind {
	return []Kind{
		RunDeployment,
		StartApp,
		StopApp,
		RestartApp,
	}
}

// HeaderFrame carries job metadata.
type HeaderFrame struct {
	Identifier UUID
	Kind       Kind
}

// Message is the unit enqueued into and received from the job queue.
type Message struct {
	Header   HeaderFrame
	Payload  JobsPayload
	Attempts uint32
}

// JobConfig holds optional parameters for job enqueueing.
type JobConfig struct {
	RunAfter    time.Time
	MaxAttempts int32
	UniqueKey   Key
}

// JobOption configures a JobConfig during enqueueing.
type JobOption interface {
	apply(c *JobConfig)
}

type jobOptionFunc func(*JobConfig)

func (f jobOptionFunc) apply(c *JobConfig) { f(c) }

// WithRunAfter schedules the job to run no earlier than t.
func WithRunAfter(t time.Time) JobOption {
	return jobOptionFunc(func(c *JobConfig) { c.RunAfter = t })
}

// WithMaxAttempts sets the retry limit before dead-lettering.
func WithMaxAttempts(n int32) JobOption {
	return jobOptionFunc(func(c *JobConfig) { c.MaxAttempts = n })
}

// WithUniqueKey serializes jobs with the same key — only one runs at a time.
func WithUniqueKey(k Key) JobOption {
	return jobOptionFunc(func(c *JobConfig) { c.UniqueKey = k })
}

// NewJobConfig applies opts and returns the config with defaults.
func NewJobConfig(opts ...JobOption) JobConfig {
	c := JobConfig{MaxAttempts: 3}
	for _, o := range opts {
		o.apply(&c)
	}
	return c
}

// Dispatch is called per job by the consume loop. A nil return acks the job;
// a Terminal error dead-letters it immediately; any other error retries with
// backoff until attempts are exhausted, then dead-letters.
type Dispatch func(ctx context.Context, msg Message) error
