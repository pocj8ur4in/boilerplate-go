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
	"go.uber.org/fx"

	"github.com/pocj8ur4in/boilerplate-go/internal/app/boilerplate/server/middleware"
	"github.com/pocj8ur4in/boilerplate-go/internal/gen/api"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
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
) (*Server, error) {
	// set default
	if config == nil {
		config = &Config{}
	}

	config.SetDefault()

	// create server
	server := &Server{
		config: config,
		logger: logger,
	}

	// setup router and handlers
	router := server.setupRouter(config)
	httpHandler := server.setupAPIHandler(apiHandler, router)
	server.httpServer = server.createHTTPServer(config, httpHandler)

	return server, nil
}

// setupRouter sets up the router.
func (s *Server) setupRouter(config *Config) *chi.Mux {
	router := chi.NewRouter()

	s.setupBasicMiddlewares(router, config)
	s.setupCORS(router, config)

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

	router.Use(middleware.LogRequest(s.logger))
	router.Use(middleware.Timeout(time.Duration(*config.ReadTimeout) * time.Second))
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

// setupAPIHandler sets up the API handler with JWT authentication.
func (s *Server) setupAPIHandler(
	apiHandler api.ServerInterface,
	router *chi.Mux,
) http.Handler {
	return api.HandlerWithOptions(apiHandler, api.ChiServerOptions{
		BaseRouter: router,
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
