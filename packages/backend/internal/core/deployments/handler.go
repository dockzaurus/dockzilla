package deployments

// Handler defines the HTTP handler interface for deployments.
type Handler interface {
	Create() error
}
