// Package api holds the HTTP route tables. Each function in it binds a set of
// paths to the handler it is given and returns the registration, so the
// transport can mount a feature without knowing what that feature does.
package api

import "github.com/gin-gonic/gin"

// SampleHandler is the method set the sample routes require. It is declared
// here, by the consumer, so any handler with these methods satisfies it
// implicitly and the handler package never has to import this one.
type SampleHandler interface {
	SayHello(c *gin.Context)
	SendHello(c *gin.Context)
}

// SampleRoutes returns a registration that mounts the sample endpoints on the
// router it is handed, dispatching to h. Pass the result to http.WithRoutes.
func SampleRoutes(h SampleHandler) func(router gin.IRouter) {
	return func(router gin.IRouter) {
		group := router.Group("/sample")

		group.GET("/hello/:name", h.SayHello)
		group.POST("/hello/:name", h.SendHello)
	}
}
