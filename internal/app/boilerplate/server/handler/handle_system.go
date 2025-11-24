package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/pocj8ur4in/boilerplate-go/internal/gen/api"
)

const (
	// healthCheckTimeout is the timeout for health check operations.
	healthCheckTimeout = 5 * time.Second
)

// StatusCheck handles GET /status endpoint.
func (c *client) StatusCheck(writer http.ResponseWriter, _ *http.Request) {
	c.sendResponse(writer, http.StatusOK, map[string]interface{}{})
}

// HealthCheck handles GET /health endpoint.
func (c *client) HealthCheck(writer http.ResponseWriter, request *http.Request) {
	// get logger from context
	log := c.log.Ctx(request.Context())

	// create context with timeout
	ctx, cancel := context.WithTimeout(request.Context(), healthCheckTimeout)

	defer func() {
		cancel()
		log.Debug("health check context cancelled by timeout")
	}()

	// set response
	response := api.SystemHealthCheckResponse{
		Timestamp: time.Now(),
		Services: api.SystemHealthCheckResponseServices{
			Database: true,
			Redis:    true,
		},
	}

	// check database health
	if err := c.db.PingContext(ctx); err != nil {
		log.Error("database health check failed", "error", err)

		response.Services.Database = false
	}

	// check redis health
	if err := c.redis.Ping(ctx).Err(); err != nil {
		log.Error("redis health check failed", "error", err)

		response.Services.Redis = false
	}

	c.sendResponse(writer, http.StatusOK, response)
}

// HandleMetrics handles GET /metrics endpoint.
func (c *client) HandleMetrics(writer http.ResponseWriter, request *http.Request) {
	promhttp.Handler().ServeHTTP(writer, request)
}
