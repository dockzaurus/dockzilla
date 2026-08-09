// Package deployments provides the use case for managing deployments.
package deployments

import (
	"context"
	"fmt"
	"log/slog"

	"dockzilla/internal/core/jobs"
	"dockzilla/pkg/domain"
)

// UseCase handles deployment operations.
type UseCase struct {
	logger    *slog.Logger
	generate  domain.Generator
	parseUUID domain.UUIDParser

	jobs jobs.Handler
	repo Repository
}

// UseCaseOption is a functional option for configuring a UseCase.
type UseCaseOption interface {
	apply(uc *UseCase)
}

type useCaseOptionFunc func(*UseCase)

func (f useCaseOptionFunc) apply(r *UseCase) { f(r) }

// WithLogger sets the logger for the use case.
func WithLogger(logger *slog.Logger) UseCaseOption {
	return useCaseOptionFunc(func(c *UseCase) {
		c.logger = logger
	})
}

// WithGenerator sets the UUID generator for the use case.
func WithGenerator(g domain.Generator) UseCaseOption {
	return useCaseOptionFunc(func(c *UseCase) {
		c.generate = g
	})
}

// WithJobs sets the jobs handler for the use case.
func WithJobs(j jobs.Handler) UseCaseOption {
	return useCaseOptionFunc(func(c *UseCase) {
		c.jobs = j
	})
}

// WithRepo sets the repository for the use case.
func WithRepo(r Repository) UseCaseOption {
	return useCaseOptionFunc(func(c *UseCase) {
		c.repo = r
	})
}

func WithUUIDParser(p domain.UUIDParser) UseCaseOption {
	return useCaseOptionFunc(func(c *UseCase) {
		c.parseUUID = p
	})
}

// NewUseCase creates a new UseCase with the given options.
func NewUseCase(opts ...UseCaseOption) *UseCase {
	uc := new(UseCase)
	for _, opt := range opts {
		opt.apply(uc)
	}

	return uc
}

// Create creates a new deployment.
func (uc *UseCase) Create(
	ctx context.Context,
	input domain.CreateDeploymentInput,
) (domain.UUID, error) {
	uc.logger.InfoContext(ctx, "creating new deployment")

	deploymentID := uc.generate()
	appID, err := uc.parseUUID(input.AppID)
	if err != nil {
		return domain.UUID{}, fmt.Errorf("failed to parse app ID: %w", err)
	}

	deployment := &domain.Deployment{
		Identifier: deploymentID,
		AppID:      appID,
		ImageRef:   input.ImageRef,
		Status:     domain.StatusRunning,
	}

	if _, err = uc.repo.Insert(ctx, *deployment); err != nil {
		uc.logger.WarnContext(ctx, "error inserting deployment", "error", err)
		return domain.UUID{}, fmt.Errorf("failed to insert deployment: %w", err)
	}

	if err = uc.jobs.Enqueue(ctx, domain.RunDeployment, domain.JobsPayload{}); err != nil {
		uc.logger.WarnContext(ctx, "failed enqueuing job", "error", err)
		return domain.UUID{}, fmt.Errorf("failed to enqueue deployment job: %w", err)
	}
	return deployment.Identifier, nil
}
