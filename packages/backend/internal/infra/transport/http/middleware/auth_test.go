package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"dockzilla/internal/infra/transport/http/middleware"
	"dockzilla/pkg/domain"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// TestRequireAuth_AttachesTheActor is what the rest of the stack depends on:
// deployments.Create refuses a context with no actor, so if this stops working
// every protected route starts failing for a reason that names the use case
// rather than the middleware that did not run.
func TestRequireAuth_AttachesTheActor(t *testing.T) {
	t.Parallel()

	var (
		actor domain.Actor
		err   error
	)

	engine := gin.New()
	engine.GET("/thing",
		middleware.RequireAuth(nil, discardLogger()),
		func(c *gin.Context) {
			actor, err = domain.ActorFromContext(c.Request.Context())
			c.Status(http.StatusOK)
		},
	)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequestWithContext(
		t.Context(), http.MethodGet, "/thing", http.NoBody))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NoError(t, err)
	require.Equal(t, domain.API, actor.Channel)

	// The stub stands in for a signed-in person, and the deployment ledger
	// records the difference between a person and a machine.
	require.True(t, actor.IsHuman())
	require.NotNil(t, actor.UserID)
}
