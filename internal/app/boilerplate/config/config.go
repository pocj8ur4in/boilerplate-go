// Package config provides the configuration for the app.
package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/caarlos0/env/v11"
	"go.uber.org/fx"

	"github.com/pocj8ur4in/boilerplate-go/internal/app/boilerplate/server"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/database"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/jwt"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/redis"
)

// Config represents the configuration for the app.
type Config struct {
	// Logger provides logger configuration.
	Logger logger.Config `envPrefix:"LOG_" json:"logger"`

	// Database provides database configuration.
	Database database.Config `envPrefix:"DB_" json:"database"`

	// JWT provides JWT configuration.
	JWT jwt.Config `envPrefix:"JWT_" json:"jwt"`

	// Redis provides redis configuration.
	Redis redis.Config `envPrefix:"REDIS_" json:"redis"`

	// Server provides server configuration.
	Server server.Config `envPrefix:"SERVER_" json:"server"`

}

// NewModule provides module for config.
func NewModule() fx.Option {
	return fx.Module("config",
		fx.Provide(
			Load,
			ProvideLoggerConfig,
			ProvideDatabaseConfig,
			ProvideJWTConfig,
			ProvideRedisConfig,
			ProvideServerConfig,
		),
	)
}

// Load loads the configuration.
func Load() (*Config, error) {
	config := &Config{}

	// load from env
	if err := loadFromEnv(config); err != nil {
		return nil, fmt.Errorf("failed to load config from env: %w", err)
	}

	// load from file
	configPath := getConfigPath()
	if configPath != "" {
		if err := loadFromFile(config, configPath); err != nil {
			log.Printf("failed to load config from file: %v", err)
		}
	}

	return config, nil
}

// loadFromEnv loads the configuration from env with setting default values.
func loadFromEnv(cfg *Config) error {
	if err := env.Parse(cfg); err != nil {
		return fmt.Errorf("failed to parse config from env: %w", err)
	}

	return nil
}

// loadFromFile loads the configuration from file.
func loadFromFile(config *Config, configPath string) error {
	// clean and validate config path
	configPath = filepath.Clean(configPath)

	// join path with working directory if not absolute
	if !filepath.IsAbs(configPath) {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		configPath = filepath.Join(wd, configPath)
	}

	// read config file from path
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// unmarshal json to config
	if err = json.Unmarshal(content, config); err != nil {
		return fmt.Errorf("failed to unmarshal json: %w", err)
	}

	return nil
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
func ProvideLoggerConfig(config *Config) logger.Config {
	return config.Logger
}

// ProvideDatabaseConfig provides database configuration.
func ProvideDatabaseConfig(config *Config) database.Config {
	return config.Database
}

// ProvideJWTConfig provides JWT configuration.
func ProvideJWTConfig(config *Config) jwt.Config {
	return config.JWT
}

// ProvideRedisConfig provides redis configuration.
func ProvideRedisConfig(config *Config) redis.Config {
	return config.Redis
}

// ProvideServerConfig provides server configuration.
func ProvideServerConfig(config *Config) server.Config {
	return config.Server
}
