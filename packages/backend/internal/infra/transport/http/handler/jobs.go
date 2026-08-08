package handler

import (
	"dockzilla/internal/core/jobs"
	"dockzilla/internal/infra/transport/http/api"
	"log/slog"

	"github.com/gin-gonic/gin"
)

type Jobs struct {
	service jobs.Handler
	logger  *slog.Logger
}

var _ api.JobsHandler = (*Jobs)(nil)

type JobOption interface {
	apply(*Jobs)
}

type jobOptionFunc func(*Jobs)

func (f jobOptionFunc) apply(j *Jobs) {}

func NewJobs(opts ...JobOption) *Jobs {
	return new(Jobs)
}

func (j *Jobs) Create(c *gin.Context) {
	j.logger.InfoContext(c, "creating new job")
}
