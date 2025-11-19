// Package database provides database client.
package database

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

// Client defines the interface for database operations.
type Client interface {
	// Connect connects to the database.
	Connect(ctx context.Context) error

	// GetPool returns the database connection pool.
	GetPool() *pgxpool.Pool

	// PingContext pings the database to verify connectivity.
	PingContext(ctx context.Context) error

	// Close closes the database connection pool.
	Close()
}

// client implements database.Client.
type client struct {
	// Config provides database configuration.
	Config Config

	// pool is the database connection pool.
	pool *pgxpool.Pool
}

// Config represents configuration for database.
type Config struct {
	// Host is host of database.
	Host string `env:"HOST" envDefault:"localhost" json:"host"`

	// Port is port of database.
	Port int `env:"PORT" envDefault:"5432" json:"port"`

	// User is user of database.
	User string `env:"USER" envDefault:"boilerplate_user" json:"user"`

	// Password is password of database.
	Password string `env:"PASSWORD" envDefault:"boilerplate_password" json:"password"`

	// DBName is name of database.
	DBName string `env:"DB_NAME" envDefault:"boilerplate" json:"db_name"`

	// SSLMode is SSL mode of database.
	SSLMode bool `env:"SSL_MODE" envDefault:"false" json:"ssl_mode"`

	// MaxConns is maximum number of connections to database.
	MaxConns int32 `env:"MAX_CONNS" envDefault:"10" json:"max_conns"`

	// MaxIdle is maximum number of idle connections to database.
	MaxIdle int32 `env:"MAX_IDLE" envDefault:"5" json:"max_idle"`
}

// NewModule provides module for database.
func NewModule() fx.Option {
	return fx.Module("database",
		// provide concrete type for constructor
		fx.Provide(func(config Config) (*client, error) {
			// create instance
			instance := newInstance(config)

			return instance, nil
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
					// connect to database
					if err := client.Connect(ctx); err != nil {
						return fmt.Errorf("failed to setup database module: %w", err)
					}

					// ping to verify connectivity
					return client.PingContext(ctx)
				},
				OnStop: func(_ context.Context) error {
					client.Close()

					return nil
				},
			})
		}),
	)
}

// newInstance creates new database instance.
func newInstance(config Config) *client {
	return &client{Config: config}
}

// Connect connects to the database.
func (c *client) Connect(ctx context.Context) error {
	// parse database connection pool config
	poolConfig, err := pgxpool.ParseConfig(c.buildConnStr())
	if err != nil {
		return fmt.Errorf("failed to parse pool config: %w", err)
	}

	// set pool configuration
	poolConfig.MaxConns = c.Config.MaxConns
	poolConfig.MinConns = c.Config.MaxIdle

	// create database connection pool
	connPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("failed to create database connection pool: %w", err)
	}

	c.pool = connPool

	return nil
}

// buildConnStr builds connection string.
func (c *client) buildConnStr() string {
	// build URL
	connURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.Config.User, c.Config.Password),
		Host:   c.buildHostPortStr(),
		Path:   "/" + c.Config.DBName,
	}

	// add SSL mode as query parameter
	query := connURL.Query()
	if c.Config.SSLMode {
		query.Set("sslmode", "require")
	} else {
		query.Set("sslmode", "disable")
	}

	// set raw query to URL
	connURL.RawQuery = query.Encode()

	return connURL.String()
}

// buildHostPort builds string of {host:port}.
func (c *client) buildHostPortStr() string {
	if c.Config.Port == 0 {
		return c.Config.Host
	}

	return c.Config.Host + ":" + strconv.Itoa(c.Config.Port)
}

// GetPool returns the database connection pool.
func (c *client) GetPool() *pgxpool.Pool {
	return c.pool
}

// PingContext pings the database.
func (c *client) PingContext(ctx context.Context) error {
	if err := c.pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

// Close closes the database connection pool.
func (c *client) Close() {
	if c.pool != nil {
		c.pool.Close()
	}
}
