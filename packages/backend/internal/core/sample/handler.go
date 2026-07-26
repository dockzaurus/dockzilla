package sample

import "context"

// Handler is the slice of the sample use case that Sample calls. It is
// declared here, by the consumer, so the transport never depends on more of the
// core than it uses.
type Handler interface {
	SayHello(ctx context.Context, name string) (string, error)
	SendHello(ctx context.Context, name string) error
}
