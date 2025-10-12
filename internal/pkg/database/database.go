// Package database provides database.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/fx"

	"github.com/pocj8ur4in/boilerplate-go/internal/gen/db"
)

var (
	// ErrMaxConnsExceedsLimit returned when max_conns exceeds int32 limit.
	ErrMaxConnsExceedsLimit = errors.New("max_conns exceeds int32 limit")

	// ErrMaxIdleExceedsLimit returned when max_idle exceeds int32 limit.
	ErrMaxIdleExceedsLimit = errors.New("max_idle exceeds int32 limit")
)

// DB represents database.
type DB struct {
	// DB provides database connection pool.
	*sql.DB

	// Queries provides database queries.
	Queries *db.Queries
}

// Config represents configuration for database.
type Config struct {
	// Host is host of database.
	Host *string `json:"host"`

	// Port is port of database.
	Port *int `json:"port"`

	// User is user of database.
	User *string `json:"user"`

	// Password is password of database.
	Password *string `json:"password"`

	// DBName is name of database.
	DBName *string `json:"db_name"`

	// SSLMode is SSL mode of database.
	SSLMode *bool `json:"ssl_mode"`

	// MaxConns is maximum number of connections to database.
	MaxConns *int `json:"max_conns"`

	// MaxIdle is maximum number of idle connections to database.
	MaxIdle *int `json:"max_idle"`
}

// SetDefault sets default values.
func (c *Config) SetDefault() {
	if c.Host == nil {
		c.Host = &[]string{"localhost"}[0]
	}

	if c.Port == nil {
		c.Port = &[]int{5432}[0]
	}

	if c.User == nil {
		c.User = &[]string{"boilerplate_user"}[0]
	}

	if c.Password == nil {
		c.Password = &[]string{"boilerplate_password"}[0]
	}

	if c.DBName == nil {
		c.DBName = &[]string{"boilerplate"}[0]
	}

	if c.SSLMode == nil {
		c.SSLMode = &[]bool{false}[0]
	}

	if c.MaxConns == nil {
		c.MaxConns = &[]int{10}[0]
	}

	if c.MaxIdle == nil {
		c.MaxIdle = &[]int{5}[0]
	}
}

// NewModule provides module for database.
func NewModule() fx.Option {
	return fx.Module("database",
		fx.Provide(New),
	)
}

// New creates new database instance.
func New(config *Config) (*DB, error) {
	ctx := context.Background()

	// set default
	if config == nil {
		config = &Config{}
	}

	config.SetDefault()

	// build database connection string
	connString := "host=" + *config.Host + " port=" + strconv.Itoa(*config.Port) +
		" user=" + *config.User + " password=" + *config.Password + " dbname=" + *config.DBName +
		" sslmode=" + map[bool]string{true: "require", false: "disable"}[*config.SSLMode]

	// parse database connection pool config
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pool config: %w", err)
	}

	// set database connection pool config, no need to validate on security scan
	if *config.MaxConns > math.MaxInt32 {
		return nil, fmt.Errorf("%w: %d", ErrMaxConnsExceedsLimit, *config.MaxConns)
	}

	if *config.MaxIdle > math.MaxInt32 {
		return nil, fmt.Errorf("%w: %d", ErrMaxIdleExceedsLimit, *config.MaxIdle)
	}

	// #nosec G115 -- validated above
	poolConfig.MaxConns = int32(*config.MaxConns)
	// #nosec G115 -- validated above
	poolConfig.MinConns = int32(*config.MaxIdle)

	// create database connection pool
	connPool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create database connection pool: %w", err)
	}

	// open database connection pool wrapper
	sqlDB := stdlib.OpenDBFromPool(connPool)

	// ping database connection
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// create queries using database connection pool
	queries := db.New(connPool)

	return &DB{
		DB:      sqlDB,
		Queries: queries,
	}, nil
}
