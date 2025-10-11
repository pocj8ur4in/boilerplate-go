package app

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	databasePkg "github.com/pocj8ur4in/boilerplate-go/internal/pkg/database"
	loggerPkg "github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
)

// setupTestConfig creates a temporary config file and sets the environment variable.
func setupTestConfig(t *testing.T, content string) {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	err := os.WriteFile(configPath, []byte(content), 0600)
	require.NoError(t, err)

	t.Setenv("CONFIG_PATH", configPath)
}

// startAndStopApp starts and stops the application with timeout.
func startAndStopApp(t *testing.T, app *fx.App) {
	t.Helper()

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
	t.Run("create new application", func(t *testing.T) {
		setupTestConfig(t, `{
			"database": {
				"host": "localhost",
				"port": 35432,
				"user": "boilerplate_user",
				"password": "boilerplate_password",
				"db_name": "boilerplate",
				"ssl_mode": false
			},
			"logger": {
				"level": "info"
			}
		}`)

		app := New()

		require.NotNil(t, app)
	})
}

//nolint:paralleltest // Cannot run in parallel due to t.Setenv usage
func TestNewWithStart(t *testing.T) {
	t.Run("start and stop application", func(t *testing.T) {
		setupTestConfig(t, `{
			"database": {
				"host": "localhost",
				"port": 35432,
				"user": "boilerplate_user",
				"password": "boilerplate_password",
				"db_name": "boilerplate",
				"ssl_mode": false
			},
			"logger": {
				"level": "info"
			}
		}`)

		app := New()
		require.NotNil(t, app)

		startAndStopApp(t, app)
	})
}

//nolint:paralleltest // Cannot run in parallel due to t.Setenv usage
func TestRegisterHooks(t *testing.T) {
	t.Run("call lifecycle hooks with fx.App to integration test", func(t *testing.T) {
		setupTestConfig(t, `{
			"database": {
				"host": "localhost",
				"port": 35432,
				"user": "boilerplate_user",
				"password": "boilerplate_password",
				"db_name": "boilerplate",
				"ssl_mode": false
			},
			"logger": {
				"level": "info"
			}
		}`)

		app := New()
		require.NotNil(t, app)

		startAndStopApp(t, app)
	})
}

func TestRegisterHooksDirectly(t *testing.T) {
	t.Parallel()

	t.Run("call lifecycle hooks directly with mocked lifecycle to unit test", func(t *testing.T) {
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

		level := "info"
		logger, err := loggerPkg.New(&loggerPkg.Config{Level: &level})
		require.NoError(t, err)

		// create a minimal DB structure (actually won't call Close on it)
		dbConn := &databasePkg.DB{DB: &sql.DB{}}

		registerHooks(lifecycle, dbConn, logger)

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

func TestNewWithInvalidConfig(t *testing.T) {
	t.Run("fail to create app with invalid config", func(t *testing.T) {
		// set non-existent config path
		t.Setenv("CONFIG_PATH", "/non/existent/path/config.json")

		app := New()
		require.NotNil(t, app)

		// start should fail due to invalid config
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := app.Start(ctx)
		require.Error(t, err)
	})
}

//nolint:paralleltest // Cannot run in parallel due to t.Setenv usage
func TestNewWithCustomConfig(t *testing.T) {
	t.Run("create app with custom config", func(t *testing.T) {
		content := `{
			"database": {
				"host": "localhost",
				"port": 35432,
				"user": "boilerplate_user",
				"password": "boilerplate_password",
				"db_name": "boilerplate",
				"ssl_mode": false
			},
			"logger": {
				"level": "debug"
			}
		}`
		setupTestConfig(t, content)

		app := New()
		require.NotNil(t, app)

		startAndStopApp(t, app)
	})
}
