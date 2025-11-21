package redis

import (
	"context"
	"strconv"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/testcontainer"
)

const (
	// TestAddr is the test addr of redis.
	TestAddr = "localhost:36379"

	// TestPassword is the test password of redis.
	TestPassword = "boilerplate_password"

	// TestDB is the test DB of redis.
	TestDB = 0

	// TestMasterName is the test master name of redis.
	TestMasterName = "" // empty string for not using sentinel mode
)

// TestOption modifies config for testing.
type TestOption func(*Config)

// WithRedisDB sets custom redis DB.
func WithRedisDB(db int) TestOption {
	return func(c *Config) {
		c.DB = db
	}
}

// InitForTest initializes for testing.
//
//revive:disable:unexported-return // returns unexported type for testing purposes
func InitForTest(t *testing.T, opts ...TestOption) *client {
	t.Helper()

	// set test container option
	testContainerOption := testcontainer.Option{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:8-alpine",
			ExposedPorts: []string{"6379/tcp"},
			Cmd:          []string{"redis-server", "--requirepass", TestPassword},
			WaitingFor:   wait.ForLog("Ready to accept connections tcp"),
		},
	}

	// set factory function
	factoryFunc := func(host string, port int) (*client, error) {
		// set config
		config := Config{
			Addrs:         []string{host + ":" + strconv.Itoa(port)},
			Password:      TestPassword,
			DB:            TestDB,
			MasterName:    TestMasterName,
			SentinelAddrs: []string{},
		}

		// apply custom options
		for _, opt := range opts {
			opt(&config)
		}

		// create instance
		testClient := newInstance(config)

		// connect to redis
		if err := testClient.Connect(context.Background()); err != nil {
			return nil, err
		}

		return testClient, nil
	}

	// set ping function
	pingFunc := func(ctx context.Context, instance *client) error {
		return instance.UniversalClient.Ping(ctx).Err()
	}

	testContainer := testcontainer.InitTestContainer(
		t,
		testContainerOption,
		factoryFunc,
		pingFunc,
	)

	// set cleanup function
	t.Cleanup(func() {
		_ = testContainer.Close()
	})

	return testContainer
}
