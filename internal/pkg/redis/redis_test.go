package redis

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestInit(t *testing.T) {
	t.Parallel()

	t.Run("init for test instance of redis", func(t *testing.T) {
		t.Parallel()

		redisClient := InitForTest(t)
		require.NotNil(t, redisClient)
		require.NotNil(t, redisClient.UniversalClient)
		require.NotEmpty(t, redisClient.Config.Addrs)
		require.Equal(t, TestPassword, redisClient.Config.Password)
		require.Equal(t, TestDB, redisClient.Config.DB)
		require.Equal(t, TestMasterName, redisClient.Config.MasterName)
		require.Equal(t, []string{}, redisClient.Config.SentinelAddrs)
	})
}

func TestNewModule(t *testing.T) {
	t.Parallel()

	t.Run("create module for redis", func(t *testing.T) {
		t.Parallel()

		module := NewModule()
		require.NotNil(t, module)
	})

	t.Run("create app with redis module", func(t *testing.T) {
		t.Parallel()

		redisClient := InitForTest(t)
		require.NotNil(t, redisClient)

		var forPopulate Client

		app := fx.New(
			fx.Supply(redisClient.Config),
			NewModule(),
			fx.Populate(&forPopulate),
		)

		ctx := context.Background()
		err := app.Start(ctx)
		require.NoError(t, err)

		require.NotNil(t, forPopulate)

		err = app.Stop(ctx)
		require.NoError(t, err)
	})
}

func TestNewModuleWithError(t *testing.T) {
	t.Parallel()

	t.Run("return error by using invalid config", func(t *testing.T) {
		t.Parallel()

		config := Config{
			Addrs:         []string{"invalid_host:9999"},
			Password:      "invalid_password",
			DB:            0,
			MasterName:    "",
			SentinelAddrs: []string{},
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
		hasSetupError := strings.Contains(errorMsg, "failed to setup redis module")
		hasPingError := strings.Contains(errorMsg, "failed to ping redis")
		assert.True(t, hasSetupError || hasPingError, errorMsg)
	})
}

func TestNewInstance(t *testing.T) {
	t.Parallel()

	t.Run("create new instance for redis", func(t *testing.T) {
		t.Parallel()

		config := Config{
			Addrs:      []string{TestAddr},
			Password:   TestPassword,
			DB:         TestDB,
			MasterName: TestMasterName,
		}

		instance := newInstance(config)
		require.NotNil(t, instance)
		require.Equal(t, TestAddr, instance.Config.Addrs[0])
		require.Equal(t, TestPassword, instance.Config.Password)
		require.Equal(t, TestDB, instance.Config.DB)
		require.Equal(t, TestMasterName, instance.Config.MasterName)
		require.Nil(t, instance.UniversalClient)
	})
}

func TestOperations(t *testing.T) {
	t.Parallel()

	t.Run("perform set, get, delete operations", func(t *testing.T) {
		t.Parallel()

		redisClient := InitForTest(t)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// set value
		key := "test_key"
		value := "test_value"
		err := redisClient.Set(ctx, key, value, 10*time.Second).Err()
		require.NoError(t, err)

		// get value
		result, err := redisClient.Get(ctx, key).Result()
		require.NoError(t, err)
		require.Equal(t, value, result)

		// delete value
		err = redisClient.Del(ctx, key).Err()
		require.NoError(t, err)

		// verify deletion
		_, err = redisClient.Get(ctx, key).Result()
		require.Error(t, err)
	})
}

func TestExpiration(t *testing.T) {
	t.Parallel()

	t.Run("return error by using short ttl for key expiration", func(t *testing.T) {
		t.Parallel()

		redisClient := InitForTest(t)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// set value with short ttl
		key := "test_expiring_key"
		value := "test_value"
		err := redisClient.Set(ctx, key, value, 1*time.Second).Err()
		require.NoError(t, err)

		// verify value exists
		result, err := redisClient.Get(ctx, key).Result()
		require.NoError(t, err)
		require.Equal(t, value, result)

		// wait for expiration
		time.Sleep(2 * time.Second)

		// verify value expired
		_, err = redisClient.Get(ctx, key).Result()
		require.Error(t, err)
	})
}

func TestDifferentDBs(t *testing.T) {
	t.Parallel()

	t.Run("use different database index", func(t *testing.T) {
		t.Parallel()

		redisClient0 := InitForTest(t, WithRedisDB(0))
		redisClient1 := InitForTest(t, WithRedisDB(1))

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// set and verify values in different DBs
		key := "db_test_key"
		require.NoError(t, redisClient0.Set(ctx, key, "value_in_db0", 10*time.Second).Err())
		require.NoError(t, redisClient1.Set(ctx, key, "value_in_db1", 10*time.Second).Err())

		result0, err := redisClient0.Get(ctx, key).Result()
		require.NoError(t, err)
		require.Equal(t, "value_in_db0", result0)

		result1, err := redisClient1.Get(ctx, key).Result()
		require.NoError(t, err)
		require.Equal(t, "value_in_db1", result1)
	})
}
