package api

import (
	"log/slog"

	"dockzilla/internal/infra/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

// DeploymentHandler defines the HTTP handler interface for deployment routes.
type DeploymentHandler interface {
	Create(c *gin.Context)
}

// DeploymentRoutes returns a function that registers deployment routes.
//
// Every route in the group runs behind RequireAuth: creating a deployment
// records who asked for it, and the use case refuses a context with no actor
// rather than writing a row that claims nobody did it. The logger is the one
// the middleware reports a construction failure on.
func DeploymentRoutes(h DeploymentHandler, logger *slog.Logger) func(router gin.IRouter) {
	return func(router gin.IRouter) {
		group := router.Group("/deployments")
		group.Use(middleware.RequireAuth(nil, logger))
		group.POST("", h.Create)
	}
}
