package repository

import (
	"context"
	"dockzilla/internal/core/deployments"
	"dockzilla/pkg/domain"
	"log/slog"
)

var _ deployments.Repository = (*Deployment)(nil)

type Deployment struct {
	logger *slog.Logger
}

type DeploymentOption interface {
	apply(*Deployment)
}
type deploymentOptionFunc func(*Deployment)

func (f deploymentOptionFunc) apply(d *Deployment) {}

func DeploymentWithLogger(logger *slog.Logger) DeploymentOption {
	return deploymentOptionFunc(func(d *Deployment) {
		d.logger = logger
	})
}

func NewDeployment(options ...DeploymentOption) *Deployment {
	d := new(Deployment)
	for _, option := range options {
		option.apply(d)
	}

	return d
}

func (d Deployment) Insert(ctx context.Context, deployment domain.Deployment) (domain.UUID, error) {
	//TODO implement me
	d.logger.InfoContext(ctx, "Inserting deployment", deployment)

	return deployment.Identifier, nil
}
