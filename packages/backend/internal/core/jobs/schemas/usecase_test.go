package schemas_test

import (
	"encoding/json"
	"errors"
	"strconv"
	"sync"
	"testing"

	"dockzilla/internal/core/jobs/schemas"
	"dockzilla/internal/core/jobs/schemas/mocks"
	"dockzilla/pkg/domain"
	errs "dockzilla/pkg/domain/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewUseCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    func(repo schemas.Repository) []schemas.UseCaseOption
		wantErr string
	}{
		{
			name:    "error - no options",
			opts:    func(schemas.Repository) []schemas.UseCaseOption { return nil },
			wantErr: "schema registry use case: logger is required",
		},
		{
			name: "error - repository missing",
			opts: func(schemas.Repository) []schemas.UseCaseOption {
				return []schemas.UseCaseOption{schemas.WithLogger(discardLogger())}
			},
			wantErr: "schema registry use case: repository is required",
		},
		{
			name: "success - every required option supplied",
			opts: func(repo schemas.Repository) []schemas.UseCaseOption {
				return []schemas.UseCaseOption{
					schemas.WithLogger(discardLogger()),
					schemas.WithRepository(repo),
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc, err := schemas.NewUseCase(tt.opts(mocks.NewMockRepository(t))...)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				require.Nil(t, uc)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, uc)
		})
	}
}

func TestUseCase_Register(t *testing.T) {
	t.Parallel()

	t.Run("success - a new reference is published and returned", func(t *testing.T) {
		t.Parallel()

		repo := newRepo(t)
		repo.EXPECT().
			Register(mock.Anything, mock.Anything).
			Return(testSchema(), nil).
			Once()

		stored, err := newUseCase(t, repo).Register(t.Context(), testRef(), testDocument())

		require.NoError(t, err)
		require.Equal(t, testRef(), stored.Ref)
	})

	t.Run("success - re-registering the same document is a no-op", func(t *testing.T) {
		t.Parallel()

		// The store keeps the row it already has; the reformatted submission
		// carries the same schema, so it must not read as a rewrite.
		repo := newRepo(t)
		repo.EXPECT().
			Register(mock.Anything, mock.Anything).
			Return(testSchema(), nil).
			Once()

		reformatted := json.RawMessage(
			`{"additionalProperties":false,"required":["app_id"],` +
				`"properties":{"app_id":{"minLength":1,"type":"string"}},"type":"object",` +
				`"$schema":"https://json-schema.org/draft/2020-12/schema"}`,
		)

		_, err := newUseCase(t, repo).Register(t.Context(), testRef(), reformatted)

		require.NoError(t, err)
	})

	t.Run("error - a published version cannot be rewritten", func(t *testing.T) {
		t.Parallel()

		// This is the drift a rolling deploy would otherwise hide: the row
		// already stored says something different from what this build ships.
		repo := newRepo(t)
		repo.EXPECT().
			Register(mock.Anything, mock.Anything).
			Return(testSchema(), nil).
			Once()

		different := json.RawMessage(`{"type": "object", "required": ["something_else"]}`)

		_, err := newUseCase(t, repo).Register(t.Context(), testRef(), different)

		require.ErrorIs(t, err, errs.ErrSchemaImmutable)
	})

	t.Run("error - a document that is not a schema never reaches the store", func(t *testing.T) {
		t.Parallel()

		// No repository expectation: reaching it would fail the mock.
		repo := newRepo(t)
		notASchema := json.RawMessage(`{"type": 42}`)

		_, err := newUseCase(t, repo).Register(t.Context(), testRef(), notASchema)

		require.ErrorIs(t, err, errs.ErrSchemaInvalid)
	})

	t.Run("error - an incomplete reference is rejected", func(t *testing.T) {
		t.Parallel()

		repo := newRepo(t)
		ref := domain.SchemaRef{Kind: domain.StartApp}

		_, err := newUseCase(t, repo).Register(t.Context(), ref, testDocument())

		require.ErrorIs(t, err, errs.ErrInvalidSchemaRef)
	})
}

func TestUseCase_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload string
		wantErr error
	}{
		{
			name:    "success - a payload that satisfies the schema",
			payload: `{"app_id": "a3f1"}`,
		},
		{
			name:    "error - a required property is missing",
			payload: `{}`,
			wantErr: errs.ErrPayloadInvalid,
		},
		{
			name:    "error - an unexpected property is present",
			payload: `{"app_id": "a3f1", "surprise": true}`,
			wantErr: errs.ErrPayloadInvalid,
		},
		{
			name:    "error - the payload is not json at all",
			payload: `not json`,
			wantErr: errs.ErrPayloadInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := newRepo(t)
			repo.EXPECT().Get(mock.Anything, testRef()).Return(testSchema(), nil).Once()

			err := newUseCase(t, repo).
				Validate(t.Context(), testRef(), domain.Payload(tt.payload))

			if tt.wantErr == nil {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, tt.wantErr)
			// A payload cannot become valid by being retried, so the engine
			// has to dead-letter it rather than burn its attempts.
			require.True(t, errs.IsTerminal(err), "validation failure must be terminal")
		})
	}
}

func TestUseCase_Validate_UnknownReferenceIsTerminal(t *testing.T) {
	t.Parallel()

	repo := newRepo(t)
	repo.EXPECT().
		Get(mock.Anything, testRef()).
		Return(domain.Schema{}, errs.ErrSchemaNotFound).
		Once()

	err := newUseCase(t, repo).Validate(t.Context(), testRef(), domain.Payload(`{}`))

	require.ErrorIs(t, err, errs.ErrSchemaNotFound)
	require.True(t, errs.IsTerminal(err), "an unpublished contract must dead-letter")
}

func TestUseCase_Validate_StoreFailureStaysRetryable(t *testing.T) {
	t.Parallel()

	// A registry that is merely unreachable says nothing about the payload, so
	// the job must keep its retries.
	unreachable := errors.New("dial tcp: connection refused")

	repo := newRepo(t)
	repo.EXPECT().
		Get(mock.Anything, testRef()).
		Return(domain.Schema{}, unreachable).
		Once()

	err := newUseCase(t, repo).Validate(t.Context(), testRef(), domain.Payload(`{"app_id":"a"}`))

	require.ErrorIs(t, err, unreachable)
	require.False(t, errs.IsTerminal(err), "an unreachable store must stay retryable")
}

func TestUseCase_Validate_CompilesEachReferenceOnce(t *testing.T) {
	t.Parallel()

	// Once is the assertion: the second validation must be served from the
	// in-process compiled cache rather than reloading the document.
	repo := newRepo(t)
	repo.EXPECT().Get(mock.Anything, testRef()).Return(testSchema(), nil).Once()

	uc := newUseCase(t, repo)

	require.NoError(t, uc.Validate(t.Context(), testRef(), domain.Payload(`{"app_id":"a"}`)))
	require.NoError(t, uc.Validate(t.Context(), testRef(), domain.Payload(`{"app_id":"b"}`)))
}

func TestUseCase_Validate_IsSafeUnderConcurrentFirstUse(t *testing.T) {
	t.Parallel()

	// Run with -race: several goroutines racing to populate the compiled cache
	// for the same reference is the interesting case.
	repo := newRepo(t)
	repo.EXPECT().Get(mock.Anything, testRef()).Return(testSchema(), nil)

	uc := newUseCase(t, repo)

	const workers = 16

	var wg sync.WaitGroup

	wg.Add(workers)

	for i := range workers {
		go func() {
			defer wg.Done()

			payload := domain.Payload(`{"app_id": "` + strconv.Itoa(i) + `"}`)
			require.NoError(t, uc.Validate(t.Context(), testRef(), payload))
		}()
	}

	wg.Wait()
}

func TestUseCase_Retrieve(t *testing.T) {
	t.Parallel()

	t.Run("success - a cache hit does not reach the store", func(t *testing.T) {
		t.Parallel()

		repo := newRepo(t)
		cache := mocks.NewMockCacheRepository(t)
		cache.EXPECT().Get(mock.Anything, testRef()).Return(testSchema(), nil).Once()

		stored, err := newUseCase(t, repo, schemas.WithCache(cache)).
			Retrieve(t.Context(), testRef())

		require.NoError(t, err)
		require.Equal(t, testRef(), stored.Ref)
	})

	t.Run("success - a cache miss falls through and warms the cache", func(t *testing.T) {
		t.Parallel()

		repo := newRepo(t)
		repo.EXPECT().Get(mock.Anything, testRef()).Return(testSchema(), nil).Once()

		cache := mocks.NewMockCacheRepository(t)
		cache.EXPECT().
			Get(mock.Anything, testRef()).
			Return(domain.Schema{}, errs.ErrSchemaNotFound).
			Once()
		cache.EXPECT().Put(mock.Anything, mock.Anything).Return(nil).Once()

		_, err := newUseCase(t, repo, schemas.WithCache(cache)).Retrieve(t.Context(), testRef())

		require.NoError(t, err)
	})

	t.Run("success - a broken cache costs latency, not correctness", func(t *testing.T) {
		t.Parallel()

		repo := newRepo(t)
		repo.EXPECT().Get(mock.Anything, testRef()).Return(testSchema(), nil).Once()

		cache := mocks.NewMockCacheRepository(t)
		cache.EXPECT().
			Get(mock.Anything, testRef()).
			Return(domain.Schema{}, errors.New("redis down")).
			Once()
		cache.EXPECT().Put(mock.Anything, mock.Anything).Return(errors.New("redis down")).Once()

		stored, err := newUseCase(t, repo, schemas.WithCache(cache)).
			Retrieve(t.Context(), testRef())

		require.NoError(t, err)
		require.Equal(t, testRef(), stored.Ref)
	})

	t.Run("error - an unregistered reference is not found", func(t *testing.T) {
		t.Parallel()

		repo := newRepo(t)
		repo.EXPECT().
			Get(mock.Anything, testRef()).
			Return(domain.Schema{}, errs.ErrSchemaNotFound).
			Once()

		_, err := newUseCase(t, repo).Retrieve(t.Context(), testRef())

		require.ErrorIs(t, err, errs.ErrSchemaNotFound)
	})
}

func TestUseCase_Bootstrap(t *testing.T) {
	t.Parallel()

	t.Run("success - every catalog entry is published", func(t *testing.T) {
		t.Parallel()

		repo := newRepo(t)
		repo.EXPECT().Register(mock.Anything, mock.Anything).Return(testSchema(), nil).Once()

		err := newUseCase(t, repo, schemas.WithCatalog([]domain.Schema{testSchema()})).
			Bootstrap(t.Context())

		require.NoError(t, err)
	})

	t.Run("error - boot fails when a shipped schema disagrees with the store", func(t *testing.T) {
		t.Parallel()

		// Starting anyway would leave this replica validating against a
		// different contract from its peers, which is the whole failure mode
		// the registry exists to prevent.
		repo := newRepo(t)
		repo.EXPECT().Register(mock.Anything, mock.Anything).Return(testSchema(), nil).Once()

		shipped := domain.Schema{
			Ref:      testRef(),
			Document: json.RawMessage(`{"type": "object", "required": ["app_id", "region"]}`),
		}

		err := newUseCase(t, repo, schemas.WithCatalog([]domain.Schema{shipped})).
			Bootstrap(t.Context())

		require.ErrorIs(t, err, errs.ErrSchemaImmutable)
	})
}
