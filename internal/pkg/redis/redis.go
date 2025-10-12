// Package redis provides redis.
package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

// Redis represents redis.
type Redis struct {
	// UniversalClient provides redis universal client.
	redis.UniversalClient
}

// Config represents configuration for redis.
type Config struct {
	// Addrs is addresses of redis servers.
	Addrs []string `json:"addrs"`

	// Password is password of redis.
	Password *string `json:"password"`

	// DB is db of redis.
	DB *int `json:"db"`

	// MasterName is master name for sentinel mode.
	MasterName *string `json:"master_name"`

	// SentinelAddrs is sentinel addresses.
	SentinelAddrs []string `json:"sentinel_addrs"`
}

// SetDefault sets default values.
func (c *Config) SetDefault() {
	if c.Addrs == nil {
		c.Addrs = []string{"localhost:6379"}
	}

	if c.Password == nil {
		c.Password = &[]string{"boilerplate_password"}[0]
	}

	if c.DB == nil {
		c.DB = &[]int{0}[0]
	}

	if c.MasterName == nil {
		c.MasterName = &[]string{""}[0]
	}

	if c.SentinelAddrs == nil {
		c.SentinelAddrs = []string{}
	}
}

// NewModule provides module for redis.
func NewModule() fx.Option {
	return fx.Module("redis",
		fx.Provide(New),
	)
}

// New creates new redis instance.
func New(config *Config) (*Redis, error) {
	ctx := context.Background()

	// set default
	if config == nil {
		config = &Config{}
	}

	config.SetDefault()

	// create universal client options
	options := &redis.UniversalOptions{
		Addrs:    config.Addrs,
		Password: *config.Password,
		DB:       *config.DB,
	}

	// set sentinel options if provided
	if *config.MasterName != "" {
		options.MasterName = *config.MasterName
	}

	if len(config.SentinelAddrs) > 0 {
		options.Addrs = config.SentinelAddrs
	}

	// create universal client
	redisClient := redis.NewUniversalClient(options)

	// ping redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &Redis{
		UniversalClient: redisClient,
	}, nil
}
