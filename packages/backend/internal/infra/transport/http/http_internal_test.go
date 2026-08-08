package http

// The gin engine the routes are mounted on is reachable only through the
// unexported server handle, so where a registration lands is asserted from
// inside the package. Serving through the engine keeps this a unit test: Run
// binds a port, which belongs in an e2e test.

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	ginihttp "github.com/zixyos/giniservice/http"
)

func TestServer_MountsRoutesUnderTheBasePath(t *testing.T) {
	t.Parallel()

	type args struct {
		basePath string
		request  string
	}
	tests := []struct {
		name     string
		args     args
		wantCode int
	}{
		{
			name:     "success - mounted at the root when no base path is set",
			args:     args{basePath: "", request: "/ping"},
			wantCode: http.StatusOK,
		},
		{
			name:     "success - mounted under the base path",
			args:     args{basePath: "/v1", request: "/v1/ping"},
			wantCode: http.StatusOK,
		},
		{
			name:     "error - the base path is not optional once set",
			args:     args{basePath: "/v1", request: "/ping"},
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &ginihttp.Config{ServiceName: "dockzilla-http"}

			s, err := NewServer(
				WithLogger(slog.New(slog.DiscardHandler)),
				WithConfig(cfg),
				WithBasePath(tt.args.basePath),
				WithRoutes(func(router gin.IRouter) {
					router.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })
				}),
			)
			require.NoError(t, err)

			rec := httptest.NewRecorder()
			s.srv.Engine().ServeHTTP(rec, httptest.NewRequestWithContext(
				t.Context(), http.MethodGet, tt.args.request, nil,
			))

			require.Equal(t, tt.wantCode, rec.Code)
		})
	}
}
