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
func (h *Handler) StatusCheck(writer http.ResponseWriter, _ *http.Request) {
	h.sendResponse(writer, http.StatusOK, map[string]interface{}{})
}

// HealthCheck handles GET /health endpoint.
func (h *Handler) HealthCheck(writer http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), healthCheckTimeout)
	defer cancel()

	// set response
	resp := api.SystemHealthCheckResponse{
		Timestamp: time.Now(),
		Services: api.SystemHealthCheckResponseServices{
			Database: true,
			Redis:    true,
		},
	}

	// check database health
	if err := h.db.PingContext(ctx); err != nil {
		h.logger.Error().Err(err).Msg("database health check failed")

		resp.Services.Database = false
	}

	// check redis health
	if err := h.redis.Ping(ctx).Err(); err != nil {
		h.logger.Error().Err(err).Msg("redis health check failed")

		resp.Services.Redis = false
	}

	h.sendResponse(writer, http.StatusOK, resp)
}

// HandleMetrics handles GET /metrics endpoint.
func (h *Handler) HandleMetrics(writer http.ResponseWriter, request *http.Request) {
	promhttp.Handler().ServeHTTP(writer, request)
}
