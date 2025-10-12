package redis

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfig_SetDefault(t *testing.T) {
	t.Parallel()

	t.Run("set all defaults", func(t *testing.T) {
		t.Parallel()

		config := &Config{}
		config.SetDefault()

		require.Equal(t, []string{"localhost:6379"}, config.Addrs)
		require.NotNil(t, config.Password)
		require.Equal(t, "boilerplate_password", *config.Password)
		require.NotNil(t, config.DB)
		require.Equal(t, 0, *config.DB)
		require.NotNil(t, config.MasterName)
		require.Empty(t, *config.MasterName)
		require.Equal(t, []string{}, config.SentinelAddrs)
	})

	t.Run("preserve existing values", func(t *testing.T) {
		t.Parallel()

		customPassword := "test_password"
		customDB := 1
		customMasterName := "test_master"
		config := &Config{
			Addrs:         []string{"redis1:6379", "redis2:6379"},
			Password:      &customPassword,
			DB:            &customDB,
			MasterName:    &customMasterName,
			SentinelAddrs: []string{"sentinel1:26379"},
		}

		config.SetDefault()

		require.Equal(t, []string{"redis1:6379", "redis2:6379"}, config.Addrs)
		require.Equal(t, "test_password", *config.Password)
		require.Equal(t, 1, *config.DB)
		require.Equal(t, "test_master", *config.MasterName)
		require.Equal(t, []string{"sentinel1:26379"}, config.SentinelAddrs)
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("create redis with valid config", func(t *testing.T) {
		t.Parallel()

		password := ""
		db := 0
		config := &Config{
			Addrs:    []string{"localhost:36379"},
			Password: &password,
			DB:       &db,
		}

		redis, err := New(config)
		require.NoError(t, err)
		require.NotNil(t, redis)
		require.NotNil(t, redis.UniversalClient)

		// test connection
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err = redis.Ping(ctx).Err()
		require.NoError(t, err)

		// cleanup
		err = redis.Close()
		require.NoError(t, err)
	})

	t.Run("create redis with nil config", func(t *testing.T) {
		t.Parallel()

		// will fail because default localhost:6379 is not available
		_, err := New(nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to ping redis")
	})

	t.Run("create redis with invalid address", func(t *testing.T) {
		t.Parallel()

		password := ""
		db := 0
		config := &Config{
			Addrs:    []string{"invalid_test_host:9999"},
			Password: &password,
			DB:       &db,
		}

		_, err := New(config)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to ping redis")
	})
}

func TestNewWithSetGetOperations(t *testing.T) {
	t.Parallel()

	t.Run("perform set and get operations", func(t *testing.T) {
		t.Parallel()

		password := ""
		db := 0
		config := &Config{
			Addrs:    []string{"localhost:36379"},
			Password: &password,
			DB:       &db,
		}

		redis, err := New(config)
		require.NoError(t, err)
		require.NotNil(t, redis)

		defer func() {
			err := redis.Close()
			require.NoError(t, err)
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// set value
		key := "test_key"
		value := "test_value"
		err = redis.Set(ctx, key, value, 10*time.Second).Err()
		require.NoError(t, err)

		// get value
		result, err := redis.Get(ctx, key).Result()
		require.NoError(t, err)
		require.Equal(t, value, result)

		// delete value
		err = redis.Del(ctx, key).Err()
		require.NoError(t, err)

		// verify deletion
		_, err = redis.Get(ctx, key).Result()
		require.Error(t, err)
	})
}

func TestNewWithExpiration(t *testing.T) {
	t.Parallel()

	t.Run("test key expiration", func(t *testing.T) {
		t.Parallel()

		password := ""
		db := 0
		config := &Config{
			Addrs:    []string{"localhost:36379"},
			Password: &password,
			DB:       &db,
		}

		redis, err := New(config)
		require.NoError(t, err)
		require.NotNil(t, redis)

		defer func() {
			err := redis.Close()
			require.NoError(t, err)
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// set value with short ttl
		key := "test_expiring_key"
		value := "test_value"
		err = redis.Set(ctx, key, value, 1*time.Second).Err()
		require.NoError(t, err)

		// verify value exists
		result, err := redis.Get(ctx, key).Result()
		require.NoError(t, err)
		require.Equal(t, value, result)

		// wait for expiration
		time.Sleep(2 * time.Second)

		// verify value expired
		_, err = redis.Get(ctx, key).Result()
		require.Error(t, err)
	})
}

func TestNewWithDifferentDB(t *testing.T) {
	t.Parallel()

	t.Run("use different database index", func(t *testing.T) {
		t.Parallel()

		password := ""
		db1 := 0
		db2 := 1

		redis1, err := New(&Config{
			Addrs:    []string{"localhost:36379"},
			Password: &password,
			DB:       &db1,
		})
		require.NoError(t, err)
		require.NotNil(t, redis1)

		redis2, err := New(&Config{
			Addrs:    []string{"localhost:36379"},
			Password: &password,
			DB:       &db2,
		})
		require.NoError(t, err)
		require.NotNil(t, redis2)

		defer func() {
			require.NoError(t, redis1.Close())
			require.NoError(t, redis2.Close())
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// set and verify values in different DBs
		key := "db_test_key"
		require.NoError(t, redis1.Set(ctx, key, "value_in_db0", 10*time.Second).Err())
		require.NoError(t, redis2.Set(ctx, key, "value_in_db1", 10*time.Second).Err())

		result1, err := redis1.Get(ctx, key).Result()
		require.NoError(t, err)
		require.Equal(t, "value_in_db0", result1)

		result2, err := redis2.Get(ctx, key).Result()
		require.NoError(t, err)
		require.Equal(t, "value_in_db1", result2)

		// cleanup
		require.NoError(t, redis1.Del(ctx, key).Err())
		require.NoError(t, redis2.Del(ctx, key).Err())
	})
}

func TestNewModule(t *testing.T) {
	t.Parallel()

	t.Run("create redis module", func(t *testing.T) {
		t.Parallel()

		module := NewModule()
		require.NotNil(t, module)
	})
}
