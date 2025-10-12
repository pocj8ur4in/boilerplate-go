package handler

import (
	"net/http"
)

// StatusCheck handles GET /status endpoint.
func (h *Handler) StatusCheck(writer http.ResponseWriter, _ *http.Request) {
	h.sendResponse(writer, http.StatusNotImplemented, map[string]interface{}{})
}

// HealthCheck handles GET /health endpoint.
func (h *Handler) HealthCheck(writer http.ResponseWriter, _ *http.Request) {
	h.sendResponse(writer, http.StatusNotImplemented, map[string]interface{}{})
}

// HandleMetrics handles GET /metrics endpoint.
func (h *Handler) HandleMetrics(writer http.ResponseWriter, _ *http.Request) {
	h.sendResponse(writer, http.StatusNotImplemented, map[string]interface{}{})
}
