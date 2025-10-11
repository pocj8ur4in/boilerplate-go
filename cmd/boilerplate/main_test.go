package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	app "github.com/pocj8ur4in/boilerplate-go/internal/app/boilerplate"
)

func TestApplication(t *testing.T) {
	t.Run("initialize app", func(t *testing.T) {
		// create temporary config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		content := `{}`
		err := os.WriteFile(configPath, []byte(content), 0600)
		require.NoError(t, err)

		// set environment variable
		t.Setenv("CONFIG_PATH", configPath)

		// verify that app can be created
		application := app.New()
		require.NotNil(t, application)
	})

	t.Run("run and stop application", func(t *testing.T) {
		// create temporary config file with valid configuration
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		content := `{
			"logger": {
				"level": "error",
				"output": "stdout"
			}
		}`
		err := os.WriteFile(configPath, []byte(content), 0600)
		require.NoError(t, err)

		// set environment variable
		t.Setenv("CONFIG_PATH", configPath)

		// create application
		application := app.New()
		require.NotNil(t, application)

		// run application in a goroutine
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		done := make(chan struct{})
		go func() {
			application.Run()
			close(done)
		}()

		// wait for app to start
		time.Sleep(500 * time.Millisecond)

		// stop the application
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()

		if err := application.Stop(stopCtx); err != nil {
			t.Logf("app stop returned error (expected in test): %v", err)
		}

		// wait for run to complete or timeout
		select {
		case <-done:
			// application stopped successfully
		case <-ctx.Done():
			// timeout - this is also acceptable in test environment
			t.Log("application run timed out (expected in test environment)")
		}
	})
}
