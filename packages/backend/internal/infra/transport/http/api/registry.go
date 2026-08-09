package api

import "github.com/gin-gonic/gin"

type RegistryHandler interface {
	RegisterSchema(c *gin.Context)
	ListSchemas(c *gin.Context)
	RetrieveSchema(c *gin.Context)
}

func RegistryRoutes(h RegistryHandler) func(router gin.IRouter) {
	return func(router gin.IRouter) {
		group := router.Group("/registry")

		group.GET("/", h.ListSchemas)
		group.POST("/", h.RegisterSchema)
		group.GET("/:kind", h.RetrieveSchema)
		group.GET("/:kind/:version", h.RetrieveSchema)
	}
}
