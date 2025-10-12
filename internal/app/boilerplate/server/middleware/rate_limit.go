package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/redis"
)

var (
	// ErrUnknownRateLimitType returned when the rate limit type is unknown.
	ErrUnknownRateLimitType = errors.New("unknown rate limit type")

	// ErrFailedToExecuteScript returned when the rate limit script is failed to execute.
	ErrFailedToExecuteScript = errors.New("failed to execute rate limit script")

	// ErrInvalidScriptResult returned when the rate limit script result is invalid.
	ErrInvalidScriptResult = errors.New("invalid rate limit script result")

	// ErrFailedToParseResult returned when the rate limit script result is failed to parse.
	ErrFailedToParseResult = errors.New("failed to parse rate limit script result")
)

// RateLimitType represents the type of rate limiting.
type RateLimitType string

const (
	// RateLimitTypeGlobal limits requests globally.
	RateLimitTypeGlobal RateLimitType = "global"

	// RateLimitTypeIP limits requests per IP address.
	RateLimitTypeIP RateLimitType = "ip"

	// RateLimitTypeEndpoint limits requests per endpoint.
	RateLimitTypeEndpoint RateLimitType = "endpoint"
)

// RateLimitConfig represents configuration for rate limiting.
type RateLimitConfig struct {
	// Global is global rate limit configuration.
	Global *RateLimitTypeConfig `json:"global"`

	// IP is IP-based rate limit configuration.
	IP *RateLimitTypeConfig `json:"ip"`

	// Endpoint is endpoint-based rate limit configuration.
	Endpoint *RateLimitTypeConfig `json:"endpoint"`
}

// RateLimitTypeConfig represents configuration for a specific rate limit type.
type RateLimitTypeConfig struct {
	// Enabled is whether this rate limit type is enabled.
	Enabled *bool `json:"enabled"`

	// Requests is the maximum number of requests allowed.
	Requests *int `json:"requests"`

	// Window is the time window for rate limiting in seconds.
	Window *int `json:"window"`
}

// GlobalRateLimit is a middleware that limits the rate of requests globally.
func GlobalRateLimit(
	requests int,
	window time.Duration,
	redis *redis.Redis,
	logger *logger.Logger,
) func(next http.Handler) http.Handler {
	return rateLimit(RateLimitTypeGlobal, requests, window, redis, logger)
}

// IPRateLimit is a middleware that limits the rate of requests per IP address.
func IPRateLimit(
	requests int,
	window time.Duration,
	redis *redis.Redis,
	logger *logger.Logger,
) func(next http.Handler) http.Handler {
	return rateLimit(RateLimitTypeIP, requests, window, redis, logger)
}

// EndpointRateLimit is a middleware that limits the rate of requests per endpoint.
func EndpointRateLimit(
	requests int,
	window time.Duration,
	redis *redis.Redis,
	logger *logger.Logger,
) func(next http.Handler) http.Handler {
	return rateLimit(RateLimitTypeEndpoint, requests, window, redis, logger)
}

// rateLimit is a common function for limiting the rate of requests.
func rateLimit(
	limitType RateLimitType,
	requests int,
	window time.Duration,
	redis *redis.Redis,
	logger *logger.Logger,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			// generate key
			key, err := generateRateLimitKey(limitType, request)
			if err != nil {
				logger.Error().Err(err).Msg("rate limit key generation failed")
				next.ServeHTTP(writer, request)

				return
			}

			// check rate limit
			allowed, current, remaining, resetTime, err := checkRateLimit(
				request.Context(),
				redis,
				*key,
				requests,
				window,
			)
			if err != nil {
				logger.Error().Err(err).Str("key", *key).Msg("rate limit check failed")
				next.ServeHTTP(writer, request)

				return
			}

			// set rate limit headers
			writer.Header().Set("X-Ratelimit-Limit", strconv.Itoa(requests))
			writer.Header().Set("X-Ratelimit-Remaining", strconv.Itoa(remaining))
			writer.Header().Set("X-Ratelimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

			// check if rate limit exceeded
			if !allowed {
				logger.Debug().
					Str("key", *key).
					Int("current", current).
					Int("limit", requests).
					Msg("rate limit exceeded")

				writer.Header().Set("Retry-After", strconv.Itoa(int(window.Seconds())))
				http.Error(writer, "Rate limit exceeded", http.StatusTooManyRequests)

				return
			}

			next.ServeHTTP(writer, request)
		})
	}
}

// generateRateLimitKey generates a redis key based on rate limit type.
func generateRateLimitKey(limitType RateLimitType, request *http.Request) (*string, error) {
	switch limitType {
	case RateLimitTypeGlobal:
		return &[]string{"rate_limit:global"}[0], nil
	case RateLimitTypeIP:
		clientIP := getClientIP(request)

		return &[]string{"rate_limit:ip:" + clientIP}[0], nil
	case RateLimitTypeEndpoint:
		clientIP := getClientIP(request)
		endpoint := request.Method + ":" + request.URL.Path

		return &[]string{"rate_limit:endpoint:" + clientIP + ":" + endpoint}[0], nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownRateLimitType, limitType)
	}
}

// checkRateLimit checks if the request is allowed based on rate limit.
func checkRateLimit(
	ctx context.Context,
	redis *redis.Redis,
	key string,
	limit int,
	window time.Duration,
) (bool, int, int, time.Time, error) {
	// lua script for atomic rate limit check (returns: [current_count, ttl_seconds])
	script := `
		-- get key and limit from arguments
		local key = KEYS[1]
		local limit = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])

		-- if key does not exist, set it to 1 and return [1, window]
		local current = redis.call('GET', key)
		if current == false then
			redis.call('SET', key, 1, 'EX', window)
			return {1, window}
		end

		-- increment count and get TTL
		local count = redis.call('INCR', key)
		local ttl = redis.call('TTL', key)

		-- return current count and TTL
		return {count, ttl}
	`

	// execute lua script
	result, err := redis.Eval(ctx, script, []string{key}, limit, int(window.Seconds())).Result()
	if err != nil {
		return false, 0, 0, time.Time{}, fmt.Errorf("%w: %w", ErrFailedToExecuteScript, err)
	}

	// get values from result
	values, ok := result.([]interface{})
	if !ok || len(values) != 2 {
		return false, 0, 0, time.Time{}, fmt.Errorf("%w: %v", ErrInvalidScriptResult, result)
	}

	// get current count and TTL from values
	current, ok1 := values[0].(int64)

	ttl, ok2 := values[1].(int64)
	if !ok1 || !ok2 {
		return false, 0, 0, time.Time{}, fmt.Errorf("%w: %v", ErrFailedToParseResult, result)
	}

	// calculate remaining and reset time
	remaining := limit - int(current)
	if remaining < 0 {
		remaining = 0
	}

	resetTime := time.Now().Add(time.Duration(ttl) * time.Second)
	allowed := current <= int64(limit)

	return allowed, int(current), remaining, resetTime, nil
}

// getClientIP extracts the client IP address from the request.
func getClientIP(request *http.Request) string {
	// check X-Forwarded-For header
	if xff := request.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}

	// check X-Real-IP header
	if xri := request.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// use RemoteAddr as fallback
	return request.RemoteAddr
}
