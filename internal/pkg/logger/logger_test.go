package logger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestInit(t *testing.T) {
	t.Parallel()

	t.Run("init for test instance of logger", func(t *testing.T) {
		t.Parallel()

		log := InitForTest(t)
		require.NotNil(t, log)
		assert.NotNil(t, log.Logger)
		require.Equal(t, TestLoggerLevel, log.Config.Level)
	})
}

func TestInitWithError(t *testing.T) {
	t.Parallel()

	t.Run("return error by using invalid log level", func(t *testing.T) {
		t.Parallel()

		config := Config{
			Level: "invalid",
		}

		instance := newInstance(config)

		err := instance.setup()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse log level")
	})

	t.Run("return error by using invalid log format", func(t *testing.T) {
		t.Parallel()

		config := Config{
			Level:  TestLoggerLevel,
			Format: "invalid",
		}

		instance := newInstance(config)

		err := instance.setup()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidLogFormat)
	})

	t.Run("return error when format is not set", func(t *testing.T) {
		t.Parallel()

		config := Config{
			Level: TestLoggerLevel,
		}

		instance := newInstance(config)

		err := instance.setup()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidLogFormat)
	})
}

func TestNewModule(t *testing.T) {
	t.Parallel()

	t.Run("create module for logger", func(t *testing.T) {
		t.Parallel()

		module := NewModule()
		require.NotNil(t, module)
	})

	t.Run("create app with logger module", func(t *testing.T) {
		t.Parallel()

		log := InitForTest(t)
		require.NotNil(t, log)

		app := fx.New(
			fx.Supply(log.Config),
			NewModule(),
			fx.Invoke(func(c *client) {
				require.NotNil(t, c)
				require.NotNil(t, c.Logger)
			}),
		)

		ctx := context.Background()
		err := app.Start(ctx)
		require.NoError(t, err)

		err = app.Stop(ctx)
		require.NoError(t, err)
	})
}

func TestNewModuleWithError(t *testing.T) {
	t.Parallel()

	t.Run("return error by using invalid log level", func(t *testing.T) {
		t.Parallel()

		config := Config{
			Level: "invalid",
		}

		app := fx.New(
			fx.Supply(config),
			NewModule(),
			fx.Invoke(func(_ Client) {
				t.Error("should not reach here with invalid config")
			}),
			fx.NopLogger,
		)

		var capturedErr error

		if err := app.Start(context.Background()); err != nil {
			capturedErr = err
		}

		require.Error(t, capturedErr)
		assert.Contains(t, capturedErr.Error(), "failed to parse log level")
	})
}

func TestNewInstanceAndSetup(t *testing.T) {
	t.Parallel()

	t.Run("create new instance and setup it for logger", func(t *testing.T) {
		t.Parallel()

		config := Config{
			Level:  TestLoggerLevel,
			Format: TestLoggerFormat,
		}

		instance := newInstance(config)
		require.NotNil(t, instance)
		require.Equal(t, TestLoggerLevel, instance.Config.Level)

		err := instance.setup()
		require.NoError(t, err)
		require.NotNil(t, instance.Logger)
	})
}

func TestLogLevels(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		level string
	}{
		{"run with debug level", "debug"},
		{"run with info level", "info"},
		{"run with warn level", "warn"},
		{"run with error level", "error"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			log := InitForTest(t, WithLevel(testCase.level))
			require.NotNil(t, log)
		})
	}
}

func TestLogLevelsInsensitive(t *testing.T) {
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

			log := InitForTest(t, WithLevel(testCase.level))
			require.NotNil(t, log)
		})
	}
}

func TestLogInfo(t *testing.T) {
	t.Parallel()

	t.Run("log info message", func(t *testing.T) {
		t.Parallel()

		log := InitForTest(t)
		require.NotNil(t, log)

		log.Info("test info message")
		log.Info("test info message with args", "key", "value", "number", 42)
	})
}

func TestLogError(t *testing.T) {
	t.Parallel()

	t.Run("log error message", func(t *testing.T) {
		t.Parallel()

		log := InitForTest(t)
		require.NotNil(t, log)

		log.Error("test error message")
		log.Error("test error message with args", "key", "value", "number", 42)
	})
}

func TestLogDebug(t *testing.T) {
	t.Parallel()

	t.Run("log debug message", func(t *testing.T) {
		t.Parallel()

		log := InitForTest(t, WithLevel("debug"))
		require.NotNil(t, log)

		log.Debug("test debug message")
		log.Debug("test debug message with args", "key", "value", "number", 42)
	})
}

func TestLogWarn(t *testing.T) {
	t.Parallel()

	t.Run("log warn message", func(t *testing.T) {
		t.Parallel()

		log := InitForTest(t)
		require.NotNil(t, log)

		log.Warn("test warn message")
		log.Warn("test warn message with args", "key", "value", "number", 42)
	})
}

func TestLogFormats(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		format string
	}{
		{"log with json format", "json"},
		{"log with console format", "console"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			log := InitForTest(t, WithFormat(testCase.format))
			require.NotNil(t, log)

			log.Info("test info message")
			log.Info("test info message with args", "key", "value", "number", 42)
			log.Error("test error message")
			log.Debug("test debug message")
			log.Warn("test warn message")
		})
	}
}

func TestWith(t *testing.T) {
	t.Parallel()

	t.Run("create logger with attributes", func(t *testing.T) {
		t.Parallel()

		log := InitForTest(t)
		require.NotNil(t, log)

		logWithAttrs := log.With(
			"key1", "value1",
			"key2", "value2",
			"key3", 123,
			"key4", true,
		)
		require.NotNil(t, logWithAttrs)

		logWithAttrs.Info("test message with multiple attributes")
	})

	t.Run("chain With calls", func(t *testing.T) {
		t.Parallel()

		log := InitForTest(t)
		require.NotNil(t, log)

		logWithAttrs := log.With("service", "test-service").
			With("version", "1.0.0").
			With("env", "test")

		require.NotNil(t, logWithAttrs)

		logWithAttrs.Info("test message with chained attributes")
	})
}

func TestCtx(t *testing.T) {
	t.Parallel()

	t.Run("retrieve logger from context", func(t *testing.T) {
		t.Parallel()

		log := InitForTest(t)
		require.NotNil(t, log)

		clientWithCtx := log.Ctx(WithContext(context.Background(), &log.Logger))

		require.NotNil(t, clientWithCtx)
		assert.NotNil(t, clientWithCtx)
	})

	t.Run("return default logger when logger not in context", func(t *testing.T) {
		t.Parallel()

		log := InitForTest(t)
		require.NotNil(t, log)

		defaultLog := log.Ctx(context.Background())

		require.NotNil(t, defaultLog)
		assert.Equal(t, &log.Logger, defaultLog)
	})
}

func TestLevel(t *testing.T) {
	t.Parallel()

	t.Run("return log level", func(t *testing.T) {
		t.Parallel()

		log := InitForTest(t)
		require.NotNil(t, log)

		level := log.Level()

		assert.NotNil(t, level)
	})
}
