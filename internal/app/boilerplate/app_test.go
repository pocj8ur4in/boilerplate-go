package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
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
		setupTestConfig(t, `{}`)

		app := New()

		require.NotNil(t, app)
	})
}

//nolint:paralleltest // Cannot run in parallel due to t.Setenv usage
func TestNewWithStart(t *testing.T) {
	t.Run("start and stop application", func(t *testing.T) {
		setupTestConfig(t, `{}`)

		app := New()
		require.NotNil(t, app)

		startAndStopApp(t, app)
	})
}

//nolint:paralleltest // Cannot run in parallel due to t.Setenv usage
func TestRegisterHooks(t *testing.T) {
	t.Run("lifecycle hooks are called on start and stop", func(t *testing.T) {
		setupTestConfig(t, `{"logger":{"level":"info"}}`)

		app := New()
		require.NotNil(t, app)

		startAndStopApp(t, app)
	})
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
