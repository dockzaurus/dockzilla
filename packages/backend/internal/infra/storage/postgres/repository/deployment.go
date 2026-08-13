package repository

import (
	"context"
	"log/slog"

	"dockzilla/internal/core/deployments"
	"dockzilla/pkg/domain"
)

var _ deployments.Repository = (*Deployment)(nil)

// Deployment is the postgres repository for deployments.
type Deployment struct {
	logger *slog.Logger
}

// DeploymentOption is a functional option for configuring a Deployment repository.
type DeploymentOption interface {
	apply(d *Deployment)
}
type deploymentOptionFunc func(*Deployment)

func (f deploymentOptionFunc) apply(d *Deployment) { f(d) }

// DeploymentWithLogger sets the logger for the Deployment repository.
func DeploymentWithLogger(logger *slog.Logger) DeploymentOption {
	return deploymentOptionFunc(func(d *Deployment) {
		d.logger = logger
	})
}

// NewDeployment creates a new Deployment repository with the given options.
func NewDeployment(options ...DeploymentOption) *Deployment {
	d := new(Deployment)
	for _, option := range options {
		option.apply(d)
	}

	return d
}

// Insert persists the given deployment and returns its identifier.
func (d Deployment) Insert(
	ctx context.Context,
	deployment domain.Deployment,
) (domain.UUID, error) {
	// TODO implement me
	d.logger.InfoContext(ctx, "Inserting deployment", "deployment", deployment)

	return deployment.Identifier, nil
}
