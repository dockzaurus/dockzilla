package handler

import (
	"log/slog"
	"net/http"

	"dockzilla/internal/core/deployments"
	dockzillahttp "dockzilla/internal/infra/transport/http"
	"dockzilla/internal/infra/transport/http/api"
	"dockzilla/pkg/domain"
	"github.com/gin-gonic/gin"
)

var _ api.DeploymentHandler = (*Deployment)(nil)

// Deployment is the HTTP handler for deployment routes.
type Deployment struct {
	svc    deployments.Handler
	logger *slog.Logger
}

// DeploymentOption is a functional option for configuring a Deployment handler.
type DeploymentOption interface {
	apply(d *Deployment)
}

type deploymentOptionFunc func(d *Deployment)

func (f deploymentOptionFunc) apply(d *Deployment) { f(d) }

// DeploymentWithLogger sets the logger for the Deployment handler.
func DeploymentWithLogger(logger *slog.Logger) DeploymentOption {
	return deploymentOptionFunc(func(d *Deployment) {
		d.logger = logger
	})
}

// DeploymentWithHandler sets the deployments service for the Deployment handler.
func DeploymentWithHandler(svc deployments.Handler) DeploymentOption {
	return deploymentOptionFunc(func(d *Deployment) {
		d.svc = svc
	})
}

// NewDeployment create the DeploymentHandler instance.
func NewDeployment(opts ...DeploymentOption) (*Deployment, error) {
	d := new(Deployment)
	for _, opt := range opts {
		opt.apply(d)
	}

	return d, nil
}

// Create handle the request to create a new deployment.
func (d *Deployment) Create(c *gin.Context) {
	d.logger.InfoContext(c, "creating new deployment")
	var payload domain.CreateDeploymentInput
	if err := c.ShouldBindBodyWithJSON(&payload); err != nil {
		d.logger.DebugContext(c, "failed to bind body", "error", err)
		dockzillahttp.Abort(c, http.StatusBadRequest, "invalid request body")

		return
	}

	// The actor rides on the request context, put there by middleware.RequireAuth.
	ctx := c.Request.Context()

	deploymentID, err := d.svc.Create(ctx, &payload)
	if err != nil {
		d.logger.ErrorContext(ctx, "failed to create deployment", "error", err)
		dockzillahttp.Abort(
			c,
			http.StatusInternalServerError,
			"failed to create deployment",
		)

		return
	}

	c.JSON(http.StatusCreated, gin.H{"deployment_id": deploymentID.String()})
}
