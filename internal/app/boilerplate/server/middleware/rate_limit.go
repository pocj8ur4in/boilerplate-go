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

	// ErrInvalidRateLimitScriptResult returned when the rate limit script result is invalid.
	ErrInvalidRateLimitScriptResult = errors.New("invalid rate limit script result")

	// ErrFailedToParseRateLimitResult returned when the rate limit script result is failed to parse.
	ErrFailedToParseRateLimitResult = errors.New("failed to parse rate limit script result")
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
	Global RateLimitTypeConfig `envPrefix:"GLOBAL_" json:"global"`

	// IP is IP-based rate limit configuration.
	IP RateLimitTypeConfig `envPrefix:"IP_" json:"ip"`

	// Endpoint is endpoint-based rate limit configuration.
	Endpoint RateLimitTypeConfig `envPrefix:"ENDPOINT_" json:"endpoint"`
}

// RateLimitTypeConfig represents configuration for a specific rate limit type.
type RateLimitTypeConfig struct {
	// Enabled is whether this rate limit type is enabled.
	Enabled bool `env:"ENABLED" envDefault:"true" json:"enabled"`

	// Requests is the maximum number of requests allowed.
	Requests int `env:"REQUESTS" envDefault:"100" json:"requests"`

	// Window is the time window for rate limiting in seconds.
	Window int `env:"WINDOW" envDefault:"60" json:"window"`
}

// GlobalRateLimit is a middleware that limits the rate of requests globally.
func GlobalRateLimit(
	requests int,
	window time.Duration,
	log logger.Client,
	redis redis.Client,
) func(next http.Handler) http.Handler {
	return rateLimit(RateLimitTypeGlobal, requests, window, log, redis)
}

// IPRateLimit is a middleware that limits the rate of requests per IP address.
func IPRateLimit(
	requests int,
	window time.Duration,
	log logger.Client,
	redis redis.Client,
) func(next http.Handler) http.Handler {
	return rateLimit(RateLimitTypeIP, requests, window, log, redis)
}

// EndpointRateLimit is a middleware that limits the rate of requests per endpoint.
func EndpointRateLimit(
	requests int,
	window time.Duration,
	log logger.Client,
	redis redis.Client,
) func(next http.Handler) http.Handler {
	return rateLimit(RateLimitTypeEndpoint, requests, window, log, redis)
}

// rateLimit is a common function for limiting the rate of requests.
func rateLimit(
	limitType RateLimitType,
	requests int,
	window time.Duration,
	log logger.Client,
	redis redis.Client,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			// generate key
			key, err := generateRateLimitKey(limitType, request)
			if err != nil {
				log.Error("rate limit key generation failed", "error", err)
				next.ServeHTTP(writer, request)

				return
			}

			// check rate limit
			allowed, current, remaining, resetTime, err := checkRateLimit(
				request.Context(),
				redis,
				key,
				requests,
				window,
			)
			if err != nil {
				log.Error("rate limit check failed", "error", err, "key", key)
				next.ServeHTTP(writer, request)

				return
			}

			// set rate limit headers
			writer.Header().Set("X-Ratelimit-Limit", strconv.Itoa(requests))
			writer.Header().Set("X-Ratelimit-Remaining", strconv.Itoa(remaining))
			writer.Header().Set("X-Ratelimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

			// check if rate limit exceeded
			if !allowed {
				log.Debug("rate limit exceeded", "key", key, "current", current, "limit", requests)

				writer.Header().Set("Retry-After", strconv.Itoa(int(window.Seconds())))
				http.Error(writer, "Rate limit exceeded", http.StatusTooManyRequests)

				return
			}

			next.ServeHTTP(writer, request)
		})
	}
}

// generateRateLimitKey generates a redis key based on rate limit type.
func generateRateLimitKey(limitType RateLimitType, request *http.Request) (string, error) {
	switch limitType {
	case RateLimitTypeGlobal:
		return "rate_limit:global", nil
	case RateLimitTypeIP:
		clientIP := getClientIP(request)

		return "rate_limit:ip:" + clientIP, nil
	case RateLimitTypeEndpoint:
		clientIP := getClientIP(request)
		endpoint := request.Method + ":" + request.URL.Path

		return "rate_limit:endpoint:" + clientIP + ":" + endpoint, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnknownRateLimitType, limitType)
	}
}

// checkRateLimitScript is a lua script for atomic rate limit check (returns: [current_count, ttl_seconds]).
const checkRateLimitScript = `
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

// checkRateLimit checks if the request is allowed based on rate limit.
func checkRateLimit(
	ctx context.Context,
	redis redis.Client,
	key string,
	limit int,
	window time.Duration,
) (bool, int, int, time.Time, error) {
	// execute lua script
	result, err := redis.Eval(ctx, checkRateLimitScript, []string{key}, limit, int(window.Seconds())).Result()
	if err != nil {
		return false, 0, 0, time.Time{}, fmt.Errorf("%w: %w", ErrFailedToExecuteScript, err)
	}

	// get values from result
	values, ok := result.([]interface{})
	if !ok || len(values) != 2 {
		return false, 0, 0, time.Time{}, fmt.Errorf("%w: %v", ErrInvalidRateLimitScriptResult, result)
	}

	// get current count and TTL from values
	current, ok1 := values[0].(int64)

	ttl, ok2 := values[1].(int64)
	if !ok1 || !ok2 {
		return false, 0, 0, time.Time{}, fmt.Errorf("%w: %v", ErrFailedToParseRateLimitResult, result)
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
