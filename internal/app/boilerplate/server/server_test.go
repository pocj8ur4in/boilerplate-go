package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocj8ur4in/boilerplate-go/internal/app/boilerplate/server/middleware"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/database"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/jwt"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/redis"
)

func TestInit(t *testing.T) {
	t.Parallel()

	t.Run("init test server", func(t *testing.T) {
		t.Parallel()

		server := InitForTest(t)
		require.NotNil(t, server)
		require.NotNil(t, server.httpServer)
		require.NotNil(t, server.log)
		require.NotNil(t, server.registry)
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("new server with custom values", func(t *testing.T) {
		t.Parallel()

		config := Config{
			Host:            "0.0.0.0",
			Port:            9090,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			IdleTimeout:     120 * time.Second,
			ShutdownTimeout: 60 * time.Second,
			MaxRequestSize:  5242880,
		}

		server := newInstance(config)

		require.NotNil(t, server)
		assert.Equal(t, "0.0.0.0", server.Config.Host)
		assert.Equal(t, 9090, server.Config.Port)
		assert.Equal(t, 30*time.Second, server.Config.ReadTimeout)
	})
}

func TestSetup(t *testing.T) {
	t.Parallel()

	t.Run("setup server with custom values", func(t *testing.T) {
		t.Parallel()

		config := Config{
			Host: "localhost",
			Port: 8080,
			Metrics: middleware.MetricsConfig{
				Enabled: true,
				Path:    "/metrics",
			},
		}

		server := newInstance(config)

		log := logger.InitForTest(t)
		redisClient := redis.InitForTest(t)
		jwtClient := jwt.InitForTest(t)
		apiHandler := &MockAPIHandler{}

		dbClient := database.InitForTest(t)
		server.setup(apiHandler, log, dbClient, jwtClient, redisClient)

		assert.NotNil(t, server.log)
		assert.NotNil(t, server.httpServer)
	})
}

func TestSetupWithMiddlewares(t *testing.T) {
	t.Parallel()

	t.Run("setup server with all middlewares enabled", func(t *testing.T) {
		t.Parallel()

		config := Config{
			Host:           "localhost",
			Port:           8080,
			MaxRequestSize: 1048576,
			Compression: CompressionConfig{
				Enabled: true,
				Level:   6,
				Format:  "gzip",
			},
			RateLimit: middleware.RateLimitConfig{
				Global: middleware.RateLimitTypeConfig{
					Enabled:  true,
					Requests: 100,
					Window:   60,
				},
				IP: middleware.RateLimitTypeConfig{
					Enabled:  true,
					Requests: 10,
					Window:   60,
				},
				Endpoint: middleware.RateLimitTypeConfig{
					Enabled:  true,
					Requests: 5,
					Window:   60,
				},
			},
			Metrics: middleware.MetricsConfig{
				Enabled: true,
				Path:    "/metrics",
			},
			RequestThrottle: middleware.ThrottleConfig{
				MaxConcurrent: 10,
			},
		}

		server := newInstance(config)

		log := logger.InitForTest(t)
		apiHandler := &MockAPIHandler{}
		redisClient := redis.InitForTest(t)
		jwtClient := jwt.InitForTest(t)

		dbClient := database.InitForTest(t)
		server.setup(apiHandler, log, dbClient, jwtClient, redisClient)

		assert.NotNil(t, server.httpServer)
	})
}

func TestSetupRouter(t *testing.T) {
	t.Parallel()

	t.Run("setup router on server", func(t *testing.T) {
		t.Parallel()

		config := Config{
			Host: "localhost",
			Port: 8080,
			Metrics: middleware.MetricsConfig{
				Enabled: true,
				Path:    "/metrics",
			},
		}

		server := newInstance(config)

		log := logger.InitForTest(t)
		redisClient := redis.InitForTest(t)
		jwtClient := jwt.InitForTest(t)

		server.log = log
		server.registry = prometheus.NewRegistry()

		router := server.setupRouter(config, log, jwtClient, redisClient)

		require.NotNil(t, router)
	})
}

func TestSetupBasicMiddlewares(t *testing.T) {
	t.Parallel()

	t.Run("setup basic middlewares without compression", func(t *testing.T) {
		t.Parallel()

		config := Config{
			MaxRequestSize: 1048576,
			Compression: CompressionConfig{
				Enabled: false,
			},
			Metrics: middleware.MetricsConfig{
				Enabled: false,
			},
		}

		server := newInstance(config)

		log := logger.InitForTest(t)
		redisClient := redis.InitForTest(t)

		server.log = log
		server.registry = prometheus.NewRegistry()
		jwtClient := jwt.InitForTest(t)

		router := server.setupRouter(config, log, jwtClient, redisClient)

		require.NotNil(t, router)
	})

	t.Run("setup basic middlewares with compression", func(t *testing.T) {
		t.Parallel()

		config := Config{
			MaxRequestSize: 1048576,
			Compression: CompressionConfig{
				Enabled: true,
				Level:   6,
				Format:  "gzip",
			},
			Metrics: middleware.MetricsConfig{
				Enabled: true,
				Path:    "/metrics",
			},
		}

		server := newInstance(config)

		log := logger.InitForTest(t)
		redisClient := redis.InitForTest(t)

		server.log = log
		server.registry = prometheus.NewRegistry()
		jwtClient := jwt.InitForTest(t)

		router := server.setupRouter(config, log, jwtClient, redisClient)

		require.NotNil(t, router)
	})
}

func TestSetupRateLimitMiddlewares(t *testing.T) {
	t.Parallel()

	t.Run("setup all rate limit middlewares disabled", func(t *testing.T) {
		t.Parallel()

		config := Config{
			RateLimit: middleware.RateLimitConfig{
				Global: middleware.RateLimitTypeConfig{
					Enabled: false,
				},
				IP: middleware.RateLimitTypeConfig{
					Enabled: false,
				},
				Endpoint: middleware.RateLimitTypeConfig{
					Enabled: false,
				},
			},
		}

		server := newInstance(config)

		log := logger.InitForTest(t)
		redisClient := redis.InitForTest(t)
		jwtClient := jwt.InitForTest(t)

		router := server.setupRouter(config, log, jwtClient, redisClient)

		require.NotNil(t, router)
	})

	t.Run("setup all rate limit middlewares enabled", func(t *testing.T) {
		t.Parallel()

		config := Config{
			RateLimit: middleware.RateLimitConfig{
				Global: middleware.RateLimitTypeConfig{
					Enabled: true,
				},
				IP: middleware.RateLimitTypeConfig{
					Enabled: true,
				},
				Endpoint: middleware.RateLimitTypeConfig{
					Enabled: true,
				},
			},
		}

		server := newInstance(config)

		log := logger.InitForTest(t)
		redisClient := redis.InitForTest(t)
		jwtClient := jwt.InitForTest(t)

		router := server.setupRouter(config, log, jwtClient, redisClient)

		require.NotNil(t, router)
	})
}

func TestSetupRequestThrottleMiddlewares(t *testing.T) {
	t.Parallel()

	t.Run("setup request throttle middleware disabled", func(t *testing.T) {
		t.Parallel()

		config := Config{
			RequestThrottle: middleware.ThrottleConfig{
				MaxConcurrent: 0,
			},
		}

		server := newInstance(config)

		log := logger.InitForTest(t)
		redisClient := redis.InitForTest(t)
		jwtClient := jwt.InitForTest(t)

		router := server.setupRouter(config, log, jwtClient, redisClient)

		require.NotNil(t, router)
	})

	t.Run("setup request throttle middleware enabled", func(t *testing.T) {
		t.Parallel()

		config := Config{
			RequestThrottle: middleware.ThrottleConfig{
				MaxConcurrent: 10,
			},
		}

		server := newInstance(config)

		log := logger.InitForTest(t)
		redisClient := redis.InitForTest(t)
		jwtClient := jwt.InitForTest(t)

		router := server.setupRouter(config, log, jwtClient, redisClient)

		require.NotNil(t, router)
	})
}

func TestSetupCORS(t *testing.T) {
	t.Parallel()

	t.Run("setup CORS with default values", func(t *testing.T) {
		t.Parallel()

		config := Config{
			CORS: CORSConfig{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
				AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
				MaxAge:         300,
			},
		}

		server := newInstance(config)

		log := logger.InitForTest(t)
		redisClient := redis.InitForTest(t)
		jwtClient := jwt.InitForTest(t)

		router := server.setupRouter(config, log, jwtClient, redisClient)

		require.NotNil(t, router)
	})

	t.Run("setup CORS with custom values", func(t *testing.T) {
		t.Parallel()

		config := Config{
			CORS: CORSConfig{
				AllowedOrigins: []string{"https://example.com"},
				AllowedMethods: []string{"GET", "POST"},
				AllowedHeaders: []string{"Content-Type"},
				MaxAge:         600,
			},
		}

		server := newInstance(config)

		log := logger.InitForTest(t)
		redisClient := redis.InitForTest(t)
		jwtClient := jwt.InitForTest(t)

		router := server.setupRouter(config, log, jwtClient, redisClient)

		require.NotNil(t, router)
	})
}

func TestSetupMetricsEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("setup metrics endpoint enabled", func(t *testing.T) {
		t.Parallel()

		config := Config{
			Metrics: middleware.MetricsConfig{
				Enabled: true,
				Path:    "/metrics",
			},
		}

		server := newInstance(config)

		log := logger.InitForTest(t)
		redisClient := redis.InitForTest(t)

		server.log = log
		server.registry = prometheus.NewRegistry()
		jwtClient := jwt.InitForTest(t)

		router := server.setupRouter(config, log, jwtClient, redisClient)

		require.NotNil(t, router)
	})

	t.Run("setup metrics endpoint disabled", func(t *testing.T) {
		t.Parallel()

		config := Config{
			Metrics: middleware.MetricsConfig{
				Enabled: false,
			},
		}

		server := newInstance(config)

		log := logger.InitForTest(t)
		redisClient := redis.InitForTest(t)

		server.log = log
		server.registry = prometheus.NewRegistry()
		jwtClient := jwt.InitForTest(t)

		router := server.setupRouter(config, log, jwtClient, redisClient)

		require.NotNil(t, router)
	})
}

func TestSetupAPIHandler(t *testing.T) {
	t.Parallel()

	t.Run("setup API handler successfully", func(t *testing.T) {
		t.Parallel()

		config := Config{
			Host: "localhost",
			Port: 8080,
		}

		server := newInstance(config)

		log := logger.InitForTest(t)
		redisClient := redis.InitForTest(t)
		jwtClient := jwt.InitForTest(t)
		apiHandler := &MockAPIHandler{}

		router := server.setupRouter(config, log, jwtClient, redisClient)
		handler := server.setupAPIHandler(apiHandler, router)

		require.NotNil(t, handler)
	})
}

func TestCreateHTTPServer(t *testing.T) {
	t.Parallel()

	t.Run("create HTTP server with default values", func(t *testing.T) {
		t.Parallel()

		config := Config{
			Host:         "localhost",
			Port:         8080,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		server := newInstance(config)

		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		httpServer := server.createHTTPServer(config, handler)

		require.NotNil(t, httpServer)
		assert.Equal(t, "localhost:8080", httpServer.Addr)
		assert.Equal(t, 10*time.Second, httpServer.ReadTimeout)
		assert.Equal(t, 10*time.Second, httpServer.WriteTimeout)
		assert.Equal(t, 60*time.Second, httpServer.IdleTimeout)
	})
}

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("run server returns error when not initialized", func(t *testing.T) {
		t.Parallel()

		config := Config{
			Host: "localhost",
			Port: 8080,
		}

		server := newInstance(config)

		// try to run server without setup
		err := server.Run()

		require.Error(t, err)
		assert.Equal(t, ErrServerNotInitialized, err)
	})
}

func TestShutdown(t *testing.T) {
	t.Parallel()

	t.Run("shutdown server when not running", func(t *testing.T) {
		t.Parallel()

		config := Config{
			Host: "localhost",
			Port: 8080,
		}

		server := newInstance(config)

		log := logger.InitForTest(t)
		server.log = log

		ctx := context.Background()
		err := server.Shutdown(ctx)

		require.NoError(t, err)
	})

	t.Run("shutdown server after setup", func(t *testing.T) {
		t.Parallel()

		config := Config{
			Host: "localhost",
			Port: 8080,
		}

		server := newInstance(config)

		log := logger.InitForTest(t)
		redisClient := redis.InitForTest(t)
		jwtClient := jwt.InitForTest(t)
		apiHandler := &MockAPIHandler{}

		dbClient := database.InitForTest(t)
		server.setup(apiHandler, log, dbClient, jwtClient, redisClient)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := server.Shutdown(ctx)

		require.NoError(t, err)
	})
}

func TestIntegrationServerLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("full server lifecycle", func(t *testing.T) {
		t.Parallel()

		server := InitForTest(t)
		require.NotNil(t, server)

		// verify server is properly configured
		assert.NotNil(t, server.httpServer)
		assert.NotNil(t, server.log)
		assert.NotNil(t, server.registry)

		// shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := server.Shutdown(ctx)
		require.NoError(t, err)
	})

	t.Run("server with custom options", func(t *testing.T) {
		t.Parallel()

		server := InitForTest(t,
			WithTestHost("0.0.0.0"),
			WithTestPort(9090),
			WithTestReadTimeout(20),
			WithTestWriteTimeout(20),
		)

		require.NotNil(t, server)
		assert.Equal(t, "0.0.0.0", server.Config.Host)
		assert.Equal(t, 9090, server.Config.Port)
	})
}

func TestServerWithMiddlewares(t *testing.T) {
	t.Parallel()

	t.Run("server with all middlewares enabled", func(t *testing.T) {
		t.Parallel()

		rateLimitConfig := &middleware.RateLimitConfig{
			Global: middleware.RateLimitTypeConfig{
				Enabled:  true,
				Requests: 100,
				Window:   60,
			},
			IP: middleware.RateLimitTypeConfig{
				Enabled:  true,
				Requests: 10,
				Window:   60,
			},
		}

		server := InitForTest(t,
			WithTestCompression(6, "gzip", true),
			WithTestRateLimit(rateLimitConfig),
			WithTestMetrics(true, "/metrics", []string{"/health"}),
		)

		require.NotNil(t, server)
		assert.True(t, server.Config.Compression.Enabled)
		assert.True(t, server.Config.RateLimit.Global.Enabled)
		assert.True(t, server.Config.Metrics.Enabled)
	})
}

func TestServerHTTPRequests(t *testing.T) {
	t.Parallel()

	t.Run("server handles basic HTTP request", func(t *testing.T) {
		t.Parallel()

		server := InitForTest(t)

		req := httptest.NewRequest(http.MethodGet, "/status", nil)
		recorder := httptest.NewRecorder()

		server.httpServer.Handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("server handles metrics endpoint", func(t *testing.T) {
		t.Parallel()

		server := InitForTest(t,
			WithTestMetrics(true, "/test-metrics", nil),
		)

		req := httptest.NewRequest(http.MethodGet, "/test-metrics", nil)
		recorder := httptest.NewRecorder()

		server.httpServer.Handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestNewModule(t *testing.T) {
	t.Parallel()

	t.Run("create server module", func(t *testing.T) {
		t.Parallel()

		module := NewModule()

		require.NotNil(t, module)
	})
}
