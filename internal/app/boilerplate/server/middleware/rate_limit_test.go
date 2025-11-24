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
	// testRemoteAddr is the test remote address.
	testRemoteAddr = "192.168.1.1:12345"

	// testIP1 is the test IP address 1.
	testIP1 = "192.168.1.1"

	// testIP2 is the test IP address 2.
	testIP2 = "192.168.1.2"
)

// createTestRateLimitHandler creates a test rate limit middleware handler.
func createTestRateLimitHandler(
	t *testing.T,
	middleware func(http.Handler) http.Handler,
) http.Handler {
	t.Helper()

	return middleware(
		http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusOK)
		}),
	)
}

// testRateLimitingBehavior tests rate limiting behavior.
func testRateLimitingBehavior(
	t *testing.T,
	createMiddleware func(logger.Client, redis.Client) func(http.Handler) http.Handler,
	limit int,
	setupRequest1 func(*http.Request),
	setupRequest2 func(*http.Request),
	expectDifferentBehavior bool,
) {
	t.Helper()

	log := logger.InitForTest(t)
	redisClient := redis.InitForTest(t)

	middleware := createMiddleware(log, redisClient)
	handler := createTestRateLimitHandler(t, middleware)

	// make requests up to limit
	for range limit {
		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		if setupRequest1 != nil {
			setupRequest1(request)
		}

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)

		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(50 * time.Millisecond)

	// next request should be rate limited
	request1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	if setupRequest1 != nil {
		setupRequest1(request1)
	}

	recorder1 := httptest.NewRecorder()

	handler.ServeHTTP(recorder1, request1)

	assert.Equal(t, http.StatusTooManyRequests, recorder1.Code)

	// request with different parameter should succeed if expectDifferentBehavior
	request2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	if setupRequest2 != nil {
		setupRequest2(request2)
	}

	recorder2 := httptest.NewRecorder()

	handler.ServeHTTP(recorder2, request2)

	if expectDifferentBehavior {
		assert.Equal(t, http.StatusOK, recorder2.Code)
	}
}

func TestGenerateRateLimitKey(t *testing.T) {
	t.Parallel()

	t.Run("generate global rate limit key", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		key, err := generateRateLimitKey(RateLimitTypeGlobal, request)

		require.NoError(t, err)
		require.NotNil(t, key)
		assert.Equal(t, "rate_limit:global", key)
	})

	t.Run("generate IP rate limit key", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		request.RemoteAddr = testRemoteAddr
		key, err := generateRateLimitKey(RateLimitTypeIP, request)

		require.NoError(t, err)
		require.NotNil(t, key)
		assert.Contains(t, key, "rate_limit:ip:")
	})

	t.Run("generate endpoint rate limit key", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		request.RemoteAddr = testRemoteAddr
		key, err := generateRateLimitKey(RateLimitTypeEndpoint, request)

		require.NoError(t, err)
		require.NotNil(t, key)
		assert.Contains(t, key, "rate_limit:endpoint:")
		assert.Contains(t, key, "GET:/test")
	})

	t.Run("return error for unknown rate limit type", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		key, err := generateRateLimitKey("unknown", request)

		require.Error(t, err)
		assert.Empty(t, key)
		assert.ErrorIs(t, err, ErrUnknownRateLimitType)
	})
}

func TestGetClientIP(t *testing.T) {
	t.Parallel()

	t.Run("extract IP from X-Forwarded-For header", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		request.Header.Set("X-Forwarded-For", "203.0.113.1")

		ip := getClientIP(request)
		assert.Equal(t, "203.0.113.1", ip)
	})

	t.Run("extract IP from X-Real-IP header", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		request.Header.Set("X-Real-IP", "203.0.113.2")

		ip := getClientIP(request)
		assert.Equal(t, "203.0.113.2", ip)
	})

	t.Run("use RemoteAddr as fallback", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		request.RemoteAddr = testRemoteAddr

		ip := getClientIP(request)
		assert.Equal(t, testRemoteAddr, ip)
	})

	t.Run("X-Forwarded-For takes precedence over X-Real-IP", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		request.Header.Set("X-Forwarded-For", "203.0.113.1")
		request.Header.Set("X-Real-IP", "203.0.113.2")

		ip := getClientIP(request)
		assert.Equal(t, "203.0.113.1", ip)
	})
}

//nolint:paralleltest // sequential execution required to avoid redis key conflicts
func TestGlobalRateLimit(t *testing.T) {
	t.Run("allow requests within limit", func(t *testing.T) {
		log := logger.InitForTest(t)
		redisClient := redis.InitForTest(t)
		middleware := GlobalRateLimit(10, 1*time.Second, log, redisClient)
		handler := createTestRateLimitHandler(t, middleware)

		// make requests
		for range 5 {
			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.NotEmpty(t, recorder.Header().Get("X-Ratelimit-Limit"))
			assert.NotEmpty(t, recorder.Header().Get("X-Ratelimit-Remaining"))
			assert.NotEmpty(t, recorder.Header().Get("X-Ratelimit-Reset"))
		}

		// wait for rate limit window to expire
		time.Sleep(1100 * time.Millisecond)
	})

	t.Run("reject requests exceeding limit", func(t *testing.T) {
		log := logger.InitForTest(t)
		redisClient := redis.InitForTest(t)

		limit := 3
		middleware := GlobalRateLimit(limit, 1*time.Second, log, redisClient)
		handler := createTestRateLimitHandler(t, middleware)

		// make requests up to limit
		for range limit {
			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			assert.Equal(t, http.StatusOK, recorder.Code)

			time.Sleep(50 * time.Millisecond)
		}

		time.Sleep(50 * time.Millisecond)

		// next request should be rate limited
		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

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
			func(log logger.Client, redis redis.Client) func(http.Handler) http.Handler {
				return IPRateLimit(limit, 1*time.Second, log, redis)
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
		log := logger.InitForTest(t)
		redisClient := redis.InitForTest(t)

		limit := 3
		middleware := EndpointRateLimit(limit, 1*time.Second, log, redisClient)
		handler := createTestRateLimitHandler(t, middleware)

		// make requests to /test endpoint
		for range limit {
			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			request.Header.Set("X-Forwarded-For", testIP1)

			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			assert.Equal(t, http.StatusOK, recorder.Code)

			time.Sleep(50 * time.Millisecond)
		}

		time.Sleep(50 * time.Millisecond)

		// next request to /test should be rate limited
		request1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		request1.Header.Set("X-Forwarded-For", testIP1)

		recorder1 := httptest.NewRecorder()

		handler.ServeHTTP(recorder1, request1)

		assert.Equal(t, http.StatusTooManyRequests, recorder1.Code)

		// request to different endpoint should succeed
		request2 := httptest.NewRequest(http.MethodGet, "/other", nil)
		request2.Header.Set("X-Forwarded-For", testIP1)

		recorder2 := httptest.NewRecorder()

		handler.ServeHTTP(recorder2, request2)

		assert.Equal(t, http.StatusOK, recorder2.Code)
	})
}

//nolint:paralleltest // sequential execution required to avoid redis key conflicts
func TestRateLimitHeaders(t *testing.T) {
	t.Run("set rate limit headers", func(t *testing.T) {
		log := logger.InitForTest(t)
		redisClient := redis.InitForTest(t)

		limit := 10
		middleware := GlobalRateLimit(limit, 1*time.Second, log, redisClient)
		handler := createTestRateLimitHandler(t, middleware)

		// make request
		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

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
	redis redis.Client,
	key string,
	limit int,
	window time.Duration,
) (bool, int, int, time.Time, error) {
	t.Helper()

	return checkRateLimit(context.Background(), redis, key, limit, window)
}

//nolint:paralleltest // sequential execution required to avoid redis key conflicts
func TestCheckRateLimit(t *testing.T) {
	t.Run("check rate limit successfully", func(t *testing.T) {
		redisClient := redis.InitForTest(t)
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
		redisClient := redis.InitForTest(t)
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
