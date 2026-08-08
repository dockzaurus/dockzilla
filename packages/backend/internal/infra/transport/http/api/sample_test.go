package api_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"dockzilla/internal/infra/transport/http/api"
	"dockzilla/internal/infra/transport/http/api/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func TestSampleRoutes(t *testing.T) {
	t.Parallel()

	type args struct {
		method string
		path   string
	}
	tests := []struct {
		name     string
		args     args
		expect   func(h *mocks.MockSampleHandler)
		wantCode int
	}{
		{
			name: "success - GET reaches SayHello with the name bound",
			args: args{method: http.MethodGet, path: "/sample/hello/killian"},
			expect: func(h *mocks.MockSampleHandler) {
				h.EXPECT().
					SayHello(mock.Anything).
					Run(func(c *gin.Context) {
						require.Equal(t, "killian", c.Param("name"))
						c.Status(http.StatusOK)
					}).
					Once()
			},
			wantCode: http.StatusOK,
		},
		{
			name: "success - POST reaches SendHello with the name bound",
			args: args{method: http.MethodPost, path: "/sample/hello/killian"},
			expect: func(h *mocks.MockSampleHandler) {
				h.EXPECT().
					SendHello(mock.Anything).
					Run(func(c *gin.Context) {
						require.Equal(t, "killian", c.Param("name"))
						c.Status(http.StatusAccepted)
					}).
					Once()
			},
			wantCode: http.StatusAccepted,
		},
		{
			name:     "error - the name segment is not optional",
			args:     args{method: http.MethodGet, path: "/sample/hello/"},
			expect:   func(*mocks.MockSampleHandler) {},
			wantCode: http.StatusNotFound,
		},
		{
			name:     "error - unknown verb on a mounted path",
			args:     args{method: http.MethodDelete, path: "/sample/hello/killian"},
			expect:   func(*mocks.MockSampleHandler) {},
			wantCode: http.StatusNotFound,
		},
		{
			name:     "error - path outside the sample group",
			args:     args{method: http.MethodGet, path: "/hello/killian"},
			expect:   func(*mocks.MockSampleHandler) {},
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := mocks.NewMockSampleHandler(t)
			tt.expect(h)

			router := gin.New()
			api.SampleRoutes(h)(router)

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequestWithContext(
				t.Context(), tt.args.method, tt.args.path, nil,
			))

			require.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestSampleRoutes_MountsUnderTheRouterItIsGiven(t *testing.T) {
	t.Parallel()

	h := mocks.NewMockSampleHandler(t)
	h.EXPECT().SayHello(mock.Anything).Run(func(c *gin.Context) {
		c.Status(http.StatusOK)
	}).Once()

	router := gin.New()
	// The transport mounts features under a base path, so the registration
	// must not hard-code the root.
	api.SampleRoutes(h)(router.Group("/v1"))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequestWithContext(
		t.Context(), http.MethodGet, "/v1/sample/hello/killian", nil,
	))

	require.Equal(t, http.StatusOK, rec.Code)
}
