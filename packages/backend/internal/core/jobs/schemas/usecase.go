package schemas

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"sync"

	"dockzilla/pkg/domain"
	errs "dockzilla/pkg/domain/errors"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

var _ Handler = (*UseCase)(nil)

// UseCase implements the registry. The zero value is not usable; build one
// with NewUseCase.
type UseCase struct {
	logger *slog.Logger

	repo  Repository
	cache CacheRepository

	// catalog holds the schemas this binary ships with, in the order Bootstrap
	// should register them. It is the seed for the database, never a read
	// source.
	catalog []domain.Schema

	// mu guards compiled, which is read on every validation and written the
	// first time a reference is seen. Compiling is pure, so two goroutines
	// racing to compile the same reference is wasteful but not wrong.
	mu       sync.RWMutex
	compiled map[domain.SchemaRef]*jsonschema.Schema
}

// UseCaseOption configures a UseCase during construction. It is an interface
// rather than a bare function type so that options stay comparable in tests
// and can grow behaviour later without breaking callers.
type UseCaseOption interface {
	apply(u *UseCase)
}

type useCaseOptionFunc func(*UseCase)

func (f useCaseOptionFunc) apply(u *UseCase) { f(u) }

// WithLogger sets the structured logger. It is required: NewUseCase fails
// when no logger is provided.
func WithLogger(logger *slog.Logger) UseCaseOption {
	return useCaseOptionFunc(func(u *UseCase) {
		u.logger = logger
	})
}

// WithRepository sets the durable store. It is required: NewUseCase fails when
// no repository is provided.
func WithRepository(repository Repository) UseCaseOption {
	return useCaseOptionFunc(func(u *UseCase) {
		u.repo = repository
	})
}

// WithCache sets the read-through cache. It is optional: without it every read
// that misses the in-process compiled cache goes to the repository.
func WithCache(cache CacheRepository) UseCaseOption {
	return useCaseOptionFunc(func(u *UseCase) {
		u.cache = cache
	})
}

// WithCatalog sets the built-in schemas Bootstrap publishes. It is optional so
// that a test can build a registry with nothing preloaded.
func WithCatalog(catalog []domain.Schema) UseCaseOption {
	return useCaseOptionFunc(func(u *UseCase) {
		u.catalog = catalog
	})
}

// NewUseCase builds a UseCase from opts, returning an error when a required
// option is missing so a caller never receives a partially initialised
// UseCase.
func NewUseCase(opts ...UseCaseOption) (*UseCase, error) {
	uc := &UseCase{
		compiled: make(map[domain.SchemaRef]*jsonschema.Schema),
	}

	for _, opt := range opts {
		opt.apply(uc)
	}

	if uc.logger == nil {
		return nil, errors.New("schema registry use case: logger is required")
	}

	if uc.repo == nil {
		return nil, errors.New("schema registry use case: repository is required")
	}

	return uc, nil
}

// Bootstrap publishes every built-in schema, and is called once per replica
// before the transports start serving.
//
// It fails the boot when a catalog entry disagrees with the row already in the
// database. That is deliberately loud: the only way to reach that state is for
// someone to have edited a frozen version, and a replica that started anyway
// would be validating against a contract different from the one its peers use.
func (u *UseCase) Bootstrap(ctx context.Context) error {
	for _, builtin := range u.catalog {
		if _, err := u.Register(ctx, builtin.Ref, builtin.Document); err != nil {
			return fmt.Errorf("publish built-in schema %s: %w", builtin.Ref, err)
		}
	}

	u.logger.InfoContext(ctx, "published built-in schemas", "count", len(u.catalog))

	return nil
}

// Register publishes document as the schema for ref.
func (u *UseCase) Register(
	ctx context.Context,
	ref domain.SchemaRef,
	document json.RawMessage,
) (domain.Schema, error) {
	if !ref.IsComplete() {
		return domain.Schema{}, fmt.Errorf("%w: %q", errs.ErrInvalidSchemaRef, ref.String())
	}

	// Compiling first means a document that is not a usable schema is rejected
	// before it reaches the table, rather than at the first validation that
	// needs it.
	compiled, err := compile(ref, document)
	if err != nil {
		return domain.Schema{}, err
	}

	stored, err := u.repo.Register(ctx, domain.Schema{Ref: ref, Document: document})
	if err != nil {
		return domain.Schema{}, fmt.Errorf("store schema %s: %w", ref, err)
	}

	identical, err := sameDocument(stored.Document, document)
	if err != nil {
		return domain.Schema{}, fmt.Errorf("compare stored schema %s: %w", ref, err)
	}

	if !identical {
		return domain.Schema{}, fmt.Errorf("%w: %s", errs.ErrSchemaImmutable, ref)
	}

	u.putCompiled(ref, compiled)
	u.putCached(ctx, stored)

	return stored, nil
}

// Retrieve returns the schema registered for ref.
func (u *UseCase) Retrieve(ctx context.Context, ref domain.SchemaRef) (domain.Schema, error) {
	if cached, ok := u.fromCache(ctx, ref); ok {
		return cached, nil
	}

	stored, err := u.repo.Get(ctx, ref)
	if err != nil {
		return domain.Schema{}, fmt.Errorf("load schema %s: %w", ref, err)
	}

	u.putCached(ctx, stored)

	return stored, nil
}

// Latest returns the most recently registered version of kind.
func (u *UseCase) Latest(ctx context.Context, kind domain.Kind) (domain.Schema, error) {
	stored, err := u.repo.Latest(ctx, kind)
	if err != nil {
		return domain.Schema{}, fmt.Errorf("load latest schema for %s: %w", kind, err)
	}

	return stored, nil
}

// List returns every registered schema, newest first.
func (u *UseCase) List(ctx context.Context, kind domain.Kind) ([]domain.Schema, error) {
	stored, err := u.repo.List(ctx, kind)
	if err != nil {
		return nil, fmt.Errorf("list schemas: %w", err)
	}

	return stored, nil
}

// Validate reports whether payload satisfies the schema ref names.
func (u *UseCase) Validate(
	ctx context.Context,
	ref domain.SchemaRef,
	payload domain.Payload,
) error {
	schema, err := u.compiledFor(ctx, ref)
	if err != nil {
		return err
	}

	// The validator needs numbers as json.Number to apply the numeric
	// keywords correctly, which is what its own decoder produces and
	// encoding/json does not.
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(payload))
	if err != nil {
		return errs.Terminal(fmt.Errorf("%w: %s: %w", errs.ErrPayloadInvalid, ref, err))
	}

	if err := schema.Validate(value); err != nil {
		return errs.Terminal(fmt.Errorf("%w: %s: %w", errs.ErrPayloadInvalid, ref, err))
	}

	return nil
}

func (u *UseCase) compiledFor(
	ctx context.Context,
	ref domain.SchemaRef,
) (*jsonschema.Schema, error) {
	u.mu.RLock()
	schema, ok := u.compiled[ref]
	u.mu.RUnlock()

	if ok {
		return schema, nil
	}

	stored, err := u.Retrieve(ctx, ref)
	if err != nil {
		// A reference nobody published will not appear by being retried, so it
		// dead-letters instead of burning the job's attempts. A repository
		// that is merely unreachable stays retryable.
		if errors.Is(err, errs.ErrSchemaNotFound) {
			return nil, errs.Terminal(err)
		}

		return nil, err
	}

	schema, err = compile(ref, stored.Document)
	if err != nil {
		return nil, err
	}

	u.putCompiled(ref, schema)

	return schema, nil
}

func (u *UseCase) putCompiled(ref domain.SchemaRef, schema *jsonschema.Schema) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.compiled[ref] = schema
}

func (u *UseCase) fromCache(ctx context.Context, ref domain.SchemaRef) (domain.Schema, bool) {
	if u.cache == nil {
		return domain.Schema{}, false
	}

	cached, err := u.cache.Get(ctx, ref)
	if err == nil {
		return cached, true
	}

	if !errors.Is(err, errs.ErrSchemaNotFound) {
		u.logger.WarnContext(ctx, "schema cache read failed",
			"schema_ref", ref.String(),
			"error", err,
		)
	}

	return domain.Schema{}, false
}

func (u *UseCase) putCached(ctx context.Context, schema domain.Schema) {
	if u.cache == nil {
		return
	}

	if err := u.cache.Put(ctx, schema); err != nil {
		u.logger.WarnContext(ctx, "schema cache write failed",
			"schema_ref", schema.Ref.String(),
			"error", err,
		)
	}
}

func compile(ref domain.SchemaRef, document json.RawMessage) (*jsonschema.Schema, error) {
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(document))
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", errs.ErrSchemaInvalid, ref, err)
	}

	location := resourceURL(ref)

	compiler := jsonschema.NewCompiler()
	if addErr := compiler.AddResource(location, value); addErr != nil {
		return nil, fmt.Errorf("%w: %s: %w", errs.ErrSchemaInvalid, ref, addErr)
	}

	schema, err := compiler.Compile(location)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", errs.ErrSchemaInvalid, ref, err)
	}

	return schema, nil
}

// resourceURL is the identity a document is compiled under. It is a URN rather
// than an HTTP URL so that a stored schema never carries a deployment's
// hostname and the compiler never tries to fetch anything over the network.
func resourceURL(ref domain.SchemaRef) string {
	return "urn:dockzilla:schema:" + string(ref.Kind) + ":" + string(ref.Version)
}

// sameDocument compares two schema documents by value rather than by bytes,
// so that reformatting or a different key order does not read as a rewrite.
func sameDocument(left, right json.RawMessage) (bool, error) {
	var decodedLeft, decodedRight any

	if err := json.Unmarshal(left, &decodedLeft); err != nil {
		return false, fmt.Errorf("decode stored document: %w", err)
	}

	if err := json.Unmarshal(right, &decodedRight); err != nil {
		return false, fmt.Errorf("decode submitted document: %w", err)
	}

	return reflect.DeepEqual(decodedLeft, decodedRight), nil
}
