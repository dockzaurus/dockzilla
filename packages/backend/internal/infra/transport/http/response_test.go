package http_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	dockzillahttp "dockzilla/internal/infra/transport/http"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// TestAbort_StopsTheChain is the reason Abort exists. A middleware that writes
// an error with c.JSON and returns still lets the route's own handler run, so
// the refusal turns into a success with two bodies in it.
func TestAbort_StopsTheChain(t *testing.T) {
	t.Parallel()

	engine := gin.New()
	handlerRan := false

	engine.GET("/thing",
		func(c *gin.Context) {
			dockzillahttp.Abort(c, http.StatusForbidden, "nope")
		},
		func(c *gin.Context) {
			handlerRan = true
			c.JSON(http.StatusOK, gin.H{"ok": true})
		},
	)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequestWithContext(
		t.Context(), http.MethodGet, "/thing", http.NoBody))

	require.False(t, handlerRan, "the aborted chain must not reach the handler")
	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.JSONEq(t, `{"error":"nope"}`, recorder.Body.String())
}

// TestFailing_RefusesEveryRequestWithoutLeakingTheCause pins both halves of
// the contract: the route stops working, and the reason it stopped working
// stays on the server.
func TestFailing_RefusesEveryRequestWithoutLeakingTheCause(t *testing.T) {
	t.Parallel()

	cause := errors.New("DOCKZILLA_SECRET_KEY is not a valid key")

	engine := gin.New()
	engine.GET("/thing",
		dockzillahttp.Failing(
			discardLogger(),
			http.StatusServiceUnavailable,
			"authentication is unavailable",
			cause,
		),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		},
	)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequestWithContext(
		t.Context(), http.MethodGet, "/thing", http.NoBody))

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, "authentication is unavailable", body["error"])
	require.NotContains(t, recorder.Body.String(), "DOCKZILLA_SECRET_KEY")
}

// TestFailing_IsSafeToCallRepeatedly guards the property that makes it a
// replacement for returning nil: it is an ordinary handler, so the router can
// invoke it as many times as requests arrive.
func TestFailing_IsSafeToCallRepeatedly(t *testing.T) {
	t.Parallel()

	engine := gin.New()
	engine.GET("/thing", dockzillahttp.Misconfigured(
		discardLogger(),
		errors.New("boom"),
	))

	for range 3 {
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, httptest.NewRequestWithContext(
			t.Context(), http.MethodGet, "/thing", http.NoBody))

		require.Equal(t, http.StatusInternalServerError, recorder.Code)
		require.JSONEq(t, `{"error":"the server is misconfigured"}`, recorder.Body.String())
	}
}
