// Package config provides the configuration for the app.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/fx"

	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/database"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/redis"
)

// NewModule provides module for config.
func NewModule() fx.Option {
	return fx.Module("config",
		fx.Provide(
			LoadFromFile,
			ProvideLoggerConfig,
			ProvideDatabaseConfig,
			ProvideRedisConfig,
		),
	)
}

// Config represents the configuration for the app.
type Config struct {
	// Logger provides logger configuration.
	Logger *logger.Config `json:"logger"`

	// Database provides database configuration.
	Database *database.Config `json:"database"`

	// Redis provides redis configuration.
	Redis *redis.Config `json:"redis"`
}

// SetDefault sets the default values.
func (c *Config) SetDefault() {
	// set logger
	if c.Logger == nil {
		c.Logger = &logger.Config{}
	}

	c.Logger.SetDefault()

	// set database
	if c.Database == nil {
		c.Database = &database.Config{}
	}

	c.Database.SetDefault()

	// set redis
	if c.Redis == nil {
		c.Redis = &redis.Config{}
	}

	c.Redis.SetDefault()
}

// New creates a new configuration.
func New() *Config {
	return &Config{}
}

// LoadFromFile loads the configuration from file.
func LoadFromFile() (*Config, error) {
	cfg := New()

	configPath := getConfigPath()

	// clean and validate config path
	configPath = filepath.Clean(configPath)

	if !filepath.IsAbs(configPath) {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}

		configPath = filepath.Join(wd, configPath)
	}

	// read file
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// unmarshal json to config
	if err = json.Unmarshal(content, cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json: %w", err)
	}

	// set default values
	cfg.SetDefault()

	return cfg, nil
}

// getConfigPath gets the config file path.
func getConfigPath() string {
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}

	// use default path
	return "config.json"
}

// ProvideLoggerConfig provides logger configuration.
func ProvideLoggerConfig(config *Config) *logger.Config {
	return config.Logger
}

// ProvideDatabaseConfig provides database configuration.
func ProvideDatabaseConfig(config *Config) *database.Config {
	return config.Database
}

// ProvideRedisConfig provides redis configuration.
func ProvideRedisConfig(config *Config) *redis.Config {
	return config.Redis
}
