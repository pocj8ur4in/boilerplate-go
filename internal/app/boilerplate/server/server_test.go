package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
)

// mockAPIHandler is a mock implementation of api.ServerInterface.
type mockAPIHandler struct{}

// StatusCheck handles GET /status endpoint.
func (m *mockAPIHandler) StatusCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// HealthCheck handles GET /health endpoint.
func (m *mockAPIHandler) HealthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// HandleMetrics handles GET /metrics endpoint.
func (m *mockAPIHandler) HandleMetrics(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestConfigSetDefault(t *testing.T) {
	t.Parallel()

	t.Run("set default server when config is empty", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		config.SetDefault()

		require.NotNil(t, config.Host)
		require.NotNil(t, config.Port)
		require.NotNil(t, config.ReadTimeout)
		require.NotNil(t, config.WriteTimeout)
		require.NotNil(t, config.IdleTimeout)
		require.NotNil(t, config.ShutdownTimeout)
		require.NotNil(t, config.MaxRequestSize)

		assert.Equal(t, "localhost", *config.Host)
		assert.Equal(t, 8080, *config.Port)
		assert.Equal(t, 10, *config.ReadTimeout)
		assert.Equal(t, 10, *config.WriteTimeout)
		assert.Equal(t, 10, *config.IdleTimeout)
		assert.Equal(t, 10, *config.ShutdownTimeout)
		assert.Equal(t, int64(10485760), *config.MaxRequestSize) // 10MB
	})

	t.Run("keep existing values when config is already set", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			Host:            &[]string{"0.0.0.0"}[0],
			Port:            &[]int{9090}[0],
			ReadTimeout:     &[]int{20}[0],
			WriteTimeout:    &[]int{30}[0],
			IdleTimeout:     &[]int{40}[0],
			ShutdownTimeout: &[]int{50}[0],
			MaxRequestSize:  &[]int64{20971520}[0],
		}

		config.SetDefault()

		require.NotNil(t, config.Host)
		require.NotNil(t, config.Port)
		require.NotNil(t, config.ReadTimeout)
		require.NotNil(t, config.WriteTimeout)
		require.NotNil(t, config.IdleTimeout)
		require.NotNil(t, config.ShutdownTimeout)
		require.NotNil(t, config.MaxRequestSize)

		assert.Equal(t, "0.0.0.0", *config.Host)
		assert.Equal(t, 9090, *config.Port)
		assert.Equal(t, 20, *config.ReadTimeout)
		assert.Equal(t, 30, *config.WriteTimeout)
		assert.Equal(t, 40, *config.IdleTimeout)
		assert.Equal(t, 50, *config.ShutdownTimeout)
		assert.Equal(t, int64(20971520), *config.MaxRequestSize)
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("create new server with nil config", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler)

		require.NoError(t, err)
		require.NotNil(t, server)
		require.NotNil(t, server.config)
		require.NotNil(t, server.logger)
		require.NotNil(t, server.httpServer)

		// check default values
		assert.Equal(t, "localhost", *server.config.Host)
		assert.Equal(t, 8080, *server.config.Port)
	})

	t.Run("create new server with custom config", func(t *testing.T) {
		t.Parallel()

		// create custom config
		config := &Config{
			Host: &[]string{"0.0.0.0"}[0],
			Port: &[]int{9090}[0],
		}

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		mockHandler := &mockAPIHandler{}
		server, err := New(config, log, mockHandler)

		require.NoError(t, err)
		require.NotNil(t, server)
		require.NotNil(t, server.config)
		require.NotNil(t, server.logger)
		require.NotNil(t, server.httpServer)

		// check custom values
		assert.Equal(t, "0.0.0.0", *server.config.Host)
		assert.Equal(t, 9090, *server.config.Port)
	})
}

func TestSetupRouter(t *testing.T) {
	t.Parallel()

	t.Run("setup router successfully", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler)
		require.NoError(t, err)

		router := server.setupRouter(server.config)

		require.NotNil(t, router)
	})
}

func TestSetupAPIHandler(t *testing.T) {
	t.Parallel()

	t.Run("setup API handler successfully", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler)
		require.NoError(t, err)

		router := server.setupRouter(server.config)
		handler := server.setupAPIHandler(mockHandler, router)

		require.NotNil(t, handler)
	})
}

func TestCreateHTTPServer(t *testing.T) {
	t.Parallel()

	t.Run("create HTTP server with default config", func(t *testing.T) {
		t.Parallel()

		config := &Config{}
		config.SetDefault()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		mockHandler := &mockAPIHandler{}
		server, err := New(config, log, mockHandler)
		require.NoError(t, err)

		router := server.setupRouter(config)
		handler := server.setupAPIHandler(mockHandler, router)
		httpServer := server.createHTTPServer(config, handler)

		require.NotNil(t, httpServer)
		assert.Equal(t, "localhost:8080", httpServer.Addr)
		assert.Equal(t, 10*time.Second, httpServer.ReadTimeout)
		assert.Equal(t, 10*time.Second, httpServer.WriteTimeout)
		assert.Equal(t, 10*time.Second, httpServer.IdleTimeout)
		assert.NotNil(t, httpServer.Handler)
	})

	t.Run("create HTTP server with custom config", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			Host:         &[]string{"0.0.0.0"}[0],
			Port:         &[]int{9090}[0],
			ReadTimeout:  &[]int{20}[0],
			WriteTimeout: &[]int{30}[0],
			IdleTimeout:  &[]int{40}[0],
		}
		config.SetDefault()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		mockHandler := &mockAPIHandler{}
		server, err := New(config, log, mockHandler)
		require.NoError(t, err)

		router := server.setupRouter(config)
		handler := server.setupAPIHandler(mockHandler, router)
		httpServer := server.createHTTPServer(config, handler)

		require.NotNil(t, httpServer)
		assert.Equal(t, "0.0.0.0:9090", httpServer.Addr)
		assert.Equal(t, 20*time.Second, httpServer.ReadTimeout)
		assert.Equal(t, 30*time.Second, httpServer.WriteTimeout)
		assert.Equal(t, 40*time.Second, httpServer.IdleTimeout)
		assert.NotNil(t, httpServer.Handler)
	})
}

func TestShutdown(t *testing.T) {
	t.Parallel()

	t.Run("shutdown server successfully", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// shutdown should succeed even if server is not running
		err = server.Shutdown(ctx)
		require.NoError(t, err)
	})
}

func TestNewModule(t *testing.T) {
	t.Parallel()

	t.Run("return fx.Option", func(t *testing.T) {
		t.Parallel()

		module := NewModule()

		require.NotNil(t, module)
	})
}

func TestServerInvalidEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("handle invalid endpoint", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler)
		require.NoError(t, err)

		// create test request for non-existent endpoint
		req := httptest.NewRequest(http.MethodGet, "/invalid", nil)
		recorder := httptest.NewRecorder()

		// serve the request
		server.httpServer.Handler.ServeHTTP(recorder, req)

		// verify response - should return 404
		assert.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

func TestServerHTTPMethods(t *testing.T) {
	t.Parallel()

	t.Run("test different HTTP methods on status endpoint", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler)
		require.NoError(t, err)

		methods := []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodPatch,
		}

		for _, method := range methods {
			t.Run(method, func(t *testing.T) {
				t.Parallel()

				req := httptest.NewRequest(method, "/status", nil)
				recorder := httptest.NewRecorder()

				server.httpServer.Handler.ServeHTTP(recorder, req)

				// status endpoint should only accept GET
				if method == http.MethodGet {
					assert.Equal(t, http.StatusOK, recorder.Code)
				} else {
					assert.NotEqual(t, http.StatusOK, recorder.Code)
				}
			})
		}
	})
}

func TestServerHandlerIntegration(t *testing.T) {
	t.Parallel()

	t.Run("verify handler is properly integrated", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler)
		require.NoError(t, err)

		// verify server components
		require.NotNil(t, server.httpServer)
		require.NotNil(t, server.httpServer.Handler)

		// test that the handler responds to requests
		req := httptest.NewRequest(http.MethodGet, "/status", nil)
		recorder := httptest.NewRecorder()

		server.httpServer.Handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("verify router is properly set up", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler)
		require.NoError(t, err)

		router := server.setupRouter(server.config)
		require.NotNil(t, router)

		// verify router can be used to create handler
		handler := server.setupAPIHandler(mockHandler, router)
		require.NotNil(t, handler)
	})
}

func TestServerConfiguration(t *testing.T) {
	t.Parallel()

	t.Run("server uses config values correctly", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			Host:         &[]string{"127.0.0.1"}[0],
			Port:         &[]int{3000}[0],
			ReadTimeout:  &[]int{5}[0],
			WriteTimeout: &[]int{5}[0],
			IdleTimeout:  &[]int{5}[0],
		}

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		mockHandler := &mockAPIHandler{}
		server, err := New(config, log, mockHandler)
		require.NoError(t, err)

		// verify config is applied to HTTP server
		assert.Equal(t, "127.0.0.1:3000", server.httpServer.Addr)
		assert.Equal(t, 5*time.Second, server.httpServer.ReadTimeout)
		assert.Equal(t, 5*time.Second, server.httpServer.WriteTimeout)
		assert.Equal(t, 5*time.Second, server.httpServer.IdleTimeout)
	})
}

func TestServerStatusEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("handle status endpoint", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler)
		require.NoError(t, err)

		// create test request
		req := httptest.NewRequest(http.MethodGet, "/status", nil)
		recorder := httptest.NewRecorder()

		// serve the request
		server.httpServer.Handler.ServeHTTP(recorder, req)

		// verify response
		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestServerHealthEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("handle health endpoint", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler)
		require.NoError(t, err)

		// create test request
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		recorder := httptest.NewRecorder()

		// serve the request
		server.httpServer.Handler.ServeHTTP(recorder, req)

		// verify response
		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestServerMetricsEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("handle metrics endpoint", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler)
		require.NoError(t, err)

		// create test request
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		recorder := httptest.NewRecorder()

		// serve the request
		server.httpServer.Handler.ServeHTTP(recorder, req)

		// verify response
		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}
