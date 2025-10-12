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
	"go.uber.org/fx"

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
}

// SetDefault sets default values.
func (c *Config) SetDefault() {
	c.setServerDefault()
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
	router := server.setupRouter()
	httpHandler := server.setupAPIHandler(router)
	server.httpServer = server.createHTTPServer(config, httpHandler)

	return server, nil
}

// setupRouter sets up the router.
func (s *Server) setupRouter() *chi.Mux {
	router := chi.NewRouter()

	return router
}

// setupAPIHandler sets up the API handler.
func (s *Server) setupAPIHandler(
	router *chi.Mux,
) http.Handler {
	return router
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
