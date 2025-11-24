package server

import (
	"net/http"
	"testing"
	"time"

	"github.com/pocj8ur4in/boilerplate-go/internal/app/boilerplate/server/middleware"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/database"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/jwt"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/redis"
)

const (
	// TestHost is the test host of the server.
	TestHost = "localhost"

	// TestPort is the test port of the server.
	TestPort = 38080

	// TestReadTimeout is the test read timeout of the server.
	TestReadTimeout = 15

	// TestWriteTimeout is the test write timeout of the server.
	TestWriteTimeout = 15

	// TestIdleTimeout is the test idle timeout of the server.
	TestIdleTimeout = 60

	// TestShutdownTimeout is the test shutdown timeout of the server.
	TestShutdownTimeout = 30

	// TestMaxRequestSize is the test max request size of the server.
	TestMaxRequestSize = int64(20971520) // 20MB

	// TestCompressionLevel is the test compression level of the server.
	TestCompressionLevel = 3

	// TestCompressionFormat is the test compression format of the server.
	TestCompressionFormat = "gzip"

	// TestCompressionEnabled is the test compression enabled of the server.
	TestCompressionEnabled = true

	// TestCORSAllowedOriginsStr is the test CORS allowed origins string of the server.
	TestCORSAllowedOriginsStr = "*"

	// TestCORSAllowedMethodsStr is the test CORS allowed methods string of the server.
	TestCORSAllowedMethodsStr = "GET,POST,PUT,PATCH,DELETE,OPTIONS"

	// TestCORSAllowedHeadersStr is the test CORS allowed headers string of the server.
	TestCORSAllowedHeadersStr = "Accept,Authorization,Content-Type,X-CSRF-Token"

	// TestCORSMaxAge is the test CORS max age of the server.
	TestCORSMaxAge = 300

	// TestGlobalRateLimitEnabled is the test global rate limit enabled of the server.
	TestGlobalRateLimitEnabled = true

	// TestGlobalRateLimitRequests is the test global rate limit requests of the server.
	TestGlobalRateLimitRequests = 100

	// TestGlobalRateLimitWindow is the test global rate limit window of the server.
	TestGlobalRateLimitWindow = 60

	// TestIPRateLimitEnabled is the test IP rate limit enabled of the server.
	TestIPRateLimitEnabled = true

	// TestIPRateLimitRequests is the test IP rate limit requests of the server.
	TestIPRateLimitRequests = 10

	// TestIPRateLimitWindow is the test IP rate limit window of the server.
	TestIPRateLimitWindow = 60

	// TestEndpointRateLimitEnabled is the test endpoint rate limit enabled of the server.
	TestEndpointRateLimitEnabled = true

	// TestEndpointRateLimitRequests is the test endpoint rate limit requests of the server.
	TestEndpointRateLimitRequests = 5

	// TestEndpointRateLimitWindow is the test endpoint rate limit window of the server.
	TestEndpointRateLimitWindow = 60

	// TestMetricsEnabled is the test metrics enabled of the server.
	TestMetricsEnabled = true

	// TestMetricsPath is the test metrics path of the server.
	TestMetricsPath = "/metrics"

	// TestMetricsExcludePathsStr is the test metrics exclude paths string of the server.
	TestMetricsExcludePathsStr = "/health,/status"
)

// MockAPIHandler is a mock implementation of api.ServerInterface.
type MockAPIHandler struct{}

// StatusCheck handles GET /status endpoint.
func (m *MockAPIHandler) StatusCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// HealthCheck handles GET /health endpoint.
func (m *MockAPIHandler) HealthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// HandleMetrics handles GET /metrics endpoint.
func (m *MockAPIHandler) HandleMetrics(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// TestServerOption modifies server config for testing.
type TestServerOption func(*Config)

// WithTestHost sets custom test host.
func WithTestHost(host string) TestServerOption {
	return func(c *Config) {
		c.Host = host
	}
}

// WithTestPort sets custom test port.
func WithTestPort(port int) TestServerOption {
	return func(c *Config) {
		c.Port = port
	}
}

// WithTestReadTimeout sets custom test read timeout.
func WithTestReadTimeout(timeout int) TestServerOption {
	return func(c *Config) {
		c.ReadTimeout = time.Duration(timeout) * time.Second
	}
}

// WithTestWriteTimeout sets custom test write timeout.
func WithTestWriteTimeout(timeout int) TestServerOption {
	return func(c *Config) {
		c.WriteTimeout = time.Duration(timeout) * time.Second
	}
}

// WithTestIdleTimeout sets custom test idle timeout.
func WithTestIdleTimeout(timeout int) TestServerOption {
	return func(c *Config) {
		c.IdleTimeout = time.Duration(timeout) * time.Second
	}
}

// WithTestShutdownTimeout sets custom test shutdown timeout.
func WithTestShutdownTimeout(timeout int) TestServerOption {
	return func(c *Config) {
		c.ShutdownTimeout = time.Duration(timeout) * time.Second
	}
}

// WithTestCompression sets custom test compression config.
func WithTestCompression(level int, format string, enabled bool) TestServerOption {
	return func(c *Config) {
		c.Compression = CompressionConfig{
			Level:   level,
			Format:  format,
			Enabled: enabled,
		}
	}
}

// WithTestCORS sets custom test CORS config.
func WithTestCORS(origins, methods, headers []string, maxAge int) TestServerOption {
	return func(c *Config) {
		c.CORS = CORSConfig{
			AllowedOrigins: origins,
			AllowedMethods: methods,
			AllowedHeaders: headers,
			MaxAge:         maxAge,
		}
	}
}

// WithTestRateLimit sets custom test rate limit config.
func WithTestRateLimit(config *middleware.RateLimitConfig) TestServerOption {
	return func(c *Config) {
		c.RateLimit = *config
	}
}

// WithTestMetrics sets custom test metrics config.
func WithTestMetrics(enabled bool, path string, excludePaths []string) TestServerOption {
	return func(c *Config) {
		c.Metrics = middleware.MetricsConfig{
			Enabled:      enabled,
			Path:         path,
			ExcludePaths: excludePaths,
		}
	}
}

// InitForTest initializes for testing.
//
//revive:disable:unexported-return // returns unexported type for testing purposes
func InitForTest(t *testing.T, opts ...TestServerOption) *client {
	t.Helper()

	// set server config
	config := Config{
		Host:            TestHost,
		Port:            TestPort,
		ReadTimeout:     TestReadTimeout,
		WriteTimeout:    TestWriteTimeout,
		IdleTimeout:     TestIdleTimeout,
		ShutdownTimeout: TestShutdownTimeout,
		MaxRequestSize:  TestMaxRequestSize,
		Compression: CompressionConfig{
			Level:   TestCompressionLevel,
			Format:  TestCompressionFormat,
			Enabled: TestCompressionEnabled,
		},
		CORS: CORSConfig{
			AllowedOrigins: []string{TestCORSAllowedOriginsStr},
			AllowedMethods: []string{TestCORSAllowedMethodsStr},
			AllowedHeaders: []string{TestCORSAllowedHeadersStr},
			MaxAge:         TestCORSMaxAge,
		},
		RateLimit: middleware.RateLimitConfig{
			Global: middleware.RateLimitTypeConfig{
				Enabled:  TestGlobalRateLimitEnabled,
				Requests: TestGlobalRateLimitRequests,
				Window:   TestGlobalRateLimitWindow,
			},
			IP: middleware.RateLimitTypeConfig{
				Enabled:  TestIPRateLimitEnabled,
				Requests: TestIPRateLimitRequests,
				Window:   TestIPRateLimitWindow,
			},
			Endpoint: middleware.RateLimitTypeConfig{
				Enabled:  TestEndpointRateLimitEnabled,
				Requests: TestEndpointRateLimitRequests,
				Window:   TestEndpointRateLimitWindow,
			},
		},
		Metrics: middleware.MetricsConfig{
			Enabled: TestMetricsEnabled,
			Path:    TestMetricsPath,
		},
	}

	// apply custom options
	for _, opt := range opts {
		opt(&config)
	}

	// initialize dependencies
	log := logger.InitForTest(t)
	dbClient := database.InitForTest(t)
	redisClient := redis.InitForTest(t)
	jwtClient := jwt.InitForTest(t)

	// create instance
	instance := newInstance(config)

	// setup server
	instance.setup(&MockAPIHandler{}, log, dbClient, jwtClient, redisClient)

	return instance
}
