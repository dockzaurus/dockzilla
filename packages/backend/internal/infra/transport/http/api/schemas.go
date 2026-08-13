package api

import "github.com/gin-gonic/gin"

// SchemasHandler is the method set the schema registry routes require. It is
// declared here, by the consumer, so any handler with these methods satisfies
// it implicitly and the handler package never has to import this one.
type SchemasHandler interface {
	List(c *gin.Context)
	Register(c *gin.Context)
	RetrieveLatest(c *gin.Context)
	Retrieve(c *gin.Context)
}

// SchemasRoutes returns a registration that mounts the schema registry on the
// router it is handed, dispatching to h. Pass the result to http.WithRoutes.
//
// Mounted under the server's base path, the endpoints are:
//
//	GET  /v1/schema/registry                  list registered schemas
//	GET  /v1/schema/registry?kind=app.stop    list one kind's versions
//	POST /v1/schema/registry                  publish a schema
//	GET  /v1/schema/registry/:kind            fetch the latest version
//	GET  /v1/schema/registry/:kind/:version   fetch one version
//
// The path says "schema registry" rather than "registry" because a registry in
// this product is where container images live.
func SchemasRoutes(h SchemasHandler) func(router gin.IRouter) {
	return func(router gin.IRouter) {
		group := router.Group("/schema/registry")

		group.GET("", h.List)
		group.POST("", h.Register)
		group.GET("/:kind", h.RetrieveLatest)
		group.GET("/:kind/:version", h.Retrieve)
	}
}
