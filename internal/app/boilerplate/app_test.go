package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/pocj8ur4in/boilerplate-go/internal/app/boilerplate/server"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/database"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/jwt"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/redis"
)

// mockLifecycle is a mock implementation of fx.Lifecycle.
type mockLifecycle struct {
	appendFunc func(fx.Hook)
}

// Append appends a hook to mockLifecycle.
func (m *mockLifecycle) Append(hook fx.Hook) {
	if m.appendFunc != nil {
		m.appendFunc(hook)
	}
}

// testConfig holds test infrastructure configuration.
type testConfig struct {
	// db provides test database client.
	db database.Client

	// dbConfig provides test database configuration.
	dbConfig database.Config

	// jwt provides test JWT client.
	jwt jwt.Client

	// jwtConfig provides test JWT configuration.
	jwtConfig jwt.Config

	// log provides test logger client.
	log logger.Client

	// logConfig provides test logger configuration.
	logConfig logger.Config

	// redis provides test redis client.
	redis redis.Client

	// redisConfig provides test redis configuration.
	redisConfig redis.Config
}

// newTestConfig creates test infrastructure configuration.
func newTestConfig(t *testing.T) *testConfig {
	t.Helper()

	// initialize client with testcontainer
	dbClient := database.InitForTest(t)
	jwtClient := jwt.InitForTest(t)
	log := logger.InitForTest(t)
	redisClient := redis.InitForTest(t)

	return &testConfig{
		db:          dbClient,
		dbConfig:    dbClient.Config,
		jwt:         jwtClient,
		jwtConfig:   jwtClient.Config,
		log:         log,
		logConfig:   log.Config,
		redis:       redisClient,
		redisConfig: redisClient.Config,
	}
}

// configContentTemplate is the template for the config file.
const configContentTemplate = `{
		"database": {
			"host": "%s",
			"port": %d,
			"user": "%s",
			"password": "%s",
			"db_name": "%s",
			"ssl_mode": false
		},
		"jwt": {
			"issuer": "%s",
			"audience": "%s",
			"secret_key": "%s"
		},
		"logger": {
			"format": "%s",
			"level": "%s"
		},
		"redis": {
			"addrs": ["%s"],
			"password": "%s",
			"db": %d
		},
		"server": {
			"host": "localhost",
			"port": 38080
		}
	}`

// beforeTest creates a temporary config file and sets the environment variable.
func beforeTest(t *testing.T, cfg *testConfig) {
	t.Helper()

	dbConfig := cfg.dbConfig
	jwtConfig := cfg.jwtConfig
	logConfig := cfg.logConfig
	redisConfig := cfg.redisConfig

	// create config content
	configContent := fmt.Sprintf(configContentTemplate,
		dbConfig.Host, dbConfig.Port,
		dbConfig.User, dbConfig.Password, dbConfig.DBName,
		jwtConfig.Issuer, jwtConfig.Audience, jwtConfig.SecretKey,
		logConfig.Format, logConfig.Level,
		redisConfig.Addrs[0],
		redisConfig.Password, redisConfig.DB,
	)

	// create temporary directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// write config to file
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	require.NoError(t, err)

	// set environment variable
	t.Setenv("CONFIG_PATH", configPath)
}

// startAndStopApp starts and stops the application with timeout.
func startAndStopApp(t *testing.T, app *fx.App) {
	t.Helper()

	// create context to timeout the application start and stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// start application
	startCtx, startCancel := context.WithTimeout(ctx, 2*time.Second)
	defer startCancel()

	err := app.Start(startCtx)
	require.NoError(t, err)

	// stop application
	stopCtx, stopCancel := context.WithTimeout(ctx, 2*time.Second)
	defer stopCancel()

	err = app.Stop(stopCtx)
	require.NoError(t, err)
}

//nolint:paralleltest // Cannot run in parallel due to t.Setenv usage
func TestNew(t *testing.T) {
	t.Run("new application", func(t *testing.T) {
		cfg := newTestConfig(t)
		beforeTest(t, cfg)

		app := New()

		require.NotNil(t, app)
	})
}

//nolint:paralleltest // Cannot run in parallel due to t.Setenv usage
func TestStartAndStop(t *testing.T) {
	t.Run("start and stop application", func(t *testing.T) {
		cfg := newTestConfig(t)
		beforeTest(t, cfg)

		app := New()
		require.NotNil(t, app)

		startAndStopApp(t, app)
	})
}

func TestRegisterHooks(t *testing.T) {
	t.Parallel()

	t.Run("register hooks with mocked lifecycle", func(t *testing.T) {
		t.Parallel()

		var hookRegistered, onStartCalled bool

		lifecycle := &mockLifecycle{
			appendFunc: func(hook fx.Hook) {
				hookRegistered = true

				if hook.OnStart != nil {
					err := hook.OnStart(context.Background())
					require.NoError(t, err)
					onStartCalled = true
				}
			},
		}

		log := logger.InitForTest(t)
		serverClient := server.InitForTest(t)

		registerHooks(lifecycle, log, serverClient)

		require.True(t, hookRegistered, "lifecycle hook should be registered")
		require.True(t, onStartCalled, "OnStart should be called successfully")
	})
}

func TestRegisterHooksOnStart(t *testing.T) {
	t.Parallel()

	t.Run("OnStart handles server Run error", func(t *testing.T) {
		t.Parallel()

		var (
			onStartCalled bool
			onStartError  error
		)

		lifecycle := &mockLifecycle{
			appendFunc: func(hook fx.Hook) {
				if hook.OnStart != nil {
					onStartCalled = true
					onStartError = hook.OnStart(context.Background())
				}
			},
		}

		log := logger.InitForTest(t)
		server := server.InitForTest(t)

		registerHooks(lifecycle, log, server)

		require.True(t, onStartCalled, "OnStart should be called")
		require.NoError(t, onStartError, "OnStart should not return error")
	})
}

func TestRegisterHooksOnStop(t *testing.T) {
	t.Parallel()

	t.Run("OnStop handles server shutdown", func(t *testing.T) {
		t.Parallel()

		var onStopCalled bool

		var onStopError error

		lifecycle := &mockLifecycle{
			appendFunc: func(hook fx.Hook) {
				if hook.OnStop != nil {
					onStopCalled = true
					onStopError = hook.OnStop(context.Background())
				}
			},
		}

		log := logger.InitForTest(t)
		serverClient := server.InitForTest(t)

		registerHooks(lifecycle, log, serverClient)

		require.True(t, onStopCalled, "OnStop should be called")
		require.NoError(t, onStopError, "OnStop should succeed")
	})
}
