package database

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/testcontainer"
)

const (
	// TestHost is the test host of database.
	TestHost = "localhost"

	// TestPort is the test port of database.
	TestPort = 35432

	// TestUser is the test user of database.
	TestUser = "boilerplate_user"

	// TestPassword is the test password of database.
	TestPassword = "boilerplate_password"

	// TestDBName is the test database name of database.
	TestDBName = "boilerplate"

	// TestSSLMode is the test SSL mode of database.
	TestSSLMode = false

	// TestMaxConns is the test maximum number of connections of database.
	TestMaxConns = int32(100)

	// TestMaxIdle is the test maximum number of idle connections of database.
	TestMaxIdle = int32(50)
)

// TestOption modifies database config for testing.
type TestOption func(*Config)

// WithSSLMode sets custom SSL mode.
func WithSSLMode(sslMode bool) TestOption {
	return func(c *Config) {
		c.SSLMode = sslMode
	}
}

// InitForTest initializes for testing.
//
//revive:disable:unexported-return // returns unexported type for testing purposes
func InitForTest(t *testing.T, opts ...TestOption) *client {
	t.Helper()

	// set test container option
	waitOccurrence := 2

	option := testcontainer.Option{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:17.4-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     TestUser,
				"POSTGRES_PASSWORD": TestPassword,
				"POSTGRES_DB":       TestDBName,
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections").
				WithOccurrence(waitOccurrence),
		},
	}

	// set factory function
	factoryFunc := func(host string, port int) (*client, error) {
		// set config
		config := Config{
			Host:     host,
			Port:     port,
			User:     TestUser,
			Password: TestPassword,
			DBName:   TestDBName,
			SSLMode:  TestSSLMode,
			MaxConns: TestMaxConns,
			MaxIdle:  TestMaxIdle,
		}

		// apply custom options
		for _, opt := range opts {
			opt(&config)
		}

		// create instance
		instance := newInstance(config)

		// connect to database
		if err := instance.Connect(context.Background()); err != nil {
			return nil, err
		}

		return instance, nil
	}

	// set ping function
	pingFunc := func(ctx context.Context, client *client) error {
		return client.PingContext(ctx)
	}

	testContainer := testcontainer.InitTestContainer(
		t,
		option,
		factoryFunc,
		pingFunc,
	)

	// set cleanup function
	t.Cleanup(func() {
		testContainer.Close()
	})

	return testContainer
}
