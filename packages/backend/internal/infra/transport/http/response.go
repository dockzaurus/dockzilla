package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Abort writes message as the error body and stops the handler chain.
//
// It is Abort rather than a plain c.JSON because a middleware that rejects a
// request must also prevent the handlers queued behind it from running. A
// c.JSON alone writes the body and then lets the route's own handler execute
// against exactly the state the middleware just refused to accept, which turns
// a rejection into a 200 with two bodies concatenated.
//
// The message reaches the client, so it says what is wrong with the request
// and never why the server is unhappy. Causes are logged, not served.
func Abort(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": message})
}

// Failing returns a handler that aborts every request it sees with status and
// message, logging cause once, here, rather than on each request.
//
// It exists for middleware whose construction can fail. A constructor that
// returns a nil gin.HandlerFunc hands the router something it will panic on at
// the first request, which reports a configuration mistake as a crash in an
// unrelated place and takes the whole process with it. Returning this instead
// keeps the failure local: the routes behind it refuse cleanly and say so, and
// every other route on the server carries on serving.
//
// The cause is deliberately not in the response body. A caller cannot act on
// it, and a misconfigured auth middleware is exactly where an internal detail
// should not be handed to whoever asks.
func Failing(logger *slog.Logger, status int, message string, cause error) gin.HandlerFunc {
	// Construction happens at startup, before any request exists, so there is
	// no request context to carry here.
	logger.ErrorContext(context.Background(),
		"middleware is misconfigured and will refuse every request",
		"status", status,
		"message", message,
		"error", cause,
	)

	return func(c *gin.Context) {
		Abort(c, status, message)
	}
}

// Misconfigured is Failing with the status and wording every construction
// failure should share: the request was fine, the server is not.
func Misconfigured(logger *slog.Logger, cause error) gin.HandlerFunc {
	return Failing(
		logger,
		http.StatusInternalServerError,
		"the server is misconfigured",
		cause,
	)
}
