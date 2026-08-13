package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dockzilla/internal/infra/transport/http/api"
	"dockzilla/internal/infra/transport/http/api/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSchemasRoutes(t *testing.T) {
	t.Parallel()

	type args struct {
		method string
		path   string
		body   string
	}
	tests := []struct {
		name     string
		args     args
		expect   func(h *mocks.MockSchemasHandler)
		wantCode int
	}{
		{
			name: "success - GET on the group lists",
			args: args{method: http.MethodGet, path: "/schema/registry"},
			expect: func(h *mocks.MockSchemasHandler) {
				h.EXPECT().List(mock.Anything).Run(func(c *gin.Context) {
					c.Status(http.StatusOK)
				}).Once()
			},
			wantCode: http.StatusOK,
		},
		{
			name: "success - a kind filter reaches the list handler",
			args: args{method: http.MethodGet, path: "/schema/registry?kind=app.stop"},
			expect: func(h *mocks.MockSchemasHandler) {
				h.EXPECT().List(mock.Anything).Run(func(c *gin.Context) {
					require.Equal(t, "app.stop", c.Query("kind"))
					c.Status(http.StatusOK)
				}).Once()
			},
			wantCode: http.StatusOK,
		},
		{
			name: "success - POST on the group registers",
			args: args{method: http.MethodPost, path: "/schema/registry", body: `{}`},
			expect: func(h *mocks.MockSchemasHandler) {
				h.EXPECT().Register(mock.Anything).Run(func(c *gin.Context) {
					c.Status(http.StatusCreated)
				}).Once()
			},
			wantCode: http.StatusCreated,
		},
		{
			name: "success - a bare kind reaches the latest handler",
			args: args{method: http.MethodGet, path: "/schema/registry/deployment.run"},
			expect: func(h *mocks.MockSchemasHandler) {
				h.EXPECT().RetrieveLatest(mock.Anything).Run(func(c *gin.Context) {
					require.Equal(t, "deployment.run", c.Param("kind"))
					c.Status(http.StatusOK)
				}).Once()
			},
			wantCode: http.StatusOK,
		},
		{
			name: "success - kind and version bind to the pinned handler",
			args: args{method: http.MethodGet, path: "/schema/registry/deployment.run/v1"},
			expect: func(h *mocks.MockSchemasHandler) {
				h.EXPECT().Retrieve(mock.Anything).Run(func(c *gin.Context) {
					require.Equal(t, "deployment.run", c.Param("kind"))
					require.Equal(t, "v1", c.Param("version"))
					c.Status(http.StatusOK)
				}).Once()
			},
			wantCode: http.StatusOK,
		},
		{
			name:     "error - unknown verb on a mounted path",
			args:     args{method: http.MethodDelete, path: "/schema/registry/deployment.run/v1"},
			expect:   func(*mocks.MockSchemasHandler) {},
			wantCode: http.StatusNotFound,
		},
		{
			name:     "error - a path deeper than a version is not mounted",
			args:     args{method: http.MethodGet, path: "/schema/registry/a/v1/extra"},
			expect:   func(*mocks.MockSchemasHandler) {},
			wantCode: http.StatusNotFound,
		},
		{
			name:     "error - path outside the registry group",
			args:     args{method: http.MethodGet, path: "/registry"},
			expect:   func(*mocks.MockSchemasHandler) {},
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := mocks.NewMockSchemasHandler(t)
			tt.expect(h)

			router := gin.New()
			api.SchemasRoutes(h)(router)

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequestWithContext(
				t.Context(), tt.args.method, tt.args.path, strings.NewReader(tt.args.body),
			))

			require.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestSchemasRoutes_MountUnderTheServerBasePath(t *testing.T) {
	t.Parallel()

	// The registry is addressed as /v1/schema/registry: "schema registry"
	// rather than "registry", which in this product means container images.
	h := mocks.NewMockSchemasHandler(t)
	h.EXPECT().Retrieve(mock.Anything).Run(func(c *gin.Context) {
		c.Status(http.StatusOK)
	}).Once()

	router := gin.New()
	api.SchemasRoutes(h)(router.Group("/v1"))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequestWithContext(
		t.Context(), http.MethodGet, "/v1/schema/registry/deployment.run/v1", nil,
	))

	require.Equal(t, http.StatusOK, rec.Code)
}
