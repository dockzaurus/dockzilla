// Package errors holds the sentinel errors and the retry classification shared
// by every layer of the backend. It lives apart from domain so that both the
// domain vocabulary and the adapters can depend on it without a cycle.
package errors

import "errors"

var (
	// ErrPayloadTooLarge is returned when a job payload exceeds MaxPayloadSize.
	ErrPayloadTooLarge = errors.New("payload exceeds maximum size")
	// ErrPayloadEmpty is returned when a job payload is empty.
	ErrPayloadEmpty = errors.New("payload cannot be empty")

	// ErrNoTransaction is returned by Insert when it is called outside a unit of
	// work. Enqueueing on the pool instead would commit the job independently of
	// the domain write it belongs to — the dual write the port exists to prevent.
	ErrNoTransaction = errors.New("jobs repository: insert requires an ambient transaction")

	// ErrUnsupportedOption is returned when a caller asks for job semantics the
	// queue substrate cannot provide. Failing beats silently dropping the option.
	ErrUnsupportedOption = errors.New("jobs repository: option unsupported by pgque")

	// ErrNoActor is returned when a use case that records provenance runs on a
	// context no middleware attached an Actor to. It is a wiring bug, not a bad
	// request: the route is reachable without the middleware that identifies
	// the caller.
	ErrNoActor = errors.New("context carries no actor")
)

type terminal struct{ error }

// Unwrap exposes the wrapped error.
//
// Embedding an error promotes Error but not Unwrap, so without this method
// Terminal is where a chain stops: errors.Is could no longer see the sentinel
// underneath, and a caller asking "is this ErrSchemaNotFound" would be told no
// purely because something classified it as non-retryable on the way up.
func (t terminal) Unwrap() error {
	return t.error
}

// Terminal wraps err so it is classified as non-retryable and dead-lettered
// immediately. Use it for bad payloads, missing handlers, and 404s.
func Terminal(err error) error {
	return terminal{err}
}

// IsTerminal reports whether err is a terminal error.
func IsTerminal(err error) bool {
	var t terminal
	return errors.As(err, &t)
}
