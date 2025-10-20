package database

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// testHost is the test host of database.
	testHost = "localhost"

	// testPort is the test port of database.
	testPort = 35432

	// testUser is the test user of database.
	testUser = "boilerplate_user"

	// testPassword is the test password of database.
	testPassword = "boilerplate_password"

	// testDBName is the test database name of database.
	testDBName = "boilerplate"

	// testSSLMode is the test SSL mode of database.
	testSSLMode = false

	// testMaxConns is the test maximum number of connections of database.
	testMaxConns = 100

	// testMaxIdle is the test maximum number of idle connections of database.
	testMaxIdle = 50
)

func TestConfig(t *testing.T) {
	t.Parallel()

	t.Run("set default values on db config", func(t *testing.T) {
		t.Parallel()

		config := &Config{}
		config.SetDefault()

		require.NotNil(t, config.Host)
		assert.Equal(t, defaultHost, *config.Host)
		require.NotNil(t, config.Port)
		assert.Equal(t, defaultPort, *config.Port)
		require.NotNil(t, config.User)
		assert.Equal(t, defaultUser, *config.User)
		require.NotNil(t, config.Password)
		assert.Equal(t, defaultPassword, *config.Password)
		require.NotNil(t, config.DBName)
		assert.Equal(t, defaultDBName, *config.DBName)
		require.NotNil(t, config.SSLMode)
		assert.False(t, *config.SSLMode)
		require.NotNil(t, config.MaxConns)
		assert.Equal(t, defaultMaxConns, *config.MaxConns)
		require.NotNil(t, config.MaxIdle)
		assert.Equal(t, defaultMaxIdle, *config.MaxIdle)
	})

	t.Run("preserve existing values on db config", func(t *testing.T) {
		t.Parallel()

		host := testHost
		port := testPort
		user := testUser
		password := testPassword
		dbName := testDBName
		sslMode := testSSLMode
		maxConns := testMaxConns
		maxIdle := testMaxIdle

		config := &Config{
			Host:     &host,
			Port:     &port,
			User:     &user,
			Password: &password,
			DBName:   &dbName,
			SSLMode:  &sslMode,
			MaxConns: &maxConns,
			MaxIdle:  &maxIdle,
		}

		config.SetDefault()

		require.Equal(t, testHost, *config.Host)
		require.Equal(t, testPort, *config.Port)
		require.Equal(t, testUser, *config.User)
		require.Equal(t, testPassword, *config.Password)
		require.Equal(t, testDBName, *config.DBName)
		require.Equal(t, testSSLMode, *config.SSLMode)
		require.Equal(t, testMaxConns, *config.MaxConns)
		require.Equal(t, testMaxIdle, *config.MaxIdle)
	})
}

func TestConfigWithSSLMode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		sslmode  bool
		expected string
	}{
		{"ssl enabled", true, "require"},
		{"ssl disabled", false, "disable"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := map[bool]string{true: "require", false: "disable"}[testCase.sslmode]
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("create db with valid config", func(t *testing.T) {
		t.Parallel()

		host := testHost
		port := testPort
		user := testUser
		password := testPassword
		dbName := testDBName
		sslMode := testSSLMode
		maxConns := testMaxConns
		maxIdle := testMaxIdle

		config := &Config{
			Host:     &host,
			Port:     &port,
			User:     &user,
			Password: &password,
			DBName:   &dbName,
			SSLMode:  &sslMode,
			MaxConns: &maxConns,
			MaxIdle:  &maxIdle,
		}

		database, err := New(config)
		require.NoError(t, err)
		require.NotNil(t, database)
		require.NotNil(t, database.DB)
		require.NotNil(t, database.Queries)

		defer func() { _ = database.Close() }()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		require.NoError(t, database.PingContext(ctx))
	})

	t.Run("return error by creating db with nil config", func(t *testing.T) {
		t.Parallel()

		_, err := New(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to ping database")
	})
}

func TestNewReturnErrors(t *testing.T) {
	t.Parallel()

	t.Run("return error by exceeding MaxConns limit", func(t *testing.T) {
		t.Parallel()

		maxConns := math.MaxInt32 + 1
		config := &Config{
			MaxConns: &maxConns,
		}

		database, err := New(config)

		require.Error(t, err)
		assert.Nil(t, database)
		assert.Contains(t, err.Error(), "max_conns exceeds int32 limit")
	})

	t.Run("return error by exceeding MaxIdle limit", func(t *testing.T) {
		t.Parallel()

		maxIdle := math.MaxInt32 + 1
		config := &Config{
			MaxIdle: &maxIdle,
		}

		database, err := New(config)

		require.Error(t, err)
		assert.Nil(t, database)
		assert.Contains(t, err.Error(), "max_idle exceeds int32 limit")
	})

	t.Run("return error by using invalid host", func(t *testing.T) {
		t.Parallel()

		invalidHost := "invalid_host_12345"
		invalidPort := 9999
		config := &Config{
			Host: &invalidHost,
			Port: &invalidPort,
		}

		database, err := New(config)

		require.Error(t, err)
		assert.Nil(t, database)
		assert.Contains(t, err.Error(), "failed to ping database")
	})
}

func TestNewModule(t *testing.T) {
	t.Parallel()

	t.Run("create db module", func(t *testing.T) {
		t.Parallel()

		module := NewModule()
		require.NotNil(t, module)
	})
}
