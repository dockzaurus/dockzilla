package handler

import (
	"bytes"
	"dockzilla/internal/core/deployments"
	"dockzilla/internal/infra/transport/http/api"
	"dockzilla/pkg/domain"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

var _ api.DeploymentHandler = (*Deployment)(nil)

type Deployment struct {
	svc    deployments.Handler
	logger *slog.Logger
}

type DeploymentOption interface {
	apply(d *Deployment)
}

type deploymentOptionFunc func(d *Deployment)

func (f deploymentOptionFunc) apply(d *Deployment) { f(d) }

func DeploymentWithLogger(logger *slog.Logger) DeploymentOption {
	return deploymentOptionFunc(func(d *Deployment) {
		d.logger = logger
	})
}

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
		d.logger.WarnContext(c, "failed to bind body")
		c.JSON(http.StatusBadRequest, http.Response{
			Status:     http.StatusText(http.StatusBadRequest),
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(bytes.NewBufferString(err.Error())),
		})

		return
	}

	deploymentID, err := d.svc.Create(c, &payload)
	if err != nil {
		d.logger.ErrorContext(c, "failed to create deployment", "error", err.Error())
		c.JSON(400, errors.New("failed to create deployment"))

		return
	}

	c.JSON(200, deploymentID)
}
