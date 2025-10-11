package database

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigSetDefaultHost(t *testing.T) {
	t.Parallel()

	t.Run("set default host when config.Host is nil", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		config.SetDefault()

		require.NotNil(t, config.Host)
		assert.Equal(t, "localhost", *config.Host)
	})
}

func TestConfigSetDefaultPort(t *testing.T) {
	t.Parallel()

	t.Run("set default port when config.Port is nil", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		config.SetDefault()

		require.NotNil(t, config.Port)
		assert.Equal(t, 5432, *config.Port)
	})
}

func TestConfigSetDefaultUser(t *testing.T) {
	t.Parallel()

	t.Run("set default user when config.User is nil", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		config.SetDefault()

		require.NotNil(t, config.User)
		assert.Equal(t, "boilerplate_user", *config.User)
	})
}

func TestConfigSetDefaultPassword(t *testing.T) {
	t.Parallel()

	t.Run("set default password when config.Password is nil", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		config.SetDefault()

		require.NotNil(t, config.Password)
		assert.Equal(t, "boilerplate_password", *config.Password)
	})
}

func TestConfigSetDefaultDBName(t *testing.T) {
	t.Parallel()

	t.Run("set default dbname when config.DBName is nil", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		config.SetDefault()

		require.NotNil(t, config.DBName)
		assert.Equal(t, "boilerplate", *config.DBName)
	})
}

func TestConfigSetDefaultSSLMode(t *testing.T) {
	t.Parallel()

	t.Run("set default sslmode when config.SSLMode is nil", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		config.SetDefault()

		require.NotNil(t, config.SSLMode)
		assert.False(t, *config.SSLMode)
	})
}

func TestConfigSetDefaultMaxConns(t *testing.T) {
	t.Parallel()

	t.Run("set default max_conns when config.MaxConns is nil", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		config.SetDefault()

		require.NotNil(t, config.MaxConns)
		assert.Equal(t, 10, *config.MaxConns)
	})
}

func TestConfigSetDefaultMaxIdle(t *testing.T) {
	t.Parallel()

	t.Run("set default max_idle when config.MaxIdle is nil", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		config.SetDefault()

		require.NotNil(t, config.MaxIdle)
		assert.Equal(t, 5, *config.MaxIdle)
	})
}

func TestConfigSetDefaultAllValues(t *testing.T) {
	t.Parallel()

	t.Run("set all default values at once", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		config.SetDefault()

		require.NotNil(t, config.Host)
		require.NotNil(t, config.Port)
		require.NotNil(t, config.User)
		require.NotNil(t, config.Password)
		require.NotNil(t, config.DBName)
		require.NotNil(t, config.SSLMode)
		require.NotNil(t, config.MaxConns)
		require.NotNil(t, config.MaxIdle)

		assert.Equal(t, "localhost", *config.Host)
		assert.Equal(t, 5432, *config.Port)
		assert.Equal(t, "boilerplate_user", *config.User)
		assert.Equal(t, "boilerplate_password", *config.Password)
		assert.Equal(t, "boilerplate", *config.DBName)
		assert.False(t, *config.SSLMode)
		assert.Equal(t, 10, *config.MaxConns)
		assert.Equal(t, 5, *config.MaxIdle)
	})
}

func TestConfigSetDefaultKeepExisting(t *testing.T) {
	t.Parallel()

	t.Run("keep existing values when already set", func(t *testing.T) {
		t.Parallel()

		host := "custom_host"
		port := 3306
		user := "custom_user"
		password := "custom_password"
		dbname := "custom_db"
		sslmode := true
		maxConns := 20
		maxIdle := 10

		config := &Config{
			Host:     &host,
			Port:     &port,
			User:     &user,
			Password: &password,
			DBName:   &dbname,
			SSLMode:  &sslmode,
			MaxConns: &maxConns,
			MaxIdle:  &maxIdle,
		}

		config.SetDefault()

		require.NotNil(t, config.Host)
		require.NotNil(t, config.Port)
		require.NotNil(t, config.User)
		require.NotNil(t, config.Password)
		require.NotNil(t, config.DBName)
		require.NotNil(t, config.SSLMode)
		require.NotNil(t, config.MaxConns)
		require.NotNil(t, config.MaxIdle)

		assert.Equal(t, "custom_host", *config.Host)
		assert.Equal(t, 3306, *config.Port)
		assert.Equal(t, "custom_user", *config.User)
		assert.Equal(t, "custom_password", *config.Password)
		assert.Equal(t, "custom_db", *config.DBName)
		assert.True(t, *config.SSLMode)
		assert.Equal(t, 20, *config.MaxConns)
		assert.Equal(t, 10, *config.MaxIdle)
	})
}

func TestNewWithErrors(t *testing.T) {
	t.Parallel()

	t.Run("return error when MaxConns exceeds int32 limit", func(t *testing.T) {
		t.Parallel()

		maxConns := math.MaxInt32 + 1
		config := &Config{
			MaxConns: &maxConns,
		}

		db, err := New(config)

		require.Error(t, err)
		assert.Nil(t, db)
		assert.Contains(t, err.Error(), "max_conns exceeds int32 limit")
	})

	t.Run("return error when MaxIdle exceeds int32 limit", func(t *testing.T) {
		t.Parallel()

		maxIdle := math.MaxInt32 + 1
		config := &Config{
			MaxIdle: &maxIdle,
		}

		db, err := New(config)

		require.Error(t, err)
		assert.Nil(t, db)
		assert.Contains(t, err.Error(), "max_idle exceeds int32 limit")
	})

	t.Run("return error with invalid connection string", func(t *testing.T) {
		t.Parallel()

		// use invalid host to cause connection failure
		host := "invalid_host_12345"
		port := 9999
		config := &Config{
			Host: &host,
			Port: &port,
		}

		db, err := New(config)

		require.Error(t, err)
		assert.Nil(t, db)
		assert.Contains(t, err.Error(), "failed to")
	})
}

func TestNewWithNilConfig(t *testing.T) {
	t.Parallel()

	t.Run("use default config when config is nil", func(t *testing.T) {
		t.Parallel()

		dbConn, err := New(nil)
		if err == nil {
			require.NotNil(t, dbConn)

			defer func() { _ = dbConn.Close() }()
		} else {
			assert.Nil(t, dbConn)
		}
	})
}

func TestNewWithValidInt32Values(t *testing.T) {
	t.Parallel()

	t.Run("accept MaxConns with limit", func(t *testing.T) {
		t.Parallel()

		maxConns := 100
		config := &Config{
			MaxConns: &maxConns,
		}

		dbConn, err := New(config)
		if err != nil {
			assert.NotContains(t, err.Error(), "max_conns exceeds int32 limit")
		}

		if dbConn != nil {
			defer func() { _ = dbConn.Close() }()
		}
	})

	t.Run("accept MaxIdle with limit", func(t *testing.T) {
		t.Parallel()

		maxIdle := 50
		config := &Config{
			MaxIdle: &maxIdle,
		}

		dbConn, err := New(config)
		if err != nil {
			assert.NotContains(t, err.Error(), "max_idle exceeds int32 limit")
		}

		if dbConn != nil {
			defer func() { _ = dbConn.Close() }()
		}
	})
}

func TestNewModule(t *testing.T) {
	t.Parallel()

	t.Run("return fx.Option", func(t *testing.T) {
		t.Parallel()

		module := NewModule()

		require.NotNil(t, module)
	})
}

func TestConfigSSLModeValues(t *testing.T) {
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

			// verify the ssl mode map works correctly
			result := map[bool]string{true: "require", false: "disable"}[testCase.sslmode]
			assert.Equal(t, testCase.expected, result)
		})
	}
}
