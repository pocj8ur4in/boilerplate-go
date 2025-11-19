package database

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestInit(t *testing.T) {
	t.Parallel()

	t.Run("init for test instance of database", func(t *testing.T) {
		t.Parallel()

		dbClient := InitForTest(t)
		require.NotNil(t, dbClient)
		require.Equal(t, TestUser, dbClient.Config.User)
		require.Equal(t, TestPassword, dbClient.Config.Password)
		require.Equal(t, TestDBName, dbClient.Config.DBName)
		require.Equal(t, TestSSLMode, dbClient.Config.SSLMode)
		require.Equal(t, TestMaxConns, dbClient.Config.MaxConns)
		require.Equal(t, TestMaxIdle, dbClient.Config.MaxIdle)
	})
}

func TestNewModule(t *testing.T) {
	t.Parallel()

	t.Run("create module for database", func(t *testing.T) {
		t.Parallel()

		module := NewModule()
		require.NotNil(t, module)
	})

	t.Run("create app with database module", func(t *testing.T) {
		t.Parallel()

		dbClient := InitForTest(t)
		require.NotNil(t, dbClient)

		var forPopulate Client

		app := fx.New(
			fx.Supply(dbClient.Config),
			NewModule(),
			fx.Populate(&forPopulate),
		)

		ctx := context.Background()
		err := app.Start(ctx)
		require.NoError(t, err)

		require.NotNil(t, forPopulate)
		require.NotNil(t, forPopulate.GetPool())

		err = app.Stop(ctx)
		require.NoError(t, err)
	})
}

func TestNewModuleWithError(t *testing.T) {
	t.Parallel()

	t.Run("return error by using invalid config", func(t *testing.T) {
		t.Parallel()

		config := Config{
			Host:     "invalid_host_12345",
			Port:     9999,
			User:     "invalid_user",
			Password: "invalid_password",
			DBName:   "invalid_db_name",
			SSLMode:  false,
			MaxConns: 10,
			MaxIdle:  5,
		}

		app := fx.New(
			fx.Supply(config),
			NewModule(),
			fx.NopLogger,
		)

		ctx := context.Background()

		err := app.Start(ctx)
		if err == nil {
			_ = app.Stop(ctx)
		}

		require.Error(t, err)

		errorMsg := err.Error()
		hasSetupError := strings.Contains(errorMsg, "failed to setup database module")
		hasPingError := strings.Contains(errorMsg, "failed to ping database")
		assert.True(t, hasSetupError || hasPingError, errorMsg)
	})
}

func TestNewInstance(t *testing.T) {
	t.Parallel()

	t.Run("create new instance for database", func(t *testing.T) {
		t.Parallel()

		config := Config{
			Host:     TestHost,
			Port:     TestPort,
			User:     TestUser,
			Password: TestPassword,
			DBName:   TestDBName,
			SSLMode:  TestSSLMode,
			MaxConns: TestMaxConns,
			MaxIdle:  TestMaxIdle,
		}

		instance := newInstance(config)
		require.NotNil(t, instance)
		require.Equal(t, TestUser, instance.Config.User)
		require.Equal(t, TestPassword, instance.Config.Password)
		require.Equal(t, TestDBName, instance.Config.DBName)
		require.Equal(t, TestSSLMode, instance.Config.SSLMode)
		require.Equal(t, TestMaxConns, instance.Config.MaxConns)
		require.Equal(t, TestMaxIdle, instance.Config.MaxIdle)
	})
}

func TestConnect(t *testing.T) {
	t.Parallel()

	t.Run("get connect from database", func(t *testing.T) {
		t.Parallel()

		dbClient := InitForTest(t)
		require.NotNil(t, dbClient)

		err := dbClient.Connect(context.Background())
		require.NoError(t, err)
	})

	t.Run("get connect from database with SSL mode enabled", func(t *testing.T) {
		t.Parallel()

		dbClient := InitForTest(t, WithSSLMode(true))
		require.NotNil(t, dbClient)
		require.True(t, dbClient.Config.SSLMode)

		err := dbClient.Connect(context.Background())
		require.NoError(t, err)
	})
}

func TestPingContext(t *testing.T) {
	t.Parallel()

	t.Run("ping database", func(t *testing.T) {
		t.Parallel()

		dbClient := InitForTest(t)
		require.NotNil(t, dbClient)

		err := dbClient.PingContext(context.Background())
		require.NoError(t, err)
	})

	t.Run("return error on ping database with cancelled context", func(t *testing.T) {
		t.Parallel()

		dbClient := InitForTest(t)
		require.NotNil(t, dbClient)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately

		// ping with cancelled context should return error
		err := dbClient.PingContext(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to ping database")
	})
}
