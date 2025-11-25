// Package server provides http server.
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/fx"

	"github.com/pocj8ur4in/boilerplate-go/internal/app/boilerplate/server/middleware"
	genAPI "github.com/pocj8ur4in/boilerplate-go/internal/gen/api"
	genDB "github.com/pocj8ur4in/boilerplate-go/internal/gen/db"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/database"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/jwt"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/redis"
)

var (
	// ErrServerNotInitialized is returned when the http server is not initialized.
	ErrServerNotInitialized = errors.New("http server is not initialized")
)

// Client defines the interface for server operations.
type Client interface {
	// Run runs the HTTP server.
	Run() error

	// Shutdown gracefully shuts down the HTTP server.
	Shutdown(ctx context.Context) error
}

// client implements server.Client.
type client struct {
	// Config provides server configuration.
	Config Config

	// log provides logger.
	log logger.Client

	// httpServer provides http server.
	httpServer *http.Server

	// queries provides database queries client.
	queries *genDB.Queries

	// registry provides prometheus registry for metrics.
	registry *prometheus.Registry
}

// Config represents configuration for server.
type Config struct {
	// Host is host of server.
	Host string `env:"HOST" envDefault:"localhost" json:"host"`

	// Port is port of server.
	Port int `env:"PORT" envDefault:"8080" json:"port"`

	// ReadTimeout is read timeout of server.
	ReadTimeout time.Duration `env:"READ_TIMEOUT" envDefault:"10s" json:"read_timeout"`

	// WriteTimeout is write timeout of server.
	WriteTimeout time.Duration `env:"WRITE_TIMEOUT" envDefault:"10s" json:"write_timeout"`

	// IdleTimeout is idle timeout of server.
	IdleTimeout time.Duration `env:"IDLE_TIMEOUT" envDefault:"10s" json:"idle_timeout"`

	// ShutdownTimeout is shutdown timeout of server.
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s" json:"shutdown_timeout"`

	// MaxRequestSize is maximum request size in bytes.
	MaxRequestSize int64 `env:"MAX_REQUEST_SIZE" envDefault:"10485760" json:"max_request_size"`

	// Compression is compression configuration of server.
	Compression CompressionConfig `envPrefix:"COMPRESSION_" json:"compression"`

	// CORS is CORS of server.
	CORS CORSConfig `envPrefix:"CORS_" json:"cors"`

	// RateLimit is rate limit of server.
	RateLimit middleware.RateLimitConfig `envPrefix:"RATE_LIMIT_" json:"rate_limit"`

	// Metrics is metrics configuration of server.
	Metrics middleware.MetricsConfig `envPrefix:"METRICS_" json:"metrics"`

	// RequestThrottle is request throttling configuration of server.
	RequestThrottle middleware.ThrottleConfig `envPrefix:"REQUEST_THROTTLE_" json:"request_throttle"`
}

// CompressionConfig represents configuration for compression.
type CompressionConfig struct {
	// Level is compression level (1-9).
	Level int `env:"LEVEL" envDefault:"6" json:"level"`

	// Format is compression format (gzip, deflate, br).
	Format string `env:"FORMAT" envDefault:"gzip" json:"format"`

	// Enabled is whether compression is enabled.
	Enabled bool `env:"ENABLED" envDefault:"true" json:"enabled"`
}

// CORSConfig represents configuration for CORS.
type CORSConfig struct {
	// AllowedOrigins is allowed origins on CORS.
	AllowedOrigins []string `env:"ALLOWED_ORIGINS" envDefault:"*" json:"allowed_origins"`

	// AllowedMethods is allowed methods on CORS.
	AllowedMethods []string `env:"ALLOWED_METHODS" envDefault:"GET,POST,PUT,PATCH,DELETE,OPTIONS" json:"allowed_methods"`

	// AllowedHeaders is allowed headers on CORS.
	AllowedHeaders []string `env:"ALLOWED_HEADERS" envDefault:"Accept,Authorization,Content-Type,X-CSRF-Token" json:"allowed_headers"` //nolint:lll // line length needed for struct tag

	// MaxAge is max age on CORS.
	MaxAge int `env:"MAX_AGE" envDefault:"300" json:"max_age"`
}

// ConstructorParams represents parameters for constructor.
type ConstructorParams struct {
	fx.In

	// Config provides server configuration.
	Config Config

	// log provides logger client.
	Log logger.Client

	// Database provides database client.
	Database database.Client

	// JWT provides JWT client.
	JWT jwt.Client

	// Redis provides redis client.
	Redis redis.Client

	// Handler provides API handler.
	Handler genAPI.ServerInterface
}

// NewModule provides module for server.
func NewModule() fx.Option {
	return fx.Module("server",
		// provide concrete type for constructor
		fx.Provide(func(params ConstructorParams) (*client, error) {
			// create instance
			instance := newInstance(params.Config)

			// setup instance
			instance.setup(params.Handler, params.Log, params.Database, params.JWT, params.Redis)

			return instance, nil
		}),
		// provide interface type for dependency injection
		fx.Provide(fx.Annotate(
			func(instance *client) Client {
				return instance
			},
		)),
	)
}

// newInstance creates new server instance.
func newInstance(config Config) *client {
	return &client{Config: config}
}

// Setup sets up the server with dependencies.
func (c *client) setup(
	apiHandler genAPI.ServerInterface,
	log logger.Client,
	database database.Client,
	jwt jwt.Client,
	redis redis.Client,
) {
	// inject outer dependencies
	c.log = log
	c.queries = genDB.New(database.GetPool())
	c.registry = prometheus.NewRegistry()

	// setup router and handlers
	router := c.setupRouter(c.Config, log, jwt, redis)
	httpHandler := c.setupAPIHandler(apiHandler, router)
	c.httpServer = c.createHTTPServer(c.Config, httpHandler)
}

// setupRouter sets up the router.
func (c *client) setupRouter(config Config, log logger.Client, jwt jwt.Client, redis redis.Client) *chi.Mux {
	router := chi.NewRouter()

	c.setupBasicMiddlewares(config, log, jwt, router)
	c.setupRateLimitMiddlewares(config, log, redis, router)
	c.setupRequestThrottleMiddlewares(config, log, redis, router)
	c.setupCORS(config, router)
	c.setupMetricsEndpoint(config, router)

	return router
}

// setupBasicMiddlewares sets up basic middlewares.
func (c *client) setupBasicMiddlewares(config Config, log logger.Client, jwt jwt.Client, router *chi.Mux) {
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.RequestSize(config.MaxRequestSize))

	if config.Compression.Enabled {
		router.Use(middleware.Compress(config.Compression.Level, config.Compression.Format))
	}

	if config.Metrics.Enabled {
		router.Use(middleware.Metrics(config.Metrics, c.registry))
	}

	router.Use(middleware.InjectLogger(c.log))
	router.Use(middleware.LogRequest(c.log))
	router.Use(middleware.Timeout(config.ReadTimeout))

	router.Use(middleware.JwtAuth(log, jwt))
}

// setupRateLimitMiddlewares sets up rate limit middlewares.
func (c *client) setupRateLimitMiddlewares(
	config Config,
	log logger.Client,
	redis redis.Client,
	router *chi.Mux,
) {
	if config.RateLimit.Global.Enabled {
		window := time.Duration(config.RateLimit.Global.Window) * time.Second

		router.Use(middleware.GlobalRateLimit(
			config.RateLimit.Global.Requests,
			window,
			log,
			redis,
		))
	}

	if config.RateLimit.IP.Enabled {
		window := time.Duration(config.RateLimit.IP.Window) * time.Second

		router.Use(middleware.IPRateLimit(
			config.RateLimit.IP.Requests,
			window,
			log,
			redis,
		))
	}

	if config.RateLimit.Endpoint.Enabled {
		window := time.Duration(config.RateLimit.Endpoint.Window) * time.Second

		router.Use(middleware.EndpointRateLimit(
			config.RateLimit.Endpoint.Requests,
			window,
			log,
			redis,
		))
	}
}

// setupRequestThrottleMiddlewares sets up request throttling middlewares.
func (c *client) setupRequestThrottleMiddlewares(
	config Config,
	log logger.Client,
	redis redis.Client,
	router *chi.Mux,
) {
	if config.RequestThrottle.MaxConcurrent > 0 {
		router.Use(middleware.GlobalRequestThrottle(
			config.RequestThrottle,
			log,
			redis,
		))
	}
}

// setupCORS sets up CORS handler on router.
func (c *client) setupCORS(config Config, router *chi.Mux) {
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   config.CORS.AllowedOrigins,
		AllowedMethods:   config.CORS.AllowedMethods,
		AllowedHeaders:   config.CORS.AllowedHeaders,
		MaxAge:           config.CORS.MaxAge,
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
	}))
}

// setupMetricsEndpoint sets up the metrics endpoint with isolated registry.
func (c *client) setupMetricsEndpoint(config Config, router *chi.Mux) {
	if config.Metrics.Enabled {
		router.Handle(config.Metrics.Path, promhttp.HandlerFor(
			c.registry,
			promhttp.HandlerOpts{},
		))
	}
}

// setupAPIHandler sets up the API handler.
func (c *client) setupAPIHandler(
	apiHandler genAPI.ServerInterface,
	router *chi.Mux,
) http.Handler {
	return genAPI.HandlerWithOptions(apiHandler, genAPI.ChiServerOptions{
		BaseRouter: router,
	})
}

// createHTTPServer creates the HTTP server.
func (c *client) createHTTPServer(config Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         config.Host + ":" + strconv.Itoa(config.Port),
		Handler:      handler,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}
}

// Run runs HTTP server.
func (c *client) Run() error {
	if c.httpServer == nil {
		return ErrServerNotInitialized
	}

	c.log.Info("starting server", "addr", c.httpServer.Addr)

	if err := c.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down HTTP server.
func (c *client) Shutdown(ctx context.Context) error {
	if c.httpServer == nil {
		c.log.Info("http server is not running, skipping shutdown")

		return nil
	}

	c.log.Info("shutting down server")

	if err := c.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	return nil
}
