package app

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	configPkg "github.com/pocj8ur4in/boilerplate-go/internal/app/boilerplate/config"
	serverPkg "github.com/pocj8ur4in/boilerplate-go/internal/app/boilerplate/server"
	databasePkg "github.com/pocj8ur4in/boilerplate-go/internal/pkg/database"
	jwtPkg "github.com/pocj8ur4in/boilerplate-go/internal/pkg/jwt"
	loggerPkg "github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
	redisPkg "github.com/pocj8ur4in/boilerplate-go/internal/pkg/redis"
)

const (
	// defaultConfigContent is the default content of the config file.
	defaultConfigContent = `{
			"database": {
				"host": "localhost",
				"port": 35432,
				"user": "boilerplate_user",
				"password": "boilerplate_password",
				"db_name": "boilerplate",
				"ssl_mode": false
			},
			"jwt": {
				"issuer": "boilerplate",
				"audience": "boilerplate_audience",
				"secret_key": "test_secret_key"
			},
			"logger": {
				"level": "info"
			},
			"redis": {
				"addrs": ["localhost:36379"],
				"password": "",
				"db": 0
			},
			"server": {
				"host": "localhost",
				"port": 38080
			}
		}`
)

// beforeTest creates a temporary config file and sets the environment variable.
func beforeTest(t *testing.T, content *string) {
	t.Helper()

	// create temporary directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// use default config content if content is nil
	if content == nil {
		configContent := defaultConfigContent
		content = &configContent
	}

	// write default config to config file
	err := os.WriteFile(configPath, []byte(*content), 0600)
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
	t.Run("create application", func(t *testing.T) {
		beforeTest(t, nil)

		app := New()

		require.NotNil(t, app)
	})
}

//nolint:paralleltest // Cannot run in parallel due to t.Setenv usage
func TestStartAndStop(t *testing.T) {
	t.Run("start and stop application", func(t *testing.T) {
		beforeTest(t, nil)

		app := New()
		require.NotNil(t, app)

		startAndStopApp(t, app)
	})
}

func TestRegisterHooks(t *testing.T) {
	t.Parallel()

	t.Run("call lifecycle hooks with mocked lifecycle", func(t *testing.T) {
		t.Parallel()

		var hookRegistered, onStartCalled bool

		lifecycle := &mockLifecycle{
			appendFunc: func(hook fx.Hook) {
				hookRegistered = true
				// test OnStart
				if hook.OnStart != nil {
					err := hook.OnStart(context.Background())
					require.NoError(t, err)
					onStartCalled = true
				}
			},
		}

		log, err := loggerPkg.New(&loggerPkg.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		// create minimal structures (won't actually call Close on them)
		dbConn := &databasePkg.DB{DB: &sql.DB{}}
		redisConn := &redisPkg.Redis{}

		// create minimal server
		server := &serverPkg.Server{}

		registerHooks(lifecycle, dbConn, log, redisConn, server)

		require.True(t, hookRegistered, "lifecycle hook should be registered")
		require.True(t, onStartCalled, "OnStart should be called successfully")
	})
}

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

func TestNewReturnErrors(t *testing.T) {
	t.Run("return error by using invalid config path", func(t *testing.T) {
		// set non-existent config path
		t.Setenv("CONFIG_PATH", "/non/existent/path/config.json")

		app := fx.New(
			fx.NopLogger,
			configPkg.NewModule(),
			loggerPkg.NewModule(),
			databasePkg.NewModule(),
			jwtPkg.NewModule(),
			redisPkg.NewModule(),
			serverPkg.NewModule(),
			fx.Invoke(registerHooks),
		)
		require.NotNil(t, app)

		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read file")
	})
}

//nolint:paralleltest // Cannot run in parallel due to t.Setenv usage
func TestNewWithCustomConfig(t *testing.T) {
	t.Run("create application with custom config", func(t *testing.T) {
		configContent := `{
			"database": {
				"host": "localhost",
				"port": 35432,
				"user": "boilerplate_user",
				"password": "boilerplate_password",
				"db_name": "boilerplate",
				"ssl_mode": false
			},
			"jwt": {
				"issuer": "custom_issuer",
				"audience": "custom_audience",
				"secret_key": "custom_secret_key"
			},
			"logger": {
				"level": "debug"
			},
			"redis": {
				"addrs": ["localhost:36379"],
				"password": "",
				"db": 0
			},
			"server": {
				"host": "localhost",
				"port": 38080
			}
		}`
		beforeTest(t, &configContent)

		app := New()
		require.NotNil(t, app)

		startAndStopApp(t, app)
	})
}
