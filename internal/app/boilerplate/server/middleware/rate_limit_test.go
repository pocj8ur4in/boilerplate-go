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

const (
	testRemoteAddr = "192.168.1.1:12345"
	testIP1        = "192.168.1.1"
	testIP2        = "192.168.1.2"
)

// setupTestRedis sets up a test redis client.
func setupTestRedis(t *testing.T) *redis.Redis {
	t.Helper()

	password := ""
	db := 0
	redisConfig := &redis.Config{
		Addrs:    []string{"localhost:36379"},
		Password: &password,
		DB:       &db,
	}

	redisClient, err := redis.New(redisConfig)
	require.NoError(t, err)

	// flush DB to ensure clean state
	ctx := context.Background()
	err = redisClient.FlushDB(ctx).Err()
	require.NoError(t, err)

	return redisClient
}

// setupTestLogger sets up a test logger.
func setupTestLogger(t *testing.T) *logger.Logger {
	t.Helper()

	log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
	require.NoError(t, err)

	return log
}

// createTestRateLimitHandler creates a test rate limit middleware handler.
func createTestRateLimitHandler(
	t *testing.T,
	middleware func(http.Handler) http.Handler,
) http.Handler {
	t.Helper()

	return middleware(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
}

// testRateLimitingBehavior tests rate limiting behavior.
func testRateLimitingBehavior(
	t *testing.T,
	createMiddleware func(*redis.Redis, *logger.Logger) func(http.Handler) http.Handler,
	limit int,
	setupReq1 func(*http.Request),
	setupReq2 func(*http.Request),
	expectDifferentBehavior bool,
) {
	t.Helper()

	redisClient := setupTestRedis(t)
	log := setupTestLogger(t)

	middleware := createMiddleware(redisClient, log)
	handler := createTestRateLimitHandler(t, middleware)

	// make requests up to limit
	for range limit {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		if setupReq1 != nil {
			setupReq1(req)
		}

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)

		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(50 * time.Millisecond)

	// next request should be rate limited
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	if setupReq1 != nil {
		setupReq1(req1)
	}

	recorder1 := httptest.NewRecorder()

	handler.ServeHTTP(recorder1, req1)

	assert.Equal(t, http.StatusTooManyRequests, recorder1.Code)

	// request with different parameter should succeed if expectDifferentBehavior
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	if setupReq2 != nil {
		setupReq2(req2)
	}

	recorder2 := httptest.NewRecorder()

	handler.ServeHTTP(recorder2, req2)

	if expectDifferentBehavior {
		assert.Equal(t, http.StatusOK, recorder2.Code)
	}
}

func TestGenerateRateLimitKey(t *testing.T) {
	t.Parallel()

	t.Run("generate global rate limit key", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		key, err := generateRateLimitKey(RateLimitTypeGlobal, req)

		require.NoError(t, err)
		require.NotNil(t, key)
		assert.Equal(t, "rate_limit:global", *key)
	})

	t.Run("generate IP rate limit key", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = testRemoteAddr
		key, err := generateRateLimitKey(RateLimitTypeIP, req)

		require.NoError(t, err)
		require.NotNil(t, key)
		assert.Contains(t, *key, "rate_limit:ip:")
	})

	t.Run("generate endpoint rate limit key", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = testRemoteAddr
		key, err := generateRateLimitKey(RateLimitTypeEndpoint, req)

		require.NoError(t, err)
		require.NotNil(t, key)
		assert.Contains(t, *key, "rate_limit:endpoint:")
		assert.Contains(t, *key, "GET:/test")
	})

	t.Run("return error for unknown rate limit type", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		key, err := generateRateLimitKey(RateLimitType("unknown"), req)

		require.Error(t, err)
		assert.Nil(t, key)
		assert.ErrorIs(t, err, ErrUnknownRateLimitType)
	})
}

func TestGetClientIP(t *testing.T) {
	t.Parallel()

	t.Run("extract IP from X-Forwarded-For header", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.1")

		ip := getClientIP(req)
		assert.Equal(t, "203.0.113.1", ip)
	})

	t.Run("extract IP from X-Real-IP header", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Real-IP", "203.0.113.2")

		ip := getClientIP(req)
		assert.Equal(t, "203.0.113.2", ip)
	})

	t.Run("use RemoteAddr as fallback", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = testRemoteAddr

		ip := getClientIP(req)
		assert.Equal(t, testRemoteAddr, ip)
	})

	t.Run("X-Forwarded-For takes precedence over X-Real-IP", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.1")
		req.Header.Set("X-Real-IP", "203.0.113.2")

		ip := getClientIP(req)
		assert.Equal(t, "203.0.113.1", ip)
	})
}

//nolint:paralleltest // sequential execution required to avoid redis key conflicts
func TestGlobalRateLimit(t *testing.T) {
	t.Run("allow requests within limit", func(t *testing.T) {
		redisClient := setupTestRedis(t)
		log := setupTestLogger(t)

		middleware := GlobalRateLimit(10, 1*time.Second, redisClient, log)
		handler := createTestRateLimitHandler(t, middleware)

		// make requests
		for range 5 {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.NotEmpty(t, recorder.Header().Get("X-Ratelimit-Limit"))
			assert.NotEmpty(t, recorder.Header().Get("X-Ratelimit-Remaining"))
			assert.NotEmpty(t, recorder.Header().Get("X-Ratelimit-Reset"))
		}

		// wait for rate limit window to expire
		time.Sleep(1100 * time.Millisecond)
	})

	t.Run("reject requests exceeding limit", func(t *testing.T) {
		redisClient := setupTestRedis(t)
		log := setupTestLogger(t)

		limit := 3
		middleware := GlobalRateLimit(limit, 1*time.Second, redisClient, log)
		handler := createTestRateLimitHandler(t, middleware)

		// make requests up to limit
		for range limit {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			assert.Equal(t, http.StatusOK, recorder.Code)

			time.Sleep(50 * time.Millisecond)
		}

		time.Sleep(50 * time.Millisecond)

		// next request should be rate limited
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusTooManyRequests, recorder.Code)
		assert.Equal(t, "3", recorder.Header().Get("X-Ratelimit-Limit"))
		assert.Equal(t, "0", recorder.Header().Get("X-Ratelimit-Remaining"))
		assert.NotEmpty(t, recorder.Header().Get("Retry-After"))
	})
}

//nolint:paralleltest // sequential execution required to avoid redis key conflicts
func TestIPRateLimit(t *testing.T) {
	t.Run("rate limit per IP address", func(t *testing.T) {
		limit := 5
		testRateLimitingBehavior(
			t,
			func(redis *redis.Redis, log *logger.Logger) func(http.Handler) http.Handler {
				return IPRateLimit(limit, 1*time.Second, redis, log)
			},
			limit,
			func(req *http.Request) { req.Header.Set("X-Forwarded-For", testIP1) },
			func(req *http.Request) { req.Header.Set("X-Forwarded-For", testIP2) },
			true,
		)
	})
}

//nolint:paralleltest // sequential execution required to avoid redis key conflicts
func TestEndpointRateLimit(t *testing.T) {
	t.Run("rate limit per endpoint", func(t *testing.T) {
		redisClient := setupTestRedis(t)
		log := setupTestLogger(t)

		limit := 3
		middleware := EndpointRateLimit(limit, 1*time.Second, redisClient, log)
		handler := createTestRateLimitHandler(t, middleware)

		// make requests to /test endpoint
		for range limit {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("X-Forwarded-For", testIP1)

			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			assert.Equal(t, http.StatusOK, recorder.Code)

			time.Sleep(50 * time.Millisecond)
		}

		time.Sleep(50 * time.Millisecond)

		// next request to /test should be rate limited
		req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req1.Header.Set("X-Forwarded-For", testIP1)

		recorder1 := httptest.NewRecorder()

		handler.ServeHTTP(recorder1, req1)

		assert.Equal(t, http.StatusTooManyRequests, recorder1.Code)

		// request to different endpoint should succeed
		req2 := httptest.NewRequest(http.MethodGet, "/other", nil)
		req2.Header.Set("X-Forwarded-For", testIP1)

		recorder2 := httptest.NewRecorder()

		handler.ServeHTTP(recorder2, req2)

		assert.Equal(t, http.StatusOK, recorder2.Code)
	})
}

//nolint:paralleltest // sequential execution required to avoid redis key conflicts
func TestRateLimitHeaders(t *testing.T) {
	t.Run("set rate limit headers", func(t *testing.T) {
		redisClient := setupTestRedis(t)
		log := setupTestLogger(t)

		limit := 10
		middleware := GlobalRateLimit(limit, 1*time.Second, redisClient, log)
		handler := createTestRateLimitHandler(t, middleware)

		// make request
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		// verify headers
		assert.Equal(t, strconv.Itoa(limit), recorder.Header().Get("X-Ratelimit-Limit"))
		assert.NotEmpty(t, recorder.Header().Get("X-Ratelimit-Remaining"))
		assert.NotEmpty(t, recorder.Header().Get("X-Ratelimit-Reset"))

		remaining, err := strconv.Atoi(recorder.Header().Get("X-Ratelimit-Remaining"))
		require.NoError(t, err)
		assert.True(t, remaining >= 0 && remaining < limit)
	})
}

// callCheckRateLimit calls checkRateLimit.
func callCheckRateLimit(
	t *testing.T,
	redisClient *redis.Redis,
	key string,
	limit int,
	window time.Duration,
) (bool, int, int, time.Time, error) {
	t.Helper()

	return checkRateLimit(context.Background(), redisClient, key, limit, window)
}

//nolint:paralleltest // sequential execution required to avoid redis key conflicts
func TestCheckRateLimit(t *testing.T) {
	t.Run("check rate limit successfully", func(t *testing.T) {
		redisClient := setupTestRedis(t)
		key := fmt.Sprintf("test:rate_limit:%d", time.Now().UnixNano())
		limit := 5
		window := 60 * time.Second

		allowed, current, remaining, resetTime, err := callCheckRateLimit(
			t, redisClient, key, limit, window)

		require.NoError(t, err)
		assert.True(t, allowed)
		assert.Equal(t, 1, current)
		assert.Equal(t, 4, remaining)
		assert.True(t, resetTime.After(time.Now()))
	})

	t.Run("enforce rate limit", func(t *testing.T) {
		redisClient := setupTestRedis(t)
		key := fmt.Sprintf("test:rate_limit_enforce:%d", time.Now().UnixNano())
		limit := 2
		window := 60 * time.Second

		// make requests up to limit
		for range limit {
			allowed, _, _, _, err := callCheckRateLimit(t, redisClient, key, limit, window)

			require.NoError(t, err)
			assert.True(t, allowed)

			time.Sleep(50 * time.Millisecond)
		}

		time.Sleep(50 * time.Millisecond)

		// next request should be denied
		allowed, current, remaining, _, err := callCheckRateLimit(
			t, redisClient, key, limit, window)

		require.NoError(t, err)
		assert.False(t, allowed)
		assert.Equal(t, limit+1, current)
		assert.Equal(t, 0, remaining)
	})
}
