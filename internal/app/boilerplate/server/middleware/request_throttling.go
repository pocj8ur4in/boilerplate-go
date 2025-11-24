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
	// ErrQueueFull returned when the request queue is full.
	ErrQueueFull = errors.New("request queue is full")

	// ErrInvalidThrottleType returned when the throttle type is unknown.
	ErrInvalidThrottleType = errors.New("invalid throttle type")

	// ErrInvalidThrottleScriptResult returned when the throttle script result is invalid.
	ErrInvalidThrottleScriptResult = errors.New("invalid throttle script result")

	// ErrFailedToParseThrottleScriptResult returned when the throttle script result is failed to parse.
	ErrFailedToParseThrottleScriptResult = errors.New("failed to parse throttle script result")
)

// ThrottleType represents the type of request throttling.
type ThrottleType string

const (
	// ThrottleTypeGlobal throttles requests globally.
	ThrottleTypeGlobal ThrottleType = "global"

	// ThrottleTypeIP throttles requests per IP address.
	ThrottleTypeIP ThrottleType = "ip"

	// ThrottleTypeEndpoint throttles requests per endpoint.
	ThrottleTypeEndpoint ThrottleType = "endpoint"
)

// ThrottleConfig represents configuration for request throttling.
type ThrottleConfig struct {
	// MaxConcurrent is the maximum number of concurrent requests.
	MaxConcurrent int `env:"MAX_CONCURRENT" envDefault:"100" json:"max_concurrent"`

	// MaxQueueSize is the maximum size of the waiting queue.
	MaxQueueSize int `env:"MAX_QUEUE_SIZE" envDefault:"100" json:"max_queue_size"`

	// Timeout is the maximum time a request can wait in the queue.
	Timeout time.Duration `env:"TIMEOUT" envDefault:"10s" json:"timeout"`

	// MinDelay is the minimum delay between processing requests.
	MinDelay time.Duration `env:"MIN_DELAY" envDefault:"1s" json:"min_delay"`

	// PollInterval is the interval for polling queue status.
	PollInterval time.Duration `env:"POLL_INTERVAL" envDefault:"1s" json:"poll_interval"`
}

// GlobalRequestThrottle is a middleware that throttles requests globally.
func GlobalRequestThrottle(
	config ThrottleConfig,
	log logger.Client,
	redis redis.Client,
) func(next http.Handler) http.Handler {
	return requestThrottle(ThrottleTypeGlobal, config, log, redis)
}

// IPRequestThrottle is a middleware that throttles requests per IP address.
func IPRequestThrottle(
	config ThrottleConfig,
	log logger.Client,
	redis redis.Client,
) func(next http.Handler) http.Handler {
	return requestThrottle(ThrottleTypeIP, config, log, redis)
}

// EndpointRequestThrottle is a middleware that throttles requests per endpoint.
func EndpointRequestThrottle(
	config ThrottleConfig,
	log logger.Client,
	redis redis.Client,
) func(next http.Handler) http.Handler {
	return requestThrottle(ThrottleTypeEndpoint, config, log, redis)
}

// requestThrottle is a common function for throttling requests.
func requestThrottle(
	throttleType ThrottleType,
	config ThrottleConfig,
	log logger.Client,
	redis redis.Client,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := request.Context()

			key, err := generateThrottleKey(throttleType, request)
			if err != nil {
				log.Error("throttle key generation failed", "error", err)
				next.ServeHTTP(writer, request)

				return
			}

			acquired, position, err := tryAcquireSlot(ctx, config, redis, key)
			if err != nil {
				log.Error("failed to acquire throttle slot", "error", err, "key", key)
				next.ServeHTTP(writer, request)

				return
			}

			// enter waiting queue if not acquired immediately
			if !acquired {
				if !waitForThrottleSlot(ctx, config, log, redis, key, position, writer, request, next) {
					return
				}
			}

			writer.Header().Set("X-Throttle-Queue-Position", "0")
			writer.Header().Set("X-Throttle-Max-Concurrent", strconv.Itoa(config.MaxConcurrent))

			// sleep for minimum delay if configured
			if config.MinDelay > 0 {
				time.Sleep(config.MinDelay)
			}

			next.ServeHTTP(writer, request)

			// release slot after request processing
			if err = releaseSlot(ctx, redis, key); err != nil {
				log.Error("failed to release slot", "error", err, "key", key)
			}
		})
	}
}

// waitForThrottleSlot waits for a throttle slot to become available.
func waitForThrottleSlot(
	ctx context.Context,
	config ThrottleConfig,
	log logger.Client,
	redis redis.Client,
	key string,
	position int,
	writer http.ResponseWriter,
	request *http.Request,
	next http.Handler,
) bool {
	log.Debug("request queued", "key", key, "queue_position", position)

	writer.Header().Set("X-Throttle-Queue-Position", strconv.Itoa(position))
	writer.Header().Set("X-Throttle-Max-Concurrent", strconv.Itoa(config.MaxConcurrent))

	waitCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	// wait for slot to become available
	if err := waitForSlot(waitCtx, config, log, redis, key); err != nil {
		// return false if timeout exceeded
		if errors.Is(err, context.DeadlineExceeded) {
			log.Warn("throttle timeout exceeded", "key", key, "timeout", config.Timeout)

			// remove from queue on timeout
			_ = removeFromQueue(ctx, redis, key)

			http.Error(writer, "request timeout: server is too busy", http.StatusServiceUnavailable)

			return false
		}

		log.Error("error while waiting for slot", "error", err, "key", key)
		next.ServeHTTP(writer, request)

		return false
	}

	return true
}

// generateThrottleKey generates a redis key based on throttle type.
func generateThrottleKey(throttleType ThrottleType, request *http.Request) (string, error) {
	switch throttleType {
	case ThrottleTypeGlobal:
		return "throttle:global", nil
	case ThrottleTypeIP:
		clientIP := getClientIP(request)

		return "throttle:ip:" + clientIP, nil
	case ThrottleTypeEndpoint:
		clientIP := getClientIP(request)
		endpoint := request.Method + ":" + request.URL.Path

		return "throttle:endpoint:" + clientIP + ":" + endpoint, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidThrottleType, throttleType)
	}
}

// acquireSlotScript is a lua script for atomic slot acquisition (returns: [acquired, position]).
const acquireSlotScript = `
-- get keys from arguments
local active_key = KEYS[1] .. ':active'
local queue_key = KEYS[1] .. ':queue'
local max_concurrent = tonumber(ARGV[1])
local max_queue = tonumber(ARGV[2])
local timestamp = tonumber(ARGV[3])

-- check current active count
local active_count = redis.call('SCARD', active_key)

-- if slot available, acquire immediately
if active_count < max_concurrent then
	redis.call('SADD', active_key, timestamp)
	return {1, 0}
end

-- check queue size
local queue_size = redis.call('LLEN', queue_key)
if queue_size >= max_queue then
	return {0, -1}  -- queue full
end

-- add to queue
redis.call('RPUSH', queue_key, timestamp)
local position = redis.call('LLEN', queue_key)

return {0, position}
`

// tryAcquireSlot tries to acquire a slot for request processing.
func tryAcquireSlot(
	ctx context.Context,
	config ThrottleConfig,
	redis redis.Client,
	key string,
) (bool, int, error) {
	timestamp := time.Now().UnixNano()

	result, err := redis.Eval(
		ctx,
		acquireSlotScript,
		[]string{key},
		config.MaxConcurrent,
		config.MaxQueueSize,
		timestamp,
	).Result()
	if err != nil {
		return false, 0, fmt.Errorf("failed to execute throttle script: %w", err)
	}

	values, ok := result.([]interface{})
	if !ok || len(values) != 2 {
		return false, 0, fmt.Errorf("%w: %v", ErrInvalidThrottleScriptResult, result)
	}

	acquired, ok1 := values[0].(int64)
	position, ok2 := values[1].(int64)

	if !ok1 || !ok2 {
		return false, 0, fmt.Errorf("%w: %v", ErrFailedToParseThrottleScriptResult, result)
	}

	// check if queue is full
	if position == -1 {
		return false, 0, ErrQueueFull
	}

	return acquired == 1, int(position), nil
}

// moveFromQueueToActiveScript is a lua script to try moving from queue to active (returns: [acquired]).
const moveFromQueueToActiveScript = `
-- get keys from arguments
local active_key = KEYS[1]
local queue_key = KEYS[2]
local max_concurrent = tonumber(ARGV[1])

-- check if queue is empty
local queue_size = redis.call('LLEN', queue_key)
if queue_size == 0 then
	return 0  -- queue is empty
end

-- check if slot is available
local active_count = redis.call('SCARD', active_key)
if active_count >= max_concurrent then
	return 0  -- still no slot
end

-- move from queue to active (get first item from queue)
local timestamp = redis.call('LPOP', queue_key)
if timestamp then
	redis.call('SADD', active_key, timestamp)
	return 1  -- acquired
end

return 0  -- failed to acquire
`

// waitForSlot waits for an available slot to process the request.
func waitForSlot(
	ctx context.Context,
	config ThrottleConfig,
	log logger.Client,
	redis redis.Client,
	key string,
) error {
	queueKey := key + ":queue"
	activeKey := key + ":active"

	// create ticker for polling queue status
	ticker := time.NewTicker(config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		case <-ticker.C:
			result, err := redis.Eval(
				ctx,
				moveFromQueueToActiveScript,
				[]string{activeKey, queueKey},
				config.MaxConcurrent,
			).Result()
			if err != nil {
				log.Error("failed to check queue position", "error", err)

				continue
			}

			if acquired, ok := result.(int64); ok && acquired == 1 {
				return nil
			}
		}
	}
}

// removeFromActiveSetScript is a lua script to remove a request from the active set (returns: [removed]).
const removeFromActiveSetScript = `
-- get key from arguments
local active_key = KEYS[1]

-- get one member and remove it
local members = redis.call('SMEMBERS', active_key)
if #members > 0 then
	redis.call('SREM', active_key, members[1])
	return 1
end

return 0
`

// releaseSlot releases a slot after request processing.
func releaseSlot(ctx context.Context, redis redis.Client, key string) error {
	activeKey := key + ":active"

	_, err := redis.Eval(ctx, removeFromActiveSetScript, []string{activeKey}).Result()
	if err != nil {
		return fmt.Errorf("failed to release slot: %w", err)
	}

	return nil
}

// removeFromQueue removes a request from the waiting queue.
func removeFromQueue(ctx context.Context, redis redis.Client, key string) error {
	queueKey := key + ":queue"

	// remove last item from queue (LIFO for timeout)
	_, err := redis.RPop(ctx, queueKey).Result()
	if err != nil {
		return fmt.Errorf("failed to remove from queue: %w", err)
	}

	return nil
}
