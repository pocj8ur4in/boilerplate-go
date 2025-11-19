// Package testcontainer provides test containers.
package testcontainer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

var (
	// ErrAtLeastOneExposedPortRequired is returned when no exposed ports are provided.
	ErrAtLeastOneExposedPortRequired = errors.New("at least one exposed port must be provided")
)

// FactoryFunc is a function that creates a client from a host and port.
type FactoryFunc[T any] func(host string, port int) (T, error)

// PingFunc is a function that pings a client to verify connectivity.
type PingFunc[T any] func(ctx context.Context, client T) error

// Option represents optional settings for a test container.
type Option struct {
	// ContainerRequest extends the testcontainers.ContainerRequest.
	testcontainers.ContainerRequest

	// PingTimeout is the timeout for ping.
	PingTimeout *time.Duration

	// RetrySleepTime is the time to sleep between retries.
	RetrySleepTime *time.Duration

	// MaxRetries is the maximum number of retry for ping.
	MaxRetries *int

	// EndDelay is the time to wait after container is ready.
	EndDelay *time.Duration
}

const (
	// defaultPingTimeout is the default timeout for ping.
	defaultPingTimeout = 2 * time.Second

	// defaultRetrySleepTime is the default time to sleep between retries.
	defaultRetrySleepTime = 500 * time.Millisecond

	// defaultMaxRetries is the default maximum number of retries for ping.
	defaultMaxRetries = 30

	// defaultEndDelay is the default time to wait after container is ready.
	defaultEndDelay = 1 * time.Second
)

// SetDefault sets default values for the option.
func (o *Option) SetDefault() {
	if o.PingTimeout == nil {
		timeout := defaultPingTimeout
		o.PingTimeout = &timeout
	}

	if o.RetrySleepTime == nil {
		sleepTime := defaultRetrySleepTime
		o.RetrySleepTime = &sleepTime
	}

	if o.MaxRetries == nil {
		maxRetries := defaultMaxRetries
		o.MaxRetries = &maxRetries
	}

	if o.EndDelay == nil {
		endDelay := defaultEndDelay
		o.EndDelay = &endDelay
	}
}

// Validate validates the option.
func (o *Option) Validate() error {
	if len(o.ExposedPorts) == 0 {
		return ErrAtLeastOneExposedPortRequired
	}

	return nil
}

// InitTestContainer initializes a test container.
func InitTestContainer[T any](
	t *testing.T,
	option Option,
	factory FactoryFunc[T],
	ping PingFunc[T],
) T {
	t.Helper()

	option.SetDefault()

	// validate exposed ports
	if err := option.Validate(); err != nil {
		require.Fail(t, err.Error())
	}

	ctx := context.Background()

	// create test container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: option.ContainerRequest,
		Started:          true,
	})
	require.NoError(t, err)

	// wait a moment for container to be ready
	time.Sleep(*option.EndDelay)

	// get container host and mapped port
	host, err := container.Host(ctx)
	require.NoError(t, err)

	// get container mapped port
	mappedPort, err := container.MappedPort(ctx, nat.Port(option.ExposedPorts[0]))
	require.NoError(t, err)

	// create client
	client := createClient(ctx, t, &option, factory, ping, host, mappedPort.Int())

	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	return client
}

// createClient creates a client.
func createClient[T any](
	ctx context.Context,
	t *testing.T,
	option *Option,
	factory FactoryFunc[T],
	ping PingFunc[T],
	host string,
	port int,
) T {
	t.Helper()

	var (
		// client is the client to test.
		client T

		// factoryErr is the error from the factory function.
		factoryErr error
	)

	for range *option.MaxRetries {
		client, factoryErr = factory(host, port)
		if factoryErr == nil {
			// ping client to ensure connection
			pingCtx, cancel := context.WithTimeout(ctx, *option.PingTimeout)
			pingErr := ping(pingCtx, client)

			cancel()

			if pingErr == nil {
				break
			}
		}

		time.Sleep(*option.RetrySleepTime)
	}

	require.NoError(t, factoryErr)

	return client
}
