package jobs

import "errors"

type terminal struct{ error }

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
