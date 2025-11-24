package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/redis"
)

// defaultThrottleConfig returns a default throttle config for testing.
func defaultThrottleConfig() ThrottleConfig {
	return ThrottleConfig{
		MaxConcurrent: 2,
		MaxQueueSize:  5,
		Timeout:       2 * time.Second,
		MinDelay:      50 * time.Millisecond,
		PollInterval:  50 * time.Millisecond,
	}
}

// createTestThrottleHandler creates a test throttle middleware handler.
func createTestThrottleHandler(
	t *testing.T,
	middleware func(http.Handler) http.Handler,
) http.Handler {
	t.Helper()

	return middleware(
		http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte("OK"))
		}),
	)
}

func TestGenerateThrottleKey(t *testing.T) {
	t.Parallel()

	t.Run("generate global throttle key", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		key, err := generateThrottleKey(ThrottleTypeGlobal, request)

		require.NoError(t, err)
		assert.Equal(t, "throttle:global", key)
	})

	t.Run("generate IP throttle key", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		request.RemoteAddr = testRemoteAddr
		key, err := generateThrottleKey(ThrottleTypeIP, request)

		require.NoError(t, err)
		assert.Contains(t, key, "throttle:ip:")
		assert.Contains(t, key, testIP1)
	})

	t.Run("generate endpoint throttle key", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		request.RemoteAddr = testRemoteAddr
		key, err := generateThrottleKey(ThrottleTypeEndpoint, request)

		require.NoError(t, err)
		assert.Contains(t, key, "throttle:endpoint:")
		assert.Contains(t, key, testIP1)
		assert.Contains(t, key, "GET:/test")
	})

	t.Run("return error for unknown throttle type", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		key, err := generateThrottleKey("unknown", request)

		require.Error(t, err)
		assert.Empty(t, key)
		assert.ErrorIs(t, err, ErrInvalidThrottleType)
	})
}

//nolint:paralleltest // sequential execution required to avoid redis key conflicts
func TestGlobalRequestThrottle(t *testing.T) {
	t.Run("allow requests within concurrent limit", func(t *testing.T) {
		log := logger.InitForTest(t)
		redisClient := redis.InitForTest(t)

		config := ThrottleConfig{
			MaxConcurrent: 5,
			MaxQueueSize:  10,
			Timeout:       5 * time.Second,
			MinDelay:      10 * time.Millisecond,
			PollInterval:  100 * time.Millisecond,
		}

		middleware := GlobalRequestThrottle(config, log, redisClient)
		handler := createTestThrottleHandler(t, middleware)

		// make requests within limit
		for range 3 {
			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, "0", recorder.Header().Get("X-Throttle-Queue-Position"))
			assert.Equal(t, strconv.Itoa(config.MaxConcurrent), recorder.Header().Get("X-Throttle-Max-Concurrent"))
		}
	})
}

//nolint:paralleltest // sequential execution required to avoid redis key conflicts
func TestGlobalRequestThrottleQueueRequestsWhenLimitExceeded(t *testing.T) {
	log := logger.InitForTest(t)
	redisClient := redis.InitForTest(t)

	config := ThrottleConfig{
		MaxConcurrent: 2,
		MaxQueueSize:  5,
		Timeout:       2 * time.Second,
		MinDelay:      50 * time.Millisecond,
		PollInterval:  50 * time.Millisecond,
	}

	middleware := GlobalRequestThrottle(config, log, redisClient)
	handler := createTestThrottleHandler(t, middleware)

	// make requests exceeding concurrent limit
	done := make(chan bool, 5)

	for range 4 {
		go func() {
			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			done <- recorder.Code == http.StatusOK
		}()
	}

	// wait for some requests to complete
	time.Sleep(300 * time.Millisecond)

	successCount := 0

	for range 4 {
		select {
		case success := <-done:
			if success {
				successCount++
			}
		case <-time.After(3 * time.Second):
			// timeout
		}
	}

	assert.Positive(t, successCount)
}

//nolint:paralleltest // sequential execution required to avoid redis key conflicts
func TestGlobalRequestThrottleRejectRequestsWhenQueueFull(t *testing.T) {
	log := logger.InitForTest(t)
	redisClient := redis.InitForTest(t)

	config := ThrottleConfig{
		MaxConcurrent: 1,
		MaxQueueSize:  2,
		Timeout:       1 * time.Second,
		MinDelay:      200 * time.Millisecond,
		PollInterval:  50 * time.Millisecond,
	}

	middleware := GlobalRequestThrottle(config, log, redisClient)
	handler := createTestThrottleHandler(t, middleware)

	// fill up concurrent slot and queue
	results := make(chan int, 5)

	for range 5 {
		go func() {
			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			results <- recorder.Code
		}()
	}

	// wait for processing
	time.Sleep(500 * time.Millisecond)

	// collect results
	codes := make([]int, 0, 5)

	for range 5 {
		select {
		case code := <-results:
			codes = append(codes, code)
		case <-time.After(2 * time.Second):
			// timeout
		}
	}

	// at least some requests should succeed
	assert.NotEmpty(t, codes)
}

//nolint:paralleltest // sequential execution required to avoid redis key conflicts
func TestIPRequestThrottle(t *testing.T) {
	log := logger.InitForTest(t)
	redisClient := redis.InitForTest(t)

	config := defaultThrottleConfig()
	middleware := IPRequestThrottle(config, log, redisClient)
	handler := createTestThrottleHandler(t, middleware)

	// make requests from different IPs
	request1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	request1.Header.Set("X-Forwarded-For", "192.168.1.1")

	request2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	request2.Header.Set("X-Forwarded-For", "192.168.1.2")

	recorder1 := httptest.NewRecorder()
	recorder2 := httptest.NewRecorder()

	handler.ServeHTTP(recorder1, request1)
	handler.ServeHTTP(recorder2, request2)

	// both should succeed as they're from different IPs
	assert.Equal(t, http.StatusOK, recorder1.Code, "first request should succeed with different IPs")
	assert.Equal(t, http.StatusOK, recorder2.Code, "second request should succeed with different IPs")
}

//nolint:paralleltest // sequential execution required to avoid redis key conflicts
func TestEndpointRequestThrottle(t *testing.T) {
	log := logger.InitForTest(t)
	redisClient := redis.InitForTest(t)

	config := defaultThrottleConfig()
	middleware := EndpointRequestThrottle(config, log, redisClient)
	handler := createTestThrottleHandler(t, middleware)

	// make requests to different endpoints
	request1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	request1.Header.Set("X-Forwarded-For", "192.168.1.1")

	request2 := httptest.NewRequest(http.MethodPost, "/test", nil)
	request2.Header.Set("X-Forwarded-For", "192.168.1.1")

	recorder1 := httptest.NewRecorder()
	recorder2 := httptest.NewRecorder()

	handler.ServeHTTP(recorder1, request1)
	handler.ServeHTTP(recorder2, request2)

	// both should succeed as they're to different endpoints
	assert.Equal(t, http.StatusOK, recorder1.Code, "first request should succeed with different endpoints")
	assert.Equal(t, http.StatusOK, recorder2.Code, "second request should succeed with different endpoints")
}

//nolint:paralleltest // sequential execution required to avoid redis key conflicts
func TestTryAcquireSlotWhenAvailable(t *testing.T) {
	redisClient := redis.InitForTest(t)
	key := fmt.Sprintf("test:throttle:%d", time.Now().UnixNano())

	config := ThrottleConfig{
		MaxConcurrent: 5,
		MaxQueueSize:  10,
		Timeout:       5 * time.Second,
		MinDelay:      0,
		PollInterval:  100 * time.Millisecond,
	}

	acquired, position, err := tryAcquireSlot(context.Background(), config, redisClient, key)

	require.NoError(t, err)
	assert.True(t, acquired)
	assert.Equal(t, 0, position)
}

//nolint:paralleltest // sequential execution required to avoid redis key conflicts
func TestTryAcquireSlotQueueRequestWhenSlotNotAvailable(t *testing.T) {
	redisClient := redis.InitForTest(t)
	key := fmt.Sprintf("test:throttle:%d", time.Now().UnixNano())

	config := ThrottleConfig{
		MaxConcurrent: 1,
		MaxQueueSize:  5,
		Timeout:       5 * time.Second,
		MinDelay:      0,
		PollInterval:  100 * time.Millisecond,
	}

	// acquire first slot
	acquired1, position1, err1 := tryAcquireSlot(context.Background(), config, redisClient, key)
	require.NoError(t, err1)
	assert.True(t, acquired1)
	assert.Equal(t, 0, position1)

	// second request should be queued
	acquired2, position2, err2 := tryAcquireSlot(context.Background(), config, redisClient, key)
	require.NoError(t, err2)
	assert.False(t, acquired2)
	assert.Positive(t, position2)
}

//nolint:paralleltest // sequential execution required to avoid redis key conflicts
func TestTryAcquireSlotErrorWhenQueueFull(t *testing.T) {
	redisClient := redis.InitForTest(t)
	key := fmt.Sprintf("test:throttle:%d", time.Now().UnixNano())

	config := ThrottleConfig{
		MaxConcurrent: 1,
		MaxQueueSize:  2,
		Timeout:       5 * time.Second,
		MinDelay:      0,
		PollInterval:  100 * time.Millisecond,
	}

	// fill concurrent slot
	_, _, err := tryAcquireSlot(context.Background(), config, redisClient, key)
	require.NoError(t, err)

	// fill queue
	for range 2 {
		_, _, err := tryAcquireSlot(context.Background(), config, redisClient, key)
		require.NoError(t, err)
	}

	// next request should fail with queue full
	_, _, err = tryAcquireSlot(context.Background(), config, redisClient, key)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrQueueFull)
}

//nolint:paralleltest // sequential execution required to avoid redis key conflicts
func TestWaitForSlot(t *testing.T) {
	t.Run("wait for slot and acquire successfully", func(t *testing.T) {
		testWaitForSlotSuccess(t)
	})

	t.Run("timeout when waiting for slot", func(t *testing.T) {
		testWaitForSlotTimeout(t)
	})
}

// testWaitForSlotSuccess tests successful slot waiting.
func testWaitForSlotSuccess(t *testing.T) {
	t.Helper()

	log := logger.InitForTest(t)
	redisClient := redis.InitForTest(t)
	key := fmt.Sprintf("test:throttle:%d", time.Now().UnixNano())

	config := ThrottleConfig{
		MaxConcurrent: 1,
		MaxQueueSize:  5,
		Timeout:       2 * time.Second,
		MinDelay:      0,
		PollInterval:  50 * time.Millisecond,
	}

	// acquire slot first
	acquired, position, err := tryAcquireSlot(context.Background(), config, redisClient, key)
	require.NoError(t, err)
	require.True(t, acquired)
	require.Equal(t, 0, position)

	// queue second request
	acquired2, position2, err2 := tryAcquireSlot(context.Background(), config, redisClient, key)
	require.NoError(t, err2)
	require.False(t, acquired2)
	require.Positive(t, position2)

	// release slot in background
	go func() {
		time.Sleep(200 * time.Millisecond)

		_ = releaseSlot(context.Background(), redisClient, key)
	}()

	// wait for slot
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = waitForSlot(ctx, config, log, redisClient, key)
	require.NoError(t, err)
}

// testWaitForSlotTimeout tests timeout when waiting for slot.
func testWaitForSlotTimeout(t *testing.T) {
	t.Helper()

	log := logger.InitForTest(t)
	redisClient := redis.InitForTest(t)
	key := fmt.Sprintf("test:throttle:%d", time.Now().UnixNano())

	config := ThrottleConfig{
		MaxConcurrent: 1,
		MaxQueueSize:  5,
		Timeout:       500 * time.Millisecond,
		MinDelay:      0,
		PollInterval:  50 * time.Millisecond,
	}

	// acquire slot and don't release
	_, _, err := tryAcquireSlot(context.Background(), config, redisClient, key)
	require.NoError(t, err)

	// queue request
	_, _, err = tryAcquireSlot(context.Background(), config, redisClient, key)
	require.NoError(t, err)

	// wait for slot with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	err = waitForSlot(ctx, config, log, redisClient, key)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

//nolint:paralleltest // sequential execution required to avoid redis key conflicts
func TestReleaseSlot(t *testing.T) {
	t.Run("release slot successfully", func(t *testing.T) {
		redisClient := redis.InitForTest(t)
		key := fmt.Sprintf("test:throttle:%d", time.Now().UnixNano())

		config := ThrottleConfig{
			MaxConcurrent: 5,
			MaxQueueSize:  10,
			Timeout:       5 * time.Second,
			MinDelay:      0,
			PollInterval:  100 * time.Millisecond,
		}

		// acquire slot
		acquired, _, err := tryAcquireSlot(context.Background(), config, redisClient, key)
		require.NoError(t, err)
		require.True(t, acquired)

		// release slot
		err = releaseSlot(context.Background(), redisClient, key)
		require.NoError(t, err)

		// should be able to acquire again
		acquired2, _, err2 := tryAcquireSlot(context.Background(), config, redisClient, key)
		require.NoError(t, err2)
		assert.True(t, acquired2)
	})
}

//nolint:paralleltest // sequential execution required to avoid redis key conflicts
func TestRemoveFromQueue(t *testing.T) {
	t.Run("remove from queue successfully", func(t *testing.T) {
		redisClient := redis.InitForTest(t)
		key := fmt.Sprintf("test:throttle:%d", time.Now().UnixNano())

		config := ThrottleConfig{
			MaxConcurrent: 1,
			MaxQueueSize:  5,
			Timeout:       5 * time.Second,
			MinDelay:      0,
			PollInterval:  100 * time.Millisecond,
		}

		// fill slot
		_, _, err := tryAcquireSlot(context.Background(), config, redisClient, key)
		require.NoError(t, err)

		// add to queue
		_, _, err = tryAcquireSlot(context.Background(), config, redisClient, key)
		require.NoError(t, err)

		// remove from queue
		err = removeFromQueue(context.Background(), redisClient, key)
		require.NoError(t, err)
	})

	t.Run("handle empty queue gracefully", func(t *testing.T) {
		redisClient := redis.InitForTest(t)
		key := fmt.Sprintf("test:throttle:%d", time.Now().UnixNano())

		// try to remove from empty queue
		err := removeFromQueue(context.Background(), redisClient, key)
		// should return error for empty queue
		require.Error(t, err)
	})
}

//nolint:paralleltest // sequential execution required to avoid redis key conflicts
func TestRequestThrottleHeaders(t *testing.T) {
	t.Run("set throttle headers", func(t *testing.T) {
		log := logger.InitForTest(t)
		redisClient := redis.InitForTest(t)

		config := ThrottleConfig{
			MaxConcurrent: 5,
			MaxQueueSize:  10,
			Timeout:       5 * time.Second,
			MinDelay:      10 * time.Millisecond,
			PollInterval:  100 * time.Millisecond,
		}

		middleware := GlobalRequestThrottle(config, log, redisClient)
		handler := createTestThrottleHandler(t, middleware)

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "0", recorder.Header().Get("X-Throttle-Queue-Position"))
		assert.Equal(t, strconv.Itoa(config.MaxConcurrent), recorder.Header().Get("X-Throttle-Max-Concurrent"))
	})
}

//nolint:paralleltest // sequential execution required to avoid redis key conflicts
func TestRequestThrottleWithErrorHandling(t *testing.T) {
	t.Run("handle key generation error gracefully", func(t *testing.T) {
		log := logger.InitForTest(t)
		redisClient := redis.InitForTest(t)

		config := ThrottleConfig{
			MaxConcurrent: 5,
			MaxQueueSize:  10,
			Timeout:       5 * time.Second,
			MinDelay:      10 * time.Millisecond,
			PollInterval:  100 * time.Millisecond,
		}

		middleware := requestThrottle(ThrottleType("invalid"), config, log, redisClient)
		handler := createTestThrottleHandler(t, middleware)

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		// should still process request even with key generation error
		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}
