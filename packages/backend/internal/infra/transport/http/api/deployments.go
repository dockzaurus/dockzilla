package api

import "github.com/gin-gonic/gin"

// DeploymentHandler defines the HTTP handler interface for deployment routes.
type DeploymentHandler interface {
	Create(c *gin.Context)
}

// DeploymentRoutes returns a function that registers deployment routes.
func DeploymentRoutes(h DeploymentHandler) func(router gin.IRouter) {
	return func(router gin.IRouter) {
		group := router.Group("/deployments")
		group.POST("", h.Create)
	}
}
