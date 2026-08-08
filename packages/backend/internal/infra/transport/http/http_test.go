package http_test

import (
	"log/slog"
	"net/http"
	"os"
	"testing"

	dockzillahttp "dockzilla/internal/infra/transport/http"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	ginihttp "github.com/zixyos/giniservice/http"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func TestNewServer(t *testing.T) {
	t.Parallel()

	type args struct {
		opts []dockzillahttp.Option
	}
	tests := []struct {
		name     string
		args     args
		wantErr  string
		wantName string
	}{
		{
			name:    "error - no options",
			args:    args{},
			wantErr: "http server: logger is required",
		},
		{
			name: "error - logger without config",
			args: args{opts: []dockzillahttp.Option{
				dockzillahttp.WithLogger(discardLogger()),
			}},
			wantErr: "http server: config is required",
		},
		{
			name: "success - every required option supplied",
			args: args{opts: []dockzillahttp.Option{
				dockzillahttp.WithLogger(discardLogger()),
				dockzillahttp.WithConfig(newConfig("dockzilla-http")),
			}},
			wantName: "dockzilla-http",
		},
		{
			name: "success - name comes from the config",
			args: args{opts: []dockzillahttp.Option{
				dockzillahttp.WithLogger(discardLogger()),
				dockzillahttp.WithConfig(newConfig("custom")),
				dockzillahttp.WithBasePath("/v1"),
			}},
			wantName: "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv, err := dockzillahttp.NewServer(tt.args.opts...)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				require.Nil(t, srv)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, srv)
			require.Equal(t, tt.wantName, srv.Name())
		})
	}
}

func TestNewServer_RoutesRunAtConstruction(t *testing.T) {
	t.Parallel()

	var mounted bool

	_, err := dockzillahttp.NewServer(
		dockzillahttp.WithLogger(discardLogger()),
		dockzillahttp.WithConfig(newConfig("dockzilla-http")),
		dockzillahttp.WithRoutes(func(router gin.IRouter) {
			mounted = true

			router.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })
		}),
	)

	require.NoError(t, err)
	require.True(t, mounted, "registrations run at construction, not on Run")
}

func TestNewServer_RoutesAccumulate(t *testing.T) {
	t.Parallel()

	var mounted []string

	register := func(name string) dockzillahttp.RouteRegistration {
		return func(gin.IRouter) { mounted = append(mounted, name) }
	}

	_, err := dockzillahttp.NewServer(
		dockzillahttp.WithLogger(discardLogger()),
		dockzillahttp.WithConfig(newConfig("dockzilla-http")),
		dockzillahttp.WithRoutes(register("first"), register("second")),
		dockzillahttp.WithRoutes(register("third")),
	)

	require.NoError(t, err)
	// WithRoutes can be passed more than once, and registrations mount in the
	// order they were given.
	require.Equal(t, []string{"first", "second", "third"}, mounted)
}

func newConfig(serviceName string) *ginihttp.Config {
	cfg := &ginihttp.Config{ServiceName: serviceName}
	cfg.HTTPServer.Port = 0

	return cfg
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
