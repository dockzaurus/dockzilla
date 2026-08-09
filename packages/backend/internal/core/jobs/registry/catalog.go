package registry

import (
	"context"
	"dockzilla/pkg/domain"
	"log/slog"
)

type EntryIdentity string
type Entries = map[EntryIdentity]*domain.Payload

func (e EntryIdentity) String() string {
	return string(e)
}

func BuildEntryIdentity(kind domain.Kind, version string) EntryIdentity {
	return EntryIdentity(string(kind) + "." + version)
}

type Catalog struct {
	logger  *slog.Logger
	entries Entries
}

type CatalogOption interface {
	apply(*Catalog)
}

type catalogOptionFunc func(*Catalog)

func (f catalogOptionFunc) apply(c *Catalog) { f(c) }

func NewCatalog(opts ...CatalogOption) *Catalog {
	c := new(Catalog)
	for _, opt := range opts {
		opt.apply(c)
	}

	return c
}

// Discover will check all go-schema to register them in the registry.
func (c *Catalog) Discover(ctx context.Context) error {
	if err := c.Register(ctx, domain.RunDeployment, "v0.0.1", &domain.Payload{}); err != nil {
		return err
	}

	return nil
}

func (c *Catalog) Register(ctx context.Context, kind domain.Kind, version string, schema *domain.Payload) error {
	return nil
}

func (c *Catalog) Entries() (Entries, error) {
	return c.entries, nil
}
