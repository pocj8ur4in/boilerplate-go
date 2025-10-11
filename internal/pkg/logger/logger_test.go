package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigSetDefault(t *testing.T) {
	t.Parallel()

	t.Run("set default value when config.Level is nil", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		config.SetDefault()

		require.NotNil(t, config.Level)
		assert.Equal(t, "info", *config.Level)
	})

	t.Run("keep existing value when config.Level is already set", func(t *testing.T) {
		t.Parallel()

		level := "debug"
		config := &Config{
			Level: &level,
		}

		config.SetDefault()

		require.NotNil(t, config.Level)
		assert.Equal(t, "debug", *config.Level)
	})
}

func TestNewIsSuccess(t *testing.T) {
	t.Parallel()

	t.Run("create logger", func(t *testing.T) {
		t.Parallel()

		level := "info"
		config := &Config{
			Level: &level,
		}

		logger, err := New(config)

		require.NoError(t, err)
		require.NotNil(t, logger)
		assert.NotNil(t, logger.Logger)
	})

	t.Run("create default logger when config is nil", func(t *testing.T) {
		t.Parallel()

		logger, err := New(nil)

		require.NoError(t, err)
		require.NotNil(t, logger)
	})
}

func TestNewWithLevels(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		level string
	}{
		{"run with trace level", "trace"},
		{"run with debug level", "debug"},
		{"run with info level", "info"},
		{"run with warn level", "warn"},
		{"run with error level", "error"},
		{"run with fatal level", "fatal"},
		{"run with panic level", "panic"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config := &Config{
				Level: &tc.level,
			}

			logger, err := New(config)

			require.NoError(t, err)
			require.NotNil(t, logger)
		})
	}
}

func TestNewWithInvalidLevels(t *testing.T) {
	t.Parallel()

	t.Run("return error when invalid log level", func(t *testing.T) {
		t.Parallel()

		invalidLevel := "invalid"
		config := &Config{
			Level: &invalidLevel,
		}

		logger, err := New(config)

		require.Error(t, err)
		assert.Nil(t, logger)
		assert.Contains(t, err.Error(), "failed to parse log level")
	})
}

func TestNewWithInsensitiveLevel(t *testing.T) {
	t.Parallel()

	testCases := []string{"INFO", "Info", "DEBUG", "Debug"}

	for _, level := range testCases {
		t.Run(level, func(t *testing.T) {
			t.Parallel()

			config := &Config{
				Level: &level,
			}

			logger, err := New(config)

			require.NoError(t, err)
			require.NotNil(t, logger)
		})
	}
}

func TestNewModule(t *testing.T) {
	t.Parallel()

	t.Run("return fx.Option", func(t *testing.T) {
		t.Parallel()

		module := NewModule()

		require.NotNil(t, module)
	})
}
