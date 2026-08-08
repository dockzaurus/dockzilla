package handler_test

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"dockzilla/internal/core/sample"
	"dockzilla/internal/core/sample/mocks"
	"dockzilla/internal/infra/transport/http/handler"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func TestNewSample(t *testing.T) {
	t.Parallel()

	type args struct {
		opts func(service sample.Handler) []handler.Option
	}
	tests := []struct {
		name    string
		args    args
		wantErr string
	}{
		{
			name: "error - no options",
			args: args{
				opts: func(sample.Handler) []handler.Option { return nil },
			},
			wantErr: "sample handler: service is required",
		},
		{
			name: "error - logger without service",
			args: args{
				opts: func(sample.Handler) []handler.Option {
					return []handler.Option{handler.WithLogger(discardLogger())}
				},
			},
			wantErr: "sample handler: service is required",
		},
		{
			name: "error - service without logger",
			args: args{
				opts: func(service sample.Handler) []handler.Option {
					return []handler.Option{handler.WithHandler(service)}
				},
			},
			wantErr: "sample handler: logger is required",
		},
		{
			name: "success - every required option supplied",
			args: args{
				opts: func(service sample.Handler) []handler.Option {
					return []handler.Option{
						handler.WithHandler(service),
						handler.WithLogger(discardLogger()),
					}
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := mocks.NewMockHandler(t)

			h, err := handler.NewSample(tt.args.opts(service)...)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				require.Nil(t, h)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, h)
		})
	}
}

func TestSample_SayHello(t *testing.T) {
	t.Parallel()

	errUseCase := errors.New("greeting unavailable")

	type args struct {
		name string
	}
	tests := []struct {
		name     string
		args     args
		greeting string
		useCase  error
		wantCode int
		wantBody string
	}{
		{
			name:     "success - greeting is written back",
			args:     args{name: "killian"},
			greeting: "Hello, killian",
			wantCode: http.StatusOK,
			wantBody: `{"message":"Hello, killian"}`,
		},
		{
			name:     "success - empty path parameter reaches the use case",
			args:     args{name: ""},
			greeting: "Hello, ",
			wantCode: http.StatusOK,
			wantBody: `{"message":"Hello, "}`,
		},
		{
			name:     "error - use case failure is a 500 with no internals leaked",
			args:     args{name: "killian"},
			useCase:  errUseCase,
			wantCode: http.StatusInternalServerError,
			wantBody: `{"error":"internal server error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := mocks.NewMockHandler(t)
			service.EXPECT().
				SayHello(mock.Anything, tt.args.name).
				Return(tt.greeting, tt.useCase).
				Once()

			rec := httptest.NewRecorder()
			c := newContext(t, rec, http.MethodGet, tt.args.name)

			newSample(t, service).SayHello(c)

			require.Equal(t, tt.wantCode, rec.Code)
			require.JSONEq(t, tt.wantBody, rec.Body.String())
		})
	}
}

func TestSample_SendHello(t *testing.T) {
	t.Parallel()

	errUseCase := errors.New("delivery unavailable")

	type args struct {
		name string
	}
	tests := []struct {
		name     string
		args     args
		useCase  error
		wantCode int
		wantBody string
	}{
		{
			name:     "success - accepted",
			args:     args{name: "killian"},
			wantCode: http.StatusAccepted,
			wantBody: `{"message":"hello sent"}`,
		},
		{
			name:     "error - use case failure is a 500 with no internals leaked",
			args:     args{name: "killian"},
			useCase:  errUseCase,
			wantCode: http.StatusInternalServerError,
			wantBody: `{"error":"internal server error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := mocks.NewMockHandler(t)
			service.EXPECT().
				SendHello(mock.Anything, tt.args.name).
				Return(tt.useCase).
				Once()

			rec := httptest.NewRecorder()
			c := newContext(t, rec, http.MethodPost, tt.args.name)

			newSample(t, service).SendHello(c)

			require.Equal(t, tt.wantCode, rec.Code)
			require.JSONEq(t, tt.wantBody, rec.Body.String())
		})
	}
}

// newContext builds a gin context carrying name as the :name path parameter,
// as the router would when the route is mounted.
func newContext(
	t *testing.T,
	rec *httptest.ResponseRecorder,
	method string,
	name string,
) *gin.Context {
	t.Helper()

	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequestWithContext(t.Context(), method, "/sample/hello/"+name, nil)
	c.Params = gin.Params{{Key: "name", Value: name}}

	return c
}

func newSample(t *testing.T, service sample.Handler) *handler.Sample {
	t.Helper()

	h, err := handler.NewSample(
		handler.WithHandler(service),
		handler.WithLogger(discardLogger()),
	)
	require.NoError(t, err)

	return h
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
