package server

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
)

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

		server, err := New(nil, log)

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

		server, err := New(config, log)

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

		server, err := New(nil, log)
		require.NoError(t, err)

		router := server.setupRouter()

		require.NotNil(t, router)
	})
}

func TestSetupAPIHandler(t *testing.T) {
	t.Parallel()

	t.Run("setup API handler successfully", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		server, err := New(nil, log)
		require.NoError(t, err)

		router := server.setupRouter()
		handler := server.setupAPIHandler(router)

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

		server, err := New(config, log)
		require.NoError(t, err)

		router := server.setupRouter()
		handler := server.setupAPIHandler(router)
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

		server, err := New(config, log)
		require.NoError(t, err)

		router := server.setupRouter()
		handler := server.setupAPIHandler(router)
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

		server, err := New(nil, log)
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
