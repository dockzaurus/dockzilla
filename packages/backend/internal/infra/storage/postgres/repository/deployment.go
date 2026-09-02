package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"dockzilla/internal/core/deployments"
	"dockzilla/internal/infra/storage/postgres"
	"dockzilla/internal/models"
	"dockzilla/pkg/domain"
	"github.com/uptrace/bun"
)

var _ deployments.Repository = (*Deployment)(nil)

// Deployment adapts the deployments table to the deployments.Repository port.
// The zero value is not usable; build one with NewDeployment.
type Deployment struct {
	logger *slog.Logger
	db     bun.IDB
}

// DeploymentOption is a functional option for configuring a Deployment repository.
type DeploymentOption interface {
	apply(d *Deployment)
}
type deploymentOptionFunc func(*Deployment)

func (f deploymentOptionFunc) apply(d *Deployment) { f(d) }

// DeploymentWithLogger sets the logger for the Deployment repository. Required.
func DeploymentWithLogger(logger *slog.Logger) DeploymentOption {
	return deploymentOptionFunc(func(d *Deployment) {
		d.logger = logger
	})
}

// DeploymentWithDB sets the database handle used when no transaction is
// ambient. Required.
func DeploymentWithDB(db bun.IDB) DeploymentOption {
	return deploymentOptionFunc(func(d *Deployment) {
		d.db = db
	})
}

// NewDeployment builds a Deployment from options, returning an error when a
// required option is missing.
func NewDeployment(options ...DeploymentOption) (*Deployment, error) {
	d := new(Deployment)
	for _, option := range options {
		option.apply(d)
	}

	if d.logger == nil {
		return nil, errors.New("deployment repository: logger is required")
	}

	if d.db == nil {
		return nil, errors.New("deployment repository: database is required")
	}

	return d, nil
}

// Insert writes deployment to the ledger and returns its identifier.
//
// The query runs through postgres.IDB so it joins the transaction the use case
// opened, which is what makes the row and the job enqueued beside it commit
// together. The fallback is the pool rather than nil: unlike the jobs
// repository, inserting a deployment outside a unit of work is a legitimate
// call, so there is nothing to forbid here.
//
// The identifier returned is the one the domain generated, never the row's
// bigint ID — that column orders insertions and must not leave storage.
func (d *Deployment) Insert(
	ctx context.Context,
	deployment domain.Deployment,
) (domain.UUID, error) {
	row := deploymentRow(deployment)

	d.logger.DebugContext(ctx, "inserting deployment",
		"deployment_id", row.Identifier,
		"app_id", row.AppID,
		"status", row.Status,
	)

	if _, err := postgres.IDB(ctx, d.db).NewInsert().Model(row).Exec(ctx); err != nil {
		return domain.UUID{}, fmt.Errorf("insert deployment %s: %w", row.Identifier, err)
	}

	return deployment.Identifier, nil
}

func deploymentRow(deployment domain.Deployment) *models.Deployments {
	row := &models.Deployments{
		Identifier:  deployment.Identifier.String(),
		AppID:       deployment.AppID.String(),
		ImageRef:    deployment.ImageRef,
		Status:      string(deployment.Status),
		TriggeredBy: string(deployment.TriggeredBy),
	}

	if deployment.TriggeredByUserID != nil {
		row.TriggeredByUserID = deployment.TriggeredByUserID.String()
	}

	return row
}
