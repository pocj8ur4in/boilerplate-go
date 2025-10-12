package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocj8ur4in/boilerplate-go/internal/app/boilerplate/server/middleware"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/jwt"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/redis"
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

// setupTestRedis creates a test redis client.
func setupTestRedis(t *testing.T) *redis.Redis {
	t.Helper()

	password := ""
	db := 0
	redisConfig := &redis.Config{
		Addrs:    []string{"localhost:36379"},
		Password: &password,
		DB:       &db,
	}

	redisClient, err := redis.New(redisConfig)
	require.NoError(t, err)

	// flush DB to ensure clean state
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = redisClient.FlushDB(ctx).Err()
	require.NoError(t, err)

	return redisClient
}

// setupTestJWT creates a test JWT.
func setupTestJWT(t *testing.T) *jwt.JWT {
	t.Helper()

	secretKey := "test-secret-key"
	jwtConfig := &jwt.Config{
		SecretKey: &secretKey,
	}

	jwtService, err := jwt.New(jwtConfig)
	require.NoError(t, err)

	return jwtService
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

func TestConfigSetDefaultCompression(t *testing.T) {
	t.Parallel()

	t.Run("set default compression when config is empty", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		config.SetDefault()

		require.NotNil(t, config.Compression)
		require.NotNil(t, config.Compression.Level)
		require.NotNil(t, config.Compression.Format)
		require.NotNil(t, config.Compression.Enabled)

		assert.Equal(t, 6, *config.Compression.Level)
		assert.Equal(t, "gzip", *config.Compression.Format)
		assert.True(t, *config.Compression.Enabled)
	})
}

func TestConfigSetDefaultRateLimit(t *testing.T) {
	t.Parallel()

	t.Run("set default rate limit when config is empty", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		config.SetDefault()

		require.NotNil(t, config.RateLimit)
		require.NotNil(t, config.RateLimit.Global)
		require.NotNil(t, config.RateLimit.IP)
		require.NotNil(t, config.RateLimit.Endpoint)

		// verify global rate limit defaults
		require.NotNil(t, config.RateLimit.Global.Enabled)
		require.NotNil(t, config.RateLimit.Global.Requests)
		require.NotNil(t, config.RateLimit.Global.Window)
		assert.False(t, *config.RateLimit.Global.Enabled)
		assert.Equal(t, 1000, *config.RateLimit.Global.Requests)
		assert.Equal(t, 60, *config.RateLimit.Global.Window)

		// verify IP rate limit defaults
		require.NotNil(t, config.RateLimit.IP.Enabled)
		require.NotNil(t, config.RateLimit.IP.Requests)
		require.NotNil(t, config.RateLimit.IP.Window)
		assert.True(t, *config.RateLimit.IP.Enabled)
		assert.Equal(t, 100, *config.RateLimit.IP.Requests)
		assert.Equal(t, 60, *config.RateLimit.IP.Window)

		// verify endpoint rate limit defaults
		require.NotNil(t, config.RateLimit.Endpoint.Enabled)
		require.NotNil(t, config.RateLimit.Endpoint.Requests)
		require.NotNil(t, config.RateLimit.Endpoint.Window)
		assert.False(t, *config.RateLimit.Endpoint.Enabled)
		assert.Equal(t, 50, *config.RateLimit.Endpoint.Requests)
		assert.Equal(t, 60, *config.RateLimit.Endpoint.Window)
	})
}

func TestConfigSetDefaultCORS(t *testing.T) {
	t.Parallel()

	t.Run("set default CORS when config is empty", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		config.SetDefault()

		require.NotNil(t, config.CORS)
		require.NotNil(t, config.CORS.AllowedOrigins)
		require.NotNil(t, config.CORS.AllowedMethods)
		require.NotNil(t, config.CORS.AllowedHeaders)

		assert.Equal(t, []string{"*"}, *config.CORS.AllowedOrigins)
		assert.Equal(t, []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}, *config.CORS.AllowedMethods)
		assert.Equal(t, []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"}, *config.CORS.AllowedHeaders)
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("create new server with nil config", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		redisClient := setupTestRedis(t)
		jwtService := setupTestJWT(t)

		cfg := &Config{
			CORS: &CORSConfig{
				AllowedOrigins: &[]string{"http://localhost:3000"},
			},
		}

		mockHandler := &mockAPIHandler{}
		server, err := New(cfg, log, mockHandler, jwtService, redisClient)

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

		redisClient := setupTestRedis(t)
		jwtService := setupTestJWT(t)

		mockHandler := &mockAPIHandler{}
		server, err := New(config, log, mockHandler, jwtService, redisClient)

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

//nolint:paralleltest // sequential execution required to avoid prometheus registry conflicts
func TestSetupRouter(t *testing.T) {
	t.Run("setup router successfully", func(t *testing.T) {
		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		redisClient := setupTestRedis(t)
		jwtService := setupTestJWT(t)

		cfg := &Config{
			CORS: &CORSConfig{
				AllowedOrigins: &[]string{"http://localhost:3000"},
			},
		}

		mockHandler := &mockAPIHandler{}
		server, err := New(cfg, log, mockHandler, jwtService, redisClient)
		require.NoError(t, err)

		require.NotNil(t, server.httpServer)
		require.NotNil(t, server.httpServer.Handler)
	})
}

//nolint:paralleltest // sequential execution required to avoid prometheus registry conflicts
func TestSetupAPIHandler(t *testing.T) {
	t.Run("setup API handler successfully", func(t *testing.T) {
		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		redisClient := setupTestRedis(t)
		jwtService := setupTestJWT(t)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler, jwtService, redisClient)
		require.NoError(t, err)

		require.NotNil(t, server.httpServer)
		require.NotNil(t, server.httpServer.Handler)
	})
}

// verifyHTTPServer verifies the HTTP server configuration.
func verifyHTTPServer(
	t *testing.T,
	httpServer *http.Server,
	expectedAddr string,
	expectedReadTimeout time.Duration,
	expectedWriteTimeout time.Duration,
	expectedIdleTimeout time.Duration,
) {
	t.Helper()

	require.NotNil(t, httpServer)
	assert.Equal(t, expectedAddr, httpServer.Addr)
	assert.Equal(t, expectedReadTimeout, httpServer.ReadTimeout)
	assert.Equal(t, expectedWriteTimeout, httpServer.WriteTimeout)
	assert.Equal(t, expectedIdleTimeout, httpServer.IdleTimeout)
	assert.NotNil(t, httpServer.Handler)
}

//nolint:paralleltest // sequential execution required to avoid prometheus registry conflicts
func TestCreateHTTPServer(t *testing.T) {
	t.Run("create HTTP server with default config", func(t *testing.T) {
		config := &Config{}
		config.SetDefault()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		redisClient := setupTestRedis(t)
		jwtService := setupTestJWT(t)

		mockHandler := &mockAPIHandler{}
		server, err := New(config, log, mockHandler, jwtService, redisClient)
		require.NoError(t, err)

		verifyHTTPServer(t, server.httpServer, "localhost:8080",
			10*time.Second, 10*time.Second, 10*time.Second)
	})

	t.Run("create HTTP server with custom config", func(t *testing.T) {
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

		redisClient := setupTestRedis(t)
		jwtService := setupTestJWT(t)

		mockHandler := &mockAPIHandler{}
		server, err := New(config, log, mockHandler, jwtService, redisClient)
		require.NoError(t, err)

		verifyHTTPServer(t, server.httpServer, "0.0.0.0:9090",
			20*time.Second, 30*time.Second, 40*time.Second)
	})
}

func TestShutdown(t *testing.T) {
	t.Parallel()

	t.Run("shutdown server successfully", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		redisClient := setupTestRedis(t)
		jwtService := setupTestJWT(t)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler, jwtService, redisClient)
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

		redisClient := setupTestRedis(t)
		jwtService := setupTestJWT(t)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler, jwtService, redisClient)
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

		redisClient := setupTestRedis(t)
		jwtService := setupTestJWT(t)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler, jwtService, redisClient)
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

//nolint:paralleltest // sequential execution required to avoid prometheus registry conflicts
func TestServerHandlerIntegration(t *testing.T) {
	t.Run("verify handler is properly integrated", func(t *testing.T) {
		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		redisClient := setupTestRedis(t)
		jwtService := setupTestJWT(t)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler, jwtService, redisClient)
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
		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		redisClient := setupTestRedis(t)
		jwtService := setupTestJWT(t)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler, jwtService, redisClient)
		require.NoError(t, err)

		// verify server httpServer handler is set
		require.NotNil(t, server.httpServer)
		require.NotNil(t, server.httpServer.Handler)

		// test router works with different endpoints
		testCases := []struct {
			method string
			path   string
		}{
			{http.MethodGet, "/status"},
			{http.MethodGet, "/health"},
			{http.MethodGet, "/metrics"},
		}

		for _, tc := range testCases {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			recorder := httptest.NewRecorder()

			server.httpServer.Handler.ServeHTTP(recorder, req)

			// verify router handled the request
			assert.NotEqual(t, http.StatusNotFound, recorder.Code)
		}
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

		redisClient := setupTestRedis(t)
		jwtService := setupTestJWT(t)

		mockHandler := &mockAPIHandler{}
		server, err := New(config, log, mockHandler, jwtService, redisClient)
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

		redisClient := setupTestRedis(t)
		jwtService := setupTestJWT(t)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler, jwtService, redisClient)
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

		redisClient := setupTestRedis(t)
		jwtService := setupTestJWT(t)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler, jwtService, redisClient)
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

		redisClient := setupTestRedis(t)
		jwtService := setupTestJWT(t)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler, jwtService, redisClient)
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

func TestCompressionEnabled(t *testing.T) {
	t.Parallel()

	t.Run("compression is enabled by default", func(t *testing.T) {
		t.Parallel()

		config := &Config{}
		config.SetDefault()

		require.NotNil(t, config.Compression)
		require.NotNil(t, config.Compression.Enabled)
		assert.True(t, *config.Compression.Enabled)
	})
}

func TestCompressionLevel(t *testing.T) {
	t.Parallel()

	t.Run("compression has default level", func(t *testing.T) {
		t.Parallel()

		config := &Config{}
		config.SetDefault()

		require.NotNil(t, config.Compression)
		require.NotNil(t, config.Compression.Level)
		assert.Equal(t, 6, *config.Compression.Level)
	})
}

func TestCompressionFormat(t *testing.T) {
	t.Parallel()

	t.Run("compression has default format", func(t *testing.T) {
		t.Parallel()

		config := &Config{}
		config.SetDefault()

		require.NotNil(t, config.Compression)
		require.NotNil(t, config.Compression.Format)
		assert.Equal(t, "gzip", *config.Compression.Format)
	})
}

func TestCompressionCustomConfiguration(t *testing.T) {
	t.Parallel()

	t.Run("custom compression configuration", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			Compression: &CompressionConfig{
				Level:   &[]int{9}[0],
				Format:  &[]string{"deflate"}[0],
				Enabled: &[]bool{false}[0],
			},
		}
		config.SetDefault()

		require.NotNil(t, config.Compression)
		assert.Equal(t, 9, *config.Compression.Level)
		assert.Equal(t, "deflate", *config.Compression.Format)
		assert.False(t, *config.Compression.Enabled)
	})
}

func TestCompressionInResponse(t *testing.T) {
	t.Parallel()

	t.Run("compression is applied when enabled", func(t *testing.T) {
		t.Parallel()

		config := &Config{Compression: &CompressionConfig{Enabled: &[]bool{true}[0]}}

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		redisClient := setupTestRedis(t)
		jwtService := setupTestJWT(t)

		mockHandler := &mockAPIHandler{}
		server, err := New(config, log, mockHandler, jwtService, redisClient)
		require.NoError(t, err)

		// create test request with Accept-Encoding header
		req := httptest.NewRequest(http.MethodGet, "/status", nil)
		req.Header.Set("Accept-Encoding", "gzip")

		recorder := httptest.NewRecorder()

		// serve the request
		server.httpServer.Handler.ServeHTTP(recorder, req)

		// verify response - compression middleware is applied
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("compression is not applied when disabled", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			Compression: &CompressionConfig{
				Enabled: &[]bool{false}[0],
			},
		}

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		redisClient := setupTestRedis(t)
		jwtService := setupTestJWT(t)

		mockHandler := &mockAPIHandler{}
		server, err := New(config, log, mockHandler, jwtService, redisClient)
		require.NoError(t, err)

		// create test request with Accept-Encoding header
		req := httptest.NewRequest(http.MethodGet, "/status", nil)
		req.Header.Set("Accept-Encoding", "gzip")

		recorder := httptest.NewRecorder()

		// serve the request
		server.httpServer.Handler.ServeHTTP(recorder, req)

		// verify response
		assert.Equal(t, http.StatusOK, recorder.Code)

		// Content-Encoding should not be set when compression is disabled
		assert.Empty(t, recorder.Header().Get("Content-Encoding"))
	})
}

// verifyRateLimitConfig verifies the rate limit type config.
func verifyRateLimitConfig(
	t *testing.T,
	config *middleware.RateLimitTypeConfig,
	expectedEnabled bool,
	expectedRequests int,
	expectedWindow int,
) {
	t.Helper()

	require.NotNil(t, config)
	require.NotNil(t, config.Enabled)
	require.NotNil(t, config.Requests)
	require.NotNil(t, config.Window)

	assert.Equal(t, expectedEnabled, *config.Enabled)
	assert.Equal(t, expectedRequests, *config.Requests)
	assert.Equal(t, expectedWindow, *config.Window)
}

func TestRateLimitGlobalDefault(t *testing.T) {
	t.Parallel()

	t.Run("global rate limit has default values", func(t *testing.T) {
		t.Parallel()

		config := &Config{}
		config.SetDefault()

		require.NotNil(t, config.RateLimit)
		verifyRateLimitConfig(t, config.RateLimit.Global, false, 1000, 60)
	})
}

func TestRateLimitIPDefault(t *testing.T) {
	t.Parallel()

	t.Run("IP rate limit has default values", func(t *testing.T) {
		t.Parallel()

		config := &Config{}
		config.SetDefault()

		require.NotNil(t, config.RateLimit)
		verifyRateLimitConfig(t, config.RateLimit.IP, true, 100, 60)
	})
}

func TestRateLimitEndpointDefault(t *testing.T) {
	t.Parallel()

	t.Run("endpoint rate limit has default values", func(t *testing.T) {
		t.Parallel()

		config := &Config{}
		config.SetDefault()

		require.NotNil(t, config.RateLimit)
		verifyRateLimitConfig(t, config.RateLimit.Endpoint, false, 50, 60)
	})
}

func TestRateLimitCustomConfiguration(t *testing.T) {
	t.Parallel()

	t.Run("custom rate limit configuration", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			RateLimit: &middleware.RateLimitConfig{
				Global: &middleware.RateLimitTypeConfig{
					Enabled:  &[]bool{true}[0],
					Requests: &[]int{500}[0],
					Window:   &[]int{30}[0],
				},
				IP: &middleware.RateLimitTypeConfig{
					Enabled:  &[]bool{false}[0],
					Requests: &[]int{50}[0],
					Window:   &[]int{120}[0],
				},
				Endpoint: &middleware.RateLimitTypeConfig{
					Enabled:  &[]bool{true}[0],
					Requests: &[]int{25}[0],
					Window:   &[]int{90}[0],
				},
			},
		}
		config.SetDefault()

		require.NotNil(t, config.RateLimit)

		// verify global settings
		assert.True(t, *config.RateLimit.Global.Enabled)
		assert.Equal(t, 500, *config.RateLimit.Global.Requests)
		assert.Equal(t, 30, *config.RateLimit.Global.Window)

		// verify IP settings
		assert.False(t, *config.RateLimit.IP.Enabled)
		assert.Equal(t, 50, *config.RateLimit.IP.Requests)
		assert.Equal(t, 120, *config.RateLimit.IP.Window)

		// verify endpoint settings
		assert.True(t, *config.RateLimit.Endpoint.Enabled)
		assert.Equal(t, 25, *config.RateLimit.Endpoint.Requests)
		assert.Equal(t, 90, *config.RateLimit.Endpoint.Window)
	})
}

func TestCORSDefaultAllowedOrigins(t *testing.T) {
	t.Parallel()

	t.Run("CORS has default allowed origins", func(t *testing.T) {
		t.Parallel()

		config := &Config{}
		config.SetDefault()

		require.NotNil(t, config.CORS)
		require.NotNil(t, config.CORS.AllowedOrigins)
		assert.Equal(t, []string{"*"}, *config.CORS.AllowedOrigins)
	})
}

func TestCORSAllowedMethods(t *testing.T) {
	t.Parallel()

	t.Run("CORS has default allowed methods", func(t *testing.T) {
		t.Parallel()

		config := &Config{}
		config.SetDefault()

		require.NotNil(t, config.CORS)
		require.NotNil(t, config.CORS.AllowedMethods)
		assert.Equal(t, []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}, *config.CORS.AllowedMethods)
	})
}

func TestCORSAllowedHeaders(t *testing.T) {
	t.Parallel()

	t.Run("CORS has default allowed headers", func(t *testing.T) {
		t.Parallel()

		config := &Config{}
		config.SetDefault()

		require.NotNil(t, config.CORS)
		require.NotNil(t, config.CORS.AllowedHeaders)
		assert.Equal(t, []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"}, *config.CORS.AllowedHeaders)
	})
}

func TestCORSCustomConfiguration(t *testing.T) {
	t.Parallel()

	t.Run("custom CORS configuration", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			CORS: &CORSConfig{
				AllowedOrigins: &[]string{"https://example.com"},
				AllowedMethods: &[]string{"GET", "POST"},
				AllowedHeaders: &[]string{"Content-Type"},
			},
		}
		config.SetDefault()

		require.NotNil(t, config.CORS)
		assert.Equal(t, []string{"https://example.com"}, *config.CORS.AllowedOrigins)
		assert.Equal(t, []string{"GET", "POST"}, *config.CORS.AllowedMethods)
		assert.Equal(t, []string{"Content-Type"}, *config.CORS.AllowedHeaders)
	})
}

func TestCORSHeaders(t *testing.T) {
	t.Parallel()

	t.Run("CORS headers are set in response", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		redisClient := setupTestRedis(t)
		jwtService := setupTestJWT(t)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler, jwtService, redisClient)
		require.NoError(t, err)

		// create test request with Origin header
		req := httptest.NewRequest(http.MethodGet, "/status", nil)
		req.Header.Set("Origin", "http://localhost:3000")

		recorder := httptest.NewRecorder()

		// serve the request
		server.httpServer.Handler.ServeHTTP(recorder, req)

		// verify CORS headers are present
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.NotEmpty(t, recorder.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("CORS handles preflight requests", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		redisClient := setupTestRedis(t)
		jwtService := setupTestJWT(t)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler, jwtService, redisClient)
		require.NoError(t, err)

		// create preflight request
		req := httptest.NewRequest(http.MethodOptions, "/status", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "GET")

		recorder := httptest.NewRecorder()

		// serve the request
		server.httpServer.Handler.ServeHTTP(recorder, req)

		// verify preflight response is successful
		assert.Contains(t, []int{http.StatusOK, http.StatusNoContent}, recorder.Code)
	})
}

// createTestServerWithCORS creates a test server with CORS config.
func createTestServerWithCORS(
	t *testing.T,
	config *Config,
) *Server {
	t.Helper()

	log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
	require.NoError(t, err)

	redisClient := setupTestRedis(t)
	jwtService := setupTestJWT(t)

	mockHandler := &mockAPIHandler{}
	server, err := New(config, log, mockHandler, jwtService, redisClient)
	require.NoError(t, err)

	return server
}

// makeRequestAndVerifyCORS makes a request and verifies CORS.
func makeRequestAndVerifyCORS(
	t *testing.T,
	server *Server,
	origin string,
	expectedOrigin string,
) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	req.Header.Set("Origin", origin)

	recorder := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(recorder, req)

	if expectedOrigin != "" {
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, expectedOrigin, recorder.Header().Get("Access-Control-Allow-Origin"))
	} else {
		assert.Empty(t, recorder.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSCustomOrigins(t *testing.T) {
	t.Parallel()

	t.Run("custom CORS origins are respected", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			CORS: &CORSConfig{
				AllowedOrigins: &[]string{"https://example.com"},
				AllowedMethods: &[]string{"GET", "POST"},
				AllowedHeaders: &[]string{"Content-Type"},
			},
		}

		server := createTestServerWithCORS(t, config)
		makeRequestAndVerifyCORS(t, server, "https://example.com", "https://example.com")
	})

	t.Run("disallowed origin is rejected", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			CORS: &CORSConfig{
				AllowedOrigins: &[]string{"https://example.com"},
				AllowedMethods: &[]string{"GET", "POST"},
				AllowedHeaders: &[]string{"Content-Type"},
			},
		}

		server := createTestServerWithCORS(t, config)
		makeRequestAndVerifyCORS(t, server, "https://evil.com", "")
	})
}

func TestServerJWTIntegration(t *testing.T) {
	t.Parallel()

	t.Run("JWT service is properly integrated", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		redisClient := setupTestRedis(t)
		jwtService := setupTestJWT(t)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler, jwtService, redisClient)
		require.NoError(t, err)

		require.NotNil(t, server)
		require.NotNil(t, server.httpServer)
	})

	t.Run("JWT middleware is applied to server", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		redisClient := setupTestRedis(t)
		jwtService := setupTestJWT(t)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler, jwtService, redisClient)
		require.NoError(t, err)

		require.NotNil(t, server.httpServer.Handler)
	})
}

func TestServerWithDifferentJWTConfig(t *testing.T) {
	t.Parallel()

	t.Run("create server with custom JWT secret", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		redisClient := setupTestRedis(t)

		secretKey := "custom-secret-key"
		jwtConfig := &jwt.Config{
			SecretKey: &secretKey,
		}
		jwtService, err := jwt.New(jwtConfig)
		require.NoError(t, err)

		mockHandler := &mockAPIHandler{}
		server, err := New(nil, log, mockHandler, jwtService, redisClient)
		require.NoError(t, err)

		require.NotNil(t, server)
		require.NotNil(t, server.httpServer)
	})
}

func TestServerSetupWithAllComponents(t *testing.T) {
	t.Parallel()

	t.Run("setup server with all components", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			Host:         &[]string{"localhost"}[0],
			Port:         &[]int{8080}[0],
			ReadTimeout:  &[]int{10}[0],
			WriteTimeout: &[]int{10}[0],
			IdleTimeout:  &[]int{10}[0],
			Compression: &CompressionConfig{
				Enabled: &[]bool{true}[0],
				Level:   &[]int{6}[0],
				Format:  &[]string{"gzip"}[0],
			},
			CORS: &CORSConfig{
				AllowedOrigins: &[]string{"*"},
			},
			RateLimit: &middleware.RateLimitConfig{
				IP: &middleware.RateLimitTypeConfig{
					Enabled:  &[]bool{true}[0],
					Requests: &[]int{100}[0],
					Window:   &[]int{60}[0],
				},
			},
		}

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		redisClient := setupTestRedis(t)
		jwtService := setupTestJWT(t)

		mockHandler := &mockAPIHandler{}
		server, err := New(config, log, mockHandler, jwtService, redisClient)
		require.NoError(t, err)

		require.NotNil(t, server)
		require.NotNil(t, server.config)
		require.NotNil(t, server.logger)
		require.NotNil(t, server.httpServer)
		require.NotNil(t, server.httpServer.Handler)
	})
}
