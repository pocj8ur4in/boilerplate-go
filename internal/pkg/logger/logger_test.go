package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// testLevel is the test level of logger.
	testLevel = "debug"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	t.Run("set default values on logger config", func(t *testing.T) {
		t.Parallel()

		config := &Config{}
		config.SetDefault()

		require.NotNil(t, config.Level)
		assert.Equal(t, defaultLevel, *config.Level)
	})

	t.Run("preserve existing values on logger config", func(t *testing.T) {
		t.Parallel()

		level := testLevel

		config := &Config{
			Level: &level,
		}

		config.SetDefault()

		require.NotNil(t, config.Level)
		assert.Equal(t, testLevel, *config.Level)
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("create logger with valid config", func(t *testing.T) {
		t.Parallel()

		level := testLevel

		config := &Config{
			Level: &level,
		}

		logger, err := New(config)
		require.NoError(t, err)
		require.NotNil(t, logger)
		assert.NotNil(t, logger.Logger)
	})

	t.Run("create logger with nil config", func(t *testing.T) {
		t.Parallel()

		logger, err := New(nil)
		require.NoError(t, err)
		require.NotNil(t, logger)
	})

	t.Run("return error by using invalid log level", func(t *testing.T) {
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

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			level := testCase.level

			config := &Config{
				Level: &level,
			}

			logger, err := New(config)
			require.NoError(t, err)
			require.NotNil(t, logger)
		})
	}
}

func TestNewWithInsensitiveLevels(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		level string
	}{
		{"run with INFO level", "INFO"},
		{"run with Info level", "Info"},
		{"run with DEBUG level", "DEBUG"},
		{"run with Debug level", "Debug"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			level := testCase.level

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

	t.Run("create logger module", func(t *testing.T) {
		t.Parallel()

		module := NewModule()
		require.NotNil(t, module)
	})
}
