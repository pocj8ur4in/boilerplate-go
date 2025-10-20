package redis

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// testAddr is the test addr of redis.
	testAddr = "localhost:36379"

	// testPassword is the test password of redis.
	testPassword = "boilerplate_password"

	// testDB is the test DB of redis.
	testDB = 0

	// testMasterName is the test master name of redis.
	testMasterName = ""
)

func TestConfig(t *testing.T) {
	t.Parallel()

	t.Run("set default values on redis config", func(t *testing.T) {
		t.Parallel()

		config := &Config{}
		config.SetDefault()

		require.NotNil(t, config.Addrs)
		assert.Equal(t, []string{defaultAddr}, config.Addrs)
		require.NotNil(t, config.Password)
		assert.Equal(t, defaultPassword, *config.Password)
		require.NotNil(t, config.DB)
		assert.Equal(t, defaultDB, *config.DB)
		require.NotNil(t, config.MasterName)
		assert.Equal(t, defaultMasterName, *config.MasterName)
		require.NotNil(t, config.SentinelAddrs)
		assert.Equal(t, []string{}, config.SentinelAddrs)
	})

	t.Run("preserve existing values on redis config", func(t *testing.T) {
		t.Parallel()

		addrs := []string{testAddr}
		password := testPassword
		db := testDB
		masterName := testMasterName
		sentinelAddrs := []string{}

		config := &Config{
			Addrs:         addrs,
			Password:      &password,
			DB:            &db,
			MasterName:    &masterName,
			SentinelAddrs: sentinelAddrs,
		}

		config.SetDefault()

		require.Equal(t, []string{testAddr}, config.Addrs)
		require.Equal(t, testPassword, *config.Password)
		require.Equal(t, testDB, *config.DB)
		require.Equal(t, testMasterName, *config.MasterName)
		require.Equal(t, []string{}, config.SentinelAddrs)
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("create redis with valid config", func(t *testing.T) {
		t.Parallel()

		addrs := []string{testAddr}
		password := testPassword
		db := testDB
		masterName := testMasterName
		sentinelAddrs := []string{}

		config := &Config{
			Addrs:         addrs,
			Password:      &password,
			DB:            &db,
			MasterName:    &masterName,
			SentinelAddrs: sentinelAddrs,
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

	t.Run("return error by creating redis with nil config", func(t *testing.T) {
		t.Parallel()

		_, err := New(nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to ping redis")
	})
}

func TestNewReturnErrors(t *testing.T) {
	t.Parallel()

	t.Run("return error by using invalid address", func(t *testing.T) {
		t.Parallel()

		invalidAddrs := []string{"invalid_test_host:9999"}
		invalidPassword := "invalid_test_password"
		invalidDB := 9999

		config := &Config{
			Addrs:    invalidAddrs,
			Password: &invalidPassword,
			DB:       &invalidDB,
		}

		_, err := New(config)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to ping redis")
	})
}

func TestNewWithOperations(t *testing.T) {
	t.Parallel()

	t.Run("perform set, get, delete operations", func(t *testing.T) {
		t.Parallel()

		addrs := []string{testAddr}
		password := testPassword
		db := testDB
		masterName := testMasterName
		sentinelAddrs := []string{}

		config := &Config{
			Addrs:         addrs,
			Password:      &password,
			DB:            &db,
			MasterName:    &masterName,
			SentinelAddrs: sentinelAddrs,
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

	t.Run("return error by using short ttl for key expiration", func(t *testing.T) {
		t.Parallel()

		addrs := []string{testAddr}
		password := testPassword
		db := testDB
		masterName := testMasterName
		sentinelAddrs := []string{}

		config := &Config{
			Addrs:         addrs,
			Password:      &password,
			DB:            &db,
			MasterName:    &masterName,
			SentinelAddrs: sentinelAddrs,
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

func TestNewWithDifferentDBs(t *testing.T) {
	t.Parallel()

	t.Run("use different database index", func(t *testing.T) {
		t.Parallel()

		addrs := []string{testAddr}
		password := testPassword
		db1 := 0
		db2 := 1

		redis1, err := New(&Config{
			Addrs:    addrs,
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
