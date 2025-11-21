// Package redis provides redis client.
package redis

import (
	"context"
	"fmt"

	redisPkg "github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

// Client defines the interface for redis operations.
type Client interface {
	// Connect connects to the redis.
	Connect(ctx context.Context) error

	// UniversalClient extends redisPkg.UniversalClient.
	redisPkg.UniversalClient
}

// client implements redis.Client interface.
type client struct {
	// Config provides redis configuration.
	Config Config

	// UniversalClient extends redisPkg.UniversalClient.
	redisPkg.UniversalClient
}

// Config represents configuration for redis.
type Config struct {
	// Addrs is addresses of redis servers.
	Addrs []string `env:"ADDRESSES" envDefault:"localhost:6379" json:"addrs"`

	// Password is password of redis.
	Password string `env:"PASSWORD" envDefault:"boilerplate_password" json:"password"`

	// DB is db of redis.
	DB int `env:"DB" envDefault:"0" json:"db"`

	// MasterName is master name for sentinel mode.
	MasterName string `env:"MASTER_NAME" envDefault:"" json:"master_name"`

	// SentinelAddrs is sentinel addresses.
	SentinelAddrs []string `env:"SENTINEL_ADDRESSES" envDefault:"" json:"sentinel_addrs"`
}

// NewModule provides module for redis.
func NewModule() fx.Option {
	return fx.Module("redis",
		// provide concrete type for constructor
		fx.Provide(func(config Config) *client {
			// create instance
			instance := newInstance(config)

			return instance
		}),
		// provide interface type for dependency injection
		fx.Provide(fx.Annotate(
			func(instance *client) Client {
				return instance
			},
		)),
		// register lifecycle hooks
		fx.Invoke(func(lifecycle fx.Lifecycle, client Client) {
			lifecycle.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					// connect to redis
					if err := client.Connect(ctx); err != nil {
						return fmt.Errorf("failed to setup redis module: %w", err)
					}

					// ping to verify connectivity
					if err := client.Ping(ctx).Err(); err != nil {
						return fmt.Errorf("failed to ping redis: %w", err)
					}

					return nil
				},
				OnStop: func(_ context.Context) error {
					return client.Close()
				},
			})
		}),
	)
}

// newInstance creates new redis instance.
func newInstance(config Config) *client {
	return &client{Config: config}
}

// Connect connects to the redis.
func (c *client) Connect(_ context.Context) error {
	// create universal client options
	options := &redisPkg.UniversalOptions{
		Addrs:    c.Config.Addrs,
		Password: c.Config.Password,
		DB:       c.Config.DB,
	}

	// set master name on options
	if c.Config.MasterName != "" {
		options.MasterName = c.Config.MasterName
	}

	// add sentinel addresses to options
	if len(c.Config.SentinelAddrs) > 0 {
		options.Addrs = append(options.Addrs, c.Config.SentinelAddrs...)
	}

	// create universal client
	c.UniversalClient = redisPkg.NewUniversalClient(options)

	return nil
}
