// Package logger provides logger.
package logger

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

// Logger represents logger.
type Logger struct {
	zerolog.Logger
}

// Config represents configuration for logger.
type Config struct {
	// Level is level of logger.
	Level *string `json:"level"`
}

const (
	// defaultLevel is default level of logger.
	defaultLevel = "info"
)

// SetDefault sets default values.
func (c *Config) SetDefault() {
	if c.Level == nil {
		level := defaultLevel
		c.Level = &level
	}
}

// NewModule provides module for logger.
func NewModule() fx.Option {
	return fx.Module("logger",
		fx.Provide(New),
	)
}

// New creates new logger instance.
func New(config *Config) (*Logger, error) {
	// set default
	if config == nil {
		config = &Config{}
	}

	config.SetDefault()

	// parse level
	level, err := zerolog.ParseLevel(*config.Level)
	if err != nil {
		return nil, fmt.Errorf("failed to parse log level: %w", err)
	}

	// set writer
	writer := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339Nano,
	}

	return &Logger{
		Logger: zerolog.New(writer).Level(level).With().Timestamp().Logger(),
	}, nil
}
