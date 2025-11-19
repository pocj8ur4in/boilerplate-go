// Package logger provides logger client.
package logger

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
	"go.uber.org/fx"
)

var (
	// ErrInvalidLogLevel is returned when the log level is invalid.
	ErrInvalidLogLevel = errors.New("invalid log level")

	// ErrInvalidLogFormat is returned when the log format is invalid.
	ErrInvalidLogFormat = errors.New("invalid log format")
)

// Client defines the interface for logger operations.
type Client interface {
	// Info logs a message with info level.
	Info(msg string, args ...any)

	// Error logs a message with error level.
	Error(msg string, args ...any)

	// Debug logs a message with debug level.
	Debug(msg string, args ...any)

	// Warn logs a message with warn level.
	Warn(msg string, args ...any)

	// With creates a logger with attributes for structured logging.
	With(args ...any) Client

	// Ctx returns a logger from context.
	Ctx(ctx context.Context) *slog.Logger
}

// client implements logger.Client interface.
type client struct {
	// Config provides logger configuration.
	Config Config

	// Logger extends slog.Logger.
	slog.Logger
}

// Config represents configuration for logger.
type Config struct {
	// Level is level of log.
	Level string `env:"LEVEL" envDefault:"info" json:"level"`

	// Format is format of log.
	Format string `env:"FORMAT" envDefault:"json" json:"format"`
}

// NewModule provides module for logger.
func NewModule() fx.Option {
	return fx.Module("logger",
		// provide concrete type for constructor
		fx.Provide(func(config Config) (*client, error) {
			// create instance
			instance := newInstance(config)

			// setup instance
			if err := instance.setup(); err != nil {
				return nil, fmt.Errorf("failed to setup log module: %w", err)
			}

			return instance, nil
		}),
		// provide interface type for dependency injection
		fx.Provide(fx.Annotate(
			func(instance *client) Client {
				return instance
			},
		)),
	)
}

// newInstance creates new logger instance.
func newInstance(config Config) *client {
	return &client{Config: config}
}

// setup sets up the logger instance.
func (c *client) setup() error {
	// parse level
	var level slog.Level
	switch c.Config.Level {
	case "debug", "DEBUG", "Debug":
		level = slog.LevelDebug
	case "info", "INFO", "Info":
		level = slog.LevelInfo
	case "warn", "WARN", "Warn":
		level = slog.LevelWarn
	case "error", "ERROR", "Error":
		level = slog.LevelError
	default:
		return fmt.Errorf("failed to parse log level: %w: %q", ErrInvalidLogLevel, c.Config.Level)
	}

	// set handler based on format
	var handler slog.Handler

	switch c.Config.Format {
	case "console", "text":
		handler = tint.NewHandler(os.Stdout, &tint.Options{
			Level:      level,
			TimeFormat: time.RFC3339Nano,
		})
	case "json":
		options := &slog.HandlerOptions{
			Level: level,
			ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
				// format time as RFC3339Nano
				if attr.Key == slog.TimeKey {
					return slog.String(attr.Key, attr.Value.Time().Format(time.RFC3339Nano))
				}

				return attr
			},
		}
		handler = slog.NewJSONHandler(os.Stdout, options)
	default:
		return ErrInvalidLogFormat
	}

	c.Logger = *slog.New(handler)

	return nil
}

// Info logs a message with info level.
func (c *client) Info(msg string, args ...any) {
	c.Logger.Info(msg, args...)
}

// Error logs a message with error level.
func (c *client) Error(msg string, args ...any) {
	c.Logger.Error(msg, args...)
}

// Debug logs a message with debug level.
func (c *client) Debug(msg string, args ...any) {
	c.Logger.Debug(msg, args...)
}

// Warn logs a message with warn level.
func (c *client) Warn(msg string, args ...any) {
	c.Logger.Warn(msg, args...)
}

// With creates a logger with attributes for structured logging.
func (c *client) With(args ...any) Client {
	return &client{
		Config: c.Config,
		Logger: *c.Logger.With(args...),
	}
}

// Ctx returns a logger from context.
func (c *client) Ctx(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey()).(*slog.Logger); ok && logger != nil {
		return logger
	}

	// return default logger if not found in context
	return &c.Logger
}

// WithContext adds logger to context.
func WithContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey(), logger)
}

// loggerKey returns the key used to store logger in context.
func loggerKey() *struct{} {
	return &struct{}{}
}
