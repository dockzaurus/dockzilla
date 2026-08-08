package api

import "github.com/gin-gonic/gin"

type JobsHandler interface {
	Create(c *gin.Context)
}

func JobsRoutes(h JobsHandler) func(router gin.IRouter) {
	return func(router gin.IRouter) {
		group := router.Group("/jobs")

		deployment := group.Group("/deployments")
		deployment.GET("/:id", func(c *gin.Context) {})
		deployment.GET("", func(c *gin.Context) {})
		deployment.POST("", h.Create)
	}
}
