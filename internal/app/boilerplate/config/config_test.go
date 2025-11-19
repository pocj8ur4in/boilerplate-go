package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocj8ur4in/boilerplate-go/internal/app/boilerplate/server"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/database"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/jwt"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/redis"
)

func TestNewModule(t *testing.T) {
	t.Parallel()

	t.Run("return fx.Option", func(t *testing.T) {
		t.Parallel()

		module := NewModule()

		require.NotNil(t, module)
	})
}

func TestLoad(t *testing.T) {
	t.Run("load config from file", func(t *testing.T) {
		// create temporary config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		// write config to file
		content := `{"logger":{"level":"debug"}}`
		err := os.WriteFile(configPath, []byte(content), 0600)
		require.NoError(t, err)

		// set config path
		t.Setenv("CONFIG_PATH", configPath)

		config, err := Load()

		require.NoError(t, err)
		require.NotNil(t, config)
		assert.Equal(t, "debug", config.Logger.Level)
	})

	t.Run("load empty config with env defaults", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		content := `{}`
		err := os.WriteFile(configPath, []byte(content), 0600)
		require.NoError(t, err)

		// set environment variable
		t.Setenv("CONFIG_PATH", configPath)

		config, err := Load()

		require.NoError(t, err)
		require.NotNil(t, config)
		assert.Equal(t, "info", config.Logger.Level)
		assert.Equal(t, "localhost", config.Database.Host)
	})
}

//nolint:paralleltest // Cannot run in parallel due to os.Chdir modifying global state
func TestLoadWithDefaultPath(t *testing.T) {
	t.Run("load config with default path", func(t *testing.T) {
		originalWd, err := os.Getwd()
		require.NoError(t, err)

		tmpDir := t.TempDir()
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		defer func() {
			_ = os.Chdir(originalWd)
		}()

		content := `{"logger":{"level":"warn"}}`
		err = os.WriteFile("config.json", []byte(content), 0600)
		require.NoError(t, err)

		// unset environment variable
		err = os.Unsetenv("CONFIG_PATH")
		require.NoError(t, err)

		config, err := Load()

		require.NoError(t, err)
		require.NotNil(t, config)
		assert.Equal(t, "warn", config.Logger.Level)
	})
}

func TestLoadWithError(t *testing.T) {
	t.Run("load config without file uses env defaults", func(t *testing.T) {
		t.Setenv("CONFIG_PATH", "/non/existent/path/config.json")

		config, err := Load()

		require.NoError(t, err)
		require.NotNil(t, config)
		assert.Equal(t, "info", config.Logger.Level)
	})

	t.Run("load config with invalid json uses env defaults", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		content := `{invalid json}`
		err := os.WriteFile(configPath, []byte(content), 0600)
		require.NoError(t, err)

		// set environment variable
		t.Setenv("CONFIG_PATH", configPath)

		config, err := Load()

		require.NoError(t, err)
		require.NotNil(t, config)
		assert.Equal(t, "info", config.Logger.Level)
	})
}

func TestProvideLoggerConfig(t *testing.T) {
	t.Parallel()

	t.Run("return logger config from config", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			Logger: logger.Config{
				Level: "debug",
			},
		}

		loggerConfig := ProvideLoggerConfig(config)

		assert.Equal(t, "debug", loggerConfig.Level)
	})
}

func TestProvideDatabaseConfig(t *testing.T) {
	t.Parallel()

	t.Run("return database config from config", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			Database: database.Config{
				Host: "localhost",
				Port: 5432,
			},
		}

		dbConfig := ProvideDatabaseConfig(config)

		assert.Equal(t, "localhost", dbConfig.Host)
		assert.Equal(t, 5432, dbConfig.Port)
	})
}

func TestProvideRedisConfig(t *testing.T) {
	t.Parallel()

	t.Run("return redis config from config", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			Redis: redis.Config{
				Addrs:    []string{"localhost:6379"},
				Password: "test_password",
				DB:       0,
			},
		}

		redisConfig := ProvideRedisConfig(config)

		assert.Equal(t, []string{"localhost:6379"}, redisConfig.Addrs)
		assert.Equal(t, "test_password", redisConfig.Password)
		assert.Equal(t, 0, redisConfig.DB)
	})
}

func TestProvideJWTConfig(t *testing.T) {
	t.Parallel()

	t.Run("return JWT config from config", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			JWT: jwt.Config{
				Issuer:    "test_issuer",
				Audience:  "test_audience",
				SecretKey: "test_secret_key",
			},
		}

		jwtConfig := ProvideJWTConfig(config)

		assert.Equal(t, "test_issuer", jwtConfig.Issuer)
		assert.Equal(t, "test_audience", jwtConfig.Audience)
		assert.Equal(t, "test_secret_key", jwtConfig.SecretKey)
	})
}

func TestProvideServerConfig(t *testing.T) {
	t.Parallel()

	t.Run("return server config from config", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			Server: server.Config{
				Host: "localhost",
				Port: 8080,
			},
		}

		serverConfig := ProvideServerConfig(config)

		require.NotNil(t, serverConfig.Host)
		require.NotNil(t, serverConfig.Port)
		assert.Equal(t, "localhost", serverConfig.Host)
		assert.Equal(t, 8080, serverConfig.Port)
	})
}
