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
	"github.com/pocj8ur4in/boilerplate-go/internal/gen/api"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/jwt"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/redis"
)

var (
	// ErrServerNotInitialized is returned when the http server is not initialized.
	ErrServerNotInitialized = errors.New("http server is not initialized")
)

// Server represents server.
type Server struct {
	// config provides server configuration.
	config *Config

	// logger provides logger.
	logger *logger.Logger

	// httpServer provides HTTP server.
	httpServer *http.Server

	// registry provides Prometheus registry for metrics.
	registry *prometheus.Registry
}

// Config represents configuration for server.
type Config struct {
	// Host is host of server.
	Host *string `json:"host"`

	// Port is port of server.
	Port *int `json:"port"`

	// ReadTimeout is read timeout of server.
	ReadTimeout *int `json:"read_timeout"`

	// WriteTimeout is write timeout of server.
	WriteTimeout *int `json:"write_timeout"`

	// IdleTimeout is idle timeout of server.
	IdleTimeout *int `json:"idle_timeout"`

	// ShutdownTimeout is shutdown timeout of server.
	ShutdownTimeout *int `json:"shutdown_timeout"`

	// MaxRequestSize is maximum request size in bytes.
	MaxRequestSize *int64 `json:"max_request_size"`

	// Compression is compression configuration of server.
	Compression *CompressionConfig `json:"compression"`

	// CORS is CORS of server.
	CORS *CORSConfig `json:"cors"`

	// RateLimit is rate limit of server.
	RateLimit *middleware.RateLimitConfig `json:"rate_limit"`

	// Metrics is metrics configuration of server.
	Metrics *middleware.MetricsConfig `json:"metrics"`
}

// CompressionConfig represents configuration for compression.
type CompressionConfig struct {
	// Level is compression level (1-9).
	Level *int `json:"level"`

	// Format is compression format (gzip, deflate, br).
	Format *string `json:"format"`

	// Enabled is whether compression is enabled.
	Enabled *bool `json:"enabled"`
}

// CORSConfig represents configuration for CORS.
type CORSConfig struct {
	// AllowedOrigins is allowed origins of CORS.
	AllowedOrigins *[]string `json:"allowed_origins"`

	// AllowedMethods is allowed methods of CORS.
	AllowedMethods *[]string `json:"allowed_methods"`

	// AllowedHeaders is allowed headers of CORS.
	AllowedHeaders *[]string `json:"allowed_headers"`
}

// SetDefault sets default values.
func (c *Config) SetDefault() {
	c.setServerDefault()
	c.setCompressionDefault()
	c.setCORSDefault()
	c.setRateLimitDefault()
	c.setMetricsDefault()
}

// setServerDefault sets default values for server.
func (c *Config) setServerDefault() {
	if c.Host == nil {
		c.Host = &[]string{"localhost"}[0]
	}

	if c.Port == nil {
		c.Port = &[]int{8080}[0]
	}

	if c.ReadTimeout == nil {
		c.ReadTimeout = &[]int{10}[0]
	}

	if c.WriteTimeout == nil {
		c.WriteTimeout = &[]int{10}[0]
	}

	if c.IdleTimeout == nil {
		c.IdleTimeout = &[]int{10}[0]
	}

	if c.ShutdownTimeout == nil {
		c.ShutdownTimeout = &[]int{10}[0]
	}

	if c.MaxRequestSize == nil {
		c.MaxRequestSize = &[]int64{10485760}[0] // 10MB
	}
}

// setCompressionDefault sets default values for compression on server.
func (c *Config) setCompressionDefault() {
	if c.Compression == nil {
		c.Compression = &CompressionConfig{}
	}

	if c.Compression.Level == nil {
		c.Compression.Level = &[]int{6}[0]
	}

	if c.Compression.Format == nil {
		c.Compression.Format = &[]string{"gzip"}[0]
	}

	if c.Compression.Enabled == nil {
		c.Compression.Enabled = &[]bool{true}[0]
	}
}

// setCORSDefault sets default values for CORS on server.
func (c *Config) setCORSDefault() {
	if c.CORS == nil {
		c.CORS = &CORSConfig{}
	}

	if c.CORS.AllowedOrigins == nil {
		c.CORS.AllowedOrigins = &[]string{"*"}
	}

	if c.CORS.AllowedMethods == nil {
		c.CORS.AllowedMethods = &[]string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}

	if c.CORS.AllowedHeaders == nil {
		c.CORS.AllowedHeaders = &[]string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"}
	}
}

// setRateLimitDefault sets default values for rate limit on server.
func (c *Config) setRateLimitDefault() {
	if c.RateLimit == nil {
		c.RateLimit = &middleware.RateLimitConfig{}
	}

	c.setGlobalRateLimitDefault()
	c.setIPRateLimitDefault()
	c.setEndpointRateLimitDefault()
}

// setGlobalRateLimitDefault sets default values for global rate limit.
func (c *Config) setGlobalRateLimitDefault() {
	if c.RateLimit.Global == nil {
		c.RateLimit.Global = &middleware.RateLimitTypeConfig{}
	}

	if c.RateLimit.Global.Enabled == nil {
		c.RateLimit.Global.Enabled = &[]bool{false}[0]
	}

	if c.RateLimit.Global.Requests == nil {
		c.RateLimit.Global.Requests = &[]int{1000}[0]
	}

	if c.RateLimit.Global.Window == nil {
		c.RateLimit.Global.Window = &[]int{60}[0]
	}
}

// setIPRateLimitDefault sets default values for IP rate limit.
func (c *Config) setIPRateLimitDefault() {
	if c.RateLimit.IP == nil {
		c.RateLimit.IP = &middleware.RateLimitTypeConfig{}
	}

	if c.RateLimit.IP.Enabled == nil {
		c.RateLimit.IP.Enabled = &[]bool{true}[0]
	}

	if c.RateLimit.IP.Requests == nil {
		c.RateLimit.IP.Requests = &[]int{100}[0]
	}

	if c.RateLimit.IP.Window == nil {
		c.RateLimit.IP.Window = &[]int{60}[0]
	}
}

// setEndpointRateLimitDefault sets default values for endpoint rate limit.
func (c *Config) setEndpointRateLimitDefault() {
	if c.RateLimit.Endpoint == nil {
		c.RateLimit.Endpoint = &middleware.RateLimitTypeConfig{}
	}

	if c.RateLimit.Endpoint.Enabled == nil {
		c.RateLimit.Endpoint.Enabled = &[]bool{false}[0]
	}

	if c.RateLimit.Endpoint.Requests == nil {
		c.RateLimit.Endpoint.Requests = &[]int{50}[0]
	}

	if c.RateLimit.Endpoint.Window == nil {
		c.RateLimit.Endpoint.Window = &[]int{60}[0]
	}
}

// setMetricsDefault sets default values for metrics.
func (c *Config) setMetricsDefault() {
	if c.Metrics == nil {
		c.Metrics = &middleware.MetricsConfig{}
	}

	if c.Metrics.Enabled == nil {
		c.Metrics.Enabled = &[]bool{true}[0]
	}

	if c.Metrics.Path == nil {
		c.Metrics.Path = &[]string{"/metrics"}[0]
	}

	if c.Metrics.ExcludePaths == nil {
		c.Metrics.ExcludePaths = []string{"/health", "/status"}
	}

	c.Metrics.SetDefault()
}

// NewModule provides module for server.
func NewModule() fx.Option {
	return fx.Module("server",
		fx.Provide(New),
	)
}

// New create new server instance.
func New(
	config *Config,
	logger *logger.Logger,
	apiHandler api.ServerInterface,
	jwtService *jwt.JWT,
	redis *redis.Redis,
) (*Server, error) {
	// set default
	if config == nil {
		config = &Config{}
	}

	config.SetDefault()

	// create server
	server := &Server{
		config:   config,
		logger:   logger,
		registry: prometheus.NewRegistry(),
	}

	// setup router and handlers
	router := server.setupRouter(config, logger, redis)
	httpHandler := server.setupAPIHandler(apiHandler, router, jwtService, logger)
	server.httpServer = server.createHTTPServer(config, httpHandler)

	return server, nil
}

// setupRouter sets up the router.
func (s *Server) setupRouter(config *Config, logger *logger.Logger, redis *redis.Redis) *chi.Mux {
	router := chi.NewRouter()

	s.setupBasicMiddlewares(router, config)
	s.setupRateLimitMiddlewares(router, config, redis, logger)
	s.setupCORS(router, config)
	s.setupMetricsEndpoint(router, config)

	return router
}

// setupBasicMiddlewares sets up basic middlewares.
func (s *Server) setupBasicMiddlewares(router *chi.Mux, config *Config) {
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.RequestSize(*config.MaxRequestSize))

	if *config.Compression.Enabled {
		router.Use(middleware.Compress(*config.Compression.Level, *config.Compression.Format))
	}

	if *config.Metrics.Enabled {
		router.Use(middleware.Metrics(config.Metrics, s.registry))
	}

	router.Use(middleware.LogRequest(s.logger))
	router.Use(middleware.Timeout(time.Duration(*config.ReadTimeout) * time.Second))
}

// setupRateLimitMiddlewares sets up rate limit middlewares.
func (s *Server) setupRateLimitMiddlewares(router *chi.Mux, config *Config, redis *redis.Redis, logger *logger.Logger) {
	if *config.RateLimit.Global.Enabled {
		router.Use(middleware.GlobalRateLimit(
			*config.RateLimit.Global.Requests,
			time.Duration(*config.RateLimit.Global.Window)*time.Second,
			redis,
			logger,
		))
	}

	if *config.RateLimit.IP.Enabled {
		router.Use(middleware.IPRateLimit(
			*config.RateLimit.IP.Requests,
			time.Duration(*config.RateLimit.IP.Window)*time.Second,
			redis,
			logger,
		))
	}

	if *config.RateLimit.Endpoint.Enabled {
		router.Use(middleware.EndpointRateLimit(
			*config.RateLimit.Endpoint.Requests,
			time.Duration(*config.RateLimit.Endpoint.Window)*time.Second,
			redis,
			logger,
		))
	}
}

// setupCORS sets up CORS handler on router.
func (s *Server) setupCORS(router *chi.Mux, config *Config) {
	const corsMaxAge = 300 // 5 minutes

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   *config.CORS.AllowedOrigins,
		AllowedMethods:   *config.CORS.AllowedMethods,
		AllowedHeaders:   *config.CORS.AllowedHeaders,
		AllowCredentials: false,
		ExposedHeaders:   []string{"Link"},
		MaxAge:           corsMaxAge,
	}))
}

// setupMetricsEndpoint sets up the metrics endpoint with isolated registry.
func (s *Server) setupMetricsEndpoint(router *chi.Mux, config *Config) {
	if *config.Metrics.Enabled {
		router.Handle(*config.Metrics.Path, promhttp.HandlerFor(
			s.registry,
			promhttp.HandlerOpts{},
		))
	}
}

// setupAPIHandler sets up the API handler with JWT authentication.
func (s *Server) setupAPIHandler(
	apiHandler api.ServerInterface,
	router *chi.Mux,
	jwtService *jwt.JWT,
	logger *logger.Logger,
) http.Handler {
	return api.HandlerWithOptions(apiHandler, api.ChiServerOptions{
		BaseRouter: router,
		Middlewares: []api.MiddlewareFunc{
			middleware.JWTAuth(jwtService, logger),
		},
	})
}

// createHTTPServer creates the HTTP server.
func (s *Server) createHTTPServer(config *Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         *config.Host + ":" + strconv.Itoa(*config.Port),
		Handler:      handler,
		ReadTimeout:  time.Duration(*config.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(*config.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(*config.IdleTimeout) * time.Second,
	}
}

// Run runs HTTP server.
func (s *Server) Run() error {
	if s.httpServer == nil {
		return ErrServerNotInitialized
	}

	s.logger.Info().
		Str("addr", s.httpServer.Addr).
		Msg("starting server")

	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		s.logger.Info().Msg("http server is not running, skipping shutdown")

		return nil
	}

	s.logger.Info().Msg("shutting down server")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	return nil
}
