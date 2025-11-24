package logger

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	// TestLoggerLevel is the test level of logger.
	TestLoggerLevel = "debug"

	// TestLoggerFormat is the test format of logger.
	TestLoggerFormat = "json"
)

// TestOption modifies config for testing.
type TestOption func(*Config)

// WithLevel sets custom logger level.
func WithLevel(level string) TestOption {
	return func(c *Config) {
		c.Level = level
	}
}

// WithFormat sets custom logger format.
func WithFormat(format string) TestOption {
	return func(c *Config) {
		c.Format = format
	}
}

// InitForTest initializes for testing.
//
//revive:disable:unexported-return // returns unexported type for testing purposes
func InitForTest(t *testing.T, opts ...TestOption) *client {
	t.Helper()

	// set config
	config := Config{
		Level:  TestLoggerLevel,
		Format: TestLoggerFormat,
	}

	// apply custom options
	for _, opt := range opts {
		opt(&config)
	}

	// create instance
	instance := newInstance(config)

	// setup instance
	err := instance.setup()
	require.NoError(t, err)

	return instance
}
