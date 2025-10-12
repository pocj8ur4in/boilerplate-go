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

func TestConfigSetDefault(t *testing.T) {
	t.Parallel()

	t.Run("set default logger when config.Logger is nil", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		config.SetDefault()

		require.NotNil(t, config.Logger)
		require.NotNil(t, config.Logger.Level)
		assert.Equal(t, "info", *config.Logger.Level)
	})

	t.Run("keep existing logger when config.Logger is already set", func(t *testing.T) {
		t.Parallel()

		level := "debug"
		config := &Config{
			Logger: &logger.Config{
				Level: &level,
			},
		}

		config.SetDefault()

		require.NotNil(t, config.Logger)
		require.NotNil(t, config.Logger.Level)
		assert.Equal(t, "debug", *config.Logger.Level)
	})
}

func TestConfigSetDefaultDatabase(t *testing.T) {
	t.Parallel()

	t.Run("set default database when config.Database is nil", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		config.SetDefault()

		require.NotNil(t, config.Database)
		require.NotNil(t, config.Database.Host)
		assert.Equal(t, "localhost", *config.Database.Host)
	})

	t.Run("keep existing database when config.Database is already set", func(t *testing.T) {
		t.Parallel()

		host := "test-host"
		port := 3306
		config := &Config{
			Database: &database.Config{
				Host: &host,
				Port: &port,
			},
		}

		config.SetDefault()

		require.NotNil(t, config.Database)
		require.NotNil(t, config.Database.Host)
		require.NotNil(t, config.Database.Port)
		assert.Equal(t, "test-host", *config.Database.Host)
		assert.Equal(t, 3306, *config.Database.Port)
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("create new config", func(t *testing.T) {
		t.Parallel()

		config := New()

		require.NotNil(t, config)
		assert.Nil(t, config.Logger)
	})
}

func TestLoadFromFileWithValidJSON(t *testing.T) {
	t.Run("load config from valid json file", func(t *testing.T) {
		// create temporary config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		content := `{"logger":{"level":"debug"}}`
		err := os.WriteFile(configPath, []byte(content), 0600)
		require.NoError(t, err)

		// set environment variable
		t.Setenv("CONFIG_PATH", configPath)

		config, err := LoadFromFile()

		require.NoError(t, err)
		require.NotNil(t, config)
		require.NotNil(t, config.Logger)
		require.NotNil(t, config.Logger.Level)
		assert.Equal(t, "debug", *config.Logger.Level)
	})

	t.Run("load config with absolute path", func(t *testing.T) {
		// create temporary config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		content := `{"logger":{"level":"error"}}`
		err := os.WriteFile(configPath, []byte(content), 0600)
		require.NoError(t, err)

		// set environment variable with absolute path
		t.Setenv("CONFIG_PATH", configPath)

		config, err := LoadFromFile()

		require.NoError(t, err)
		require.NotNil(t, config)
		require.NotNil(t, config.Logger)
		require.NotNil(t, config.Logger.Level)
		assert.Equal(t, "error", *config.Logger.Level)
	})

	t.Run("load empty config and apply defaults", func(t *testing.T) {
		// create temporary config file with empty json
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		content := `{}`
		err := os.WriteFile(configPath, []byte(content), 0600)
		require.NoError(t, err)

		// set environment variable
		t.Setenv("CONFIG_PATH", configPath)

		config, err := LoadFromFile()

		require.NoError(t, err)
		require.NotNil(t, config)
		require.NotNil(t, config.Logger)
		require.NotNil(t, config.Logger.Level)
		assert.Equal(t, "info", *config.Logger.Level)
	})
}

//nolint:paralleltest // Cannot run in parallel due to os.Chdir modifying global state
func TestLoadFromFileWithDefaultPath(t *testing.T) {
	t.Run("load config with default path", func(t *testing.T) {
		// create config.json in current directory
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

		config, err := LoadFromFile()

		require.NoError(t, err)
		require.NotNil(t, config)
		require.NotNil(t, config.Logger)
		require.NotNil(t, config.Logger.Level)
		assert.Equal(t, "warn", *config.Logger.Level)
	})
}

func TestLoadFromFileWithErrors(t *testing.T) {
	t.Run("return error when file does not exist", func(t *testing.T) {
		// set non-existent file path
		t.Setenv("CONFIG_PATH", "/non/existent/path/config.json")

		config, err := LoadFromFile()

		require.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "failed to read file")
	})

	t.Run("return error when json is invalid", func(t *testing.T) {
		// create temporary config file with invalid json
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		content := `{invalid json}`
		err := os.WriteFile(configPath, []byte(content), 0600)
		require.NoError(t, err)

		// set environment variable
		t.Setenv("CONFIG_PATH", configPath)

		config, err := LoadFromFile()

		require.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "failed to unmarshal json")
	})
}

func TestProvideLoggerConfig(t *testing.T) {
	t.Parallel()

	t.Run("return logger config from config", func(t *testing.T) {
		t.Parallel()

		level := "debug"
		config := &Config{
			Logger: &logger.Config{
				Level: &level,
			},
		}

		loggerConfig := ProvideLoggerConfig(config)

		require.NotNil(t, loggerConfig)
		require.NotNil(t, loggerConfig.Level)
		assert.Equal(t, "debug", *loggerConfig.Level)
	})

	t.Run("return nil when config.Logger is nil", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		loggerConfig := ProvideLoggerConfig(config)

		assert.Nil(t, loggerConfig)
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

func TestProvideDatabaseConfig(t *testing.T) {
	t.Parallel()

	t.Run("return database config from config", func(t *testing.T) {
		t.Parallel()

		host := "localhost"
		port := 5432

		config := &Config{
			Database: &database.Config{
				Host: &host,
				Port: &port,
			},
		}

		dbConfig := ProvideDatabaseConfig(config)

		require.NotNil(t, dbConfig)
		require.NotNil(t, dbConfig.Host)
		require.NotNil(t, dbConfig.Port)
		assert.Equal(t, "localhost", *dbConfig.Host)
		assert.Equal(t, 5432, *dbConfig.Port)
	})

	t.Run("return nil when config.Database is nil", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		dbConfig := ProvideDatabaseConfig(config)

		assert.Nil(t, dbConfig)
	})
}

func TestProvideRedisConfig(t *testing.T) {
	t.Parallel()

	t.Run("return redis config from config", func(t *testing.T) {
		t.Parallel()

		password := "test_password"
		redisDB := 0

		config := &Config{
			Redis: &redis.Config{
				Addrs:    []string{"localhost:6379"},
				Password: &password,
				DB:       &redisDB,
			},
		}

		redisConfig := ProvideRedisConfig(config)

		require.NotNil(t, redisConfig)
		require.NotNil(t, redisConfig.Addrs)
		require.NotNil(t, redisConfig.Password)
		require.NotNil(t, redisConfig.DB)
		assert.Equal(t, []string{"localhost:6379"}, redisConfig.Addrs)
		assert.Equal(t, "test_password", *redisConfig.Password)
		assert.Equal(t, 0, *redisConfig.DB)
	})

	t.Run("return nil when config.Redis is nil", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		redisConfig := ProvideRedisConfig(config)

		assert.Nil(t, redisConfig)
	})
}

func TestConfigSetDefaultRedis(t *testing.T) {
	t.Parallel()

	t.Run("set default redis when config.Redis is nil", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		config.SetDefault()

		require.NotNil(t, config.Redis)
		require.NotNil(t, config.Redis.Addrs)
		assert.Equal(t, []string{"localhost:6379"}, config.Redis.Addrs)
	})

	t.Run("keep existing redis when config.Redis is already set", func(t *testing.T) {
		t.Parallel()

		password := "test_password"
		db := 1
		config := &Config{
			Redis: &redis.Config{
				Addrs:    []string{"redis1:6379"},
				Password: &password,
				DB:       &db,
			},
		}

		config.SetDefault()

		require.NotNil(t, config.Redis)
		require.NotNil(t, config.Redis.Addrs)
		require.NotNil(t, config.Redis.Password)
		require.NotNil(t, config.Redis.DB)
		assert.Equal(t, []string{"redis1:6379"}, config.Redis.Addrs)
		assert.Equal(t, "test_password", *config.Redis.Password)
		assert.Equal(t, 1, *config.Redis.DB)
	})
}

func TestProvideJWTConfig(t *testing.T) {
	t.Parallel()

	t.Run("return JWT config from config", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			JWT: &jwt.Config{
				Issuer:    &[]string{"test_issuer"}[0],
				Audience:  &[]string{"test_audience"}[0],
				SecretKey: &[]string{"test_secret_key"}[0],
			},
		}

		jwtConfig := ProvideJWTConfig(config)

		require.NotNil(t, jwtConfig)
		require.NotNil(t, jwtConfig.Issuer)
		require.NotNil(t, jwtConfig.Audience)
		require.NotNil(t, jwtConfig.SecretKey)
		assert.Equal(t, "test_issuer", *jwtConfig.Issuer)
		assert.Equal(t, "test_audience", *jwtConfig.Audience)
		assert.Equal(t, "test_secret_key", *jwtConfig.SecretKey)
	})

	t.Run("return nil when config.JWT is nil", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		jwtConfig := ProvideJWTConfig(config)

		assert.Nil(t, jwtConfig)
	})
}

func TestConfigSetDefaultJWT(t *testing.T) {
	t.Parallel()

	t.Run("set default JWT when config.JWT is nil", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		config.SetDefault()

		require.NotNil(t, config.JWT)
		require.NotNil(t, config.JWT.Issuer)
		assert.Equal(t, "boilerplate", *config.JWT.Issuer)
	})

	t.Run("keep existing JWT when config.JWT is already set", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			JWT: &jwt.Config{
				Issuer:    &[]string{"test_issuer"}[0],
				Audience:  &[]string{"test_audience"}[0],
				SecretKey: &[]string{"test_secret_key"}[0],
			},
		}

		config.SetDefault()

		require.NotNil(t, config.JWT)
		require.NotNil(t, config.JWT.Issuer)
		require.NotNil(t, config.JWT.Audience)
		require.NotNil(t, config.JWT.SecretKey)
		assert.Equal(t, "test_issuer", *config.JWT.Issuer)
		assert.Equal(t, "test_audience", *config.JWT.Audience)
		assert.Equal(t, "test_secret_key", *config.JWT.SecretKey)
	})
}

func TestProvideServerConfig(t *testing.T) {
	t.Parallel()

	t.Run("return server config from config", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			Server: &server.Config{
				Host: &[]string{"localhost"}[0],
				Port: &[]int{8080}[0],
			},
		}

		serverConfig := ProvideServerConfig(config)

		require.NotNil(t, serverConfig)
		require.NotNil(t, serverConfig.Host)
		require.NotNil(t, serverConfig.Port)
		assert.Equal(t, "localhost", *serverConfig.Host)
		assert.Equal(t, 8080, *serverConfig.Port)
	})

	t.Run("return nil when config.Server is nil", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		serverConfig := ProvideServerConfig(config)

		assert.Nil(t, serverConfig)
	})
}

func TestConfigSetDefaultServer(t *testing.T) {
	t.Parallel()

	t.Run("set default server when config.Server is nil", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		config.SetDefault()

		require.NotNil(t, config.Server)
		require.NotNil(t, config.Server.Host)
		assert.Equal(t, "localhost", *config.Server.Host)
	})

	t.Run("keep existing server when config.Server is already set", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			Server: &server.Config{
				Host: &[]string{"0.0.0.0"}[0],
				Port: &[]int{9090}[0],
			},
		}

		config.SetDefault()

		require.NotNil(t, config.Server)
		require.NotNil(t, config.Server.Host)
		require.NotNil(t, config.Server.Port)
		assert.Equal(t, "0.0.0.0", *config.Server.Host)
		assert.Equal(t, 9090, *config.Server.Port)
	})
}
