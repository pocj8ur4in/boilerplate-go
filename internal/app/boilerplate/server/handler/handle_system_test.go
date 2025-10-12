package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
)

func TestStatusCheck(t *testing.T) {
	t.Parallel()

	t.Run("status check returns not implemented", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		handler := &Handler{
			logger: log,
		}

		// create test request
		req := httptest.NewRequest(http.MethodGet, "/status", nil)
		recorder := httptest.NewRecorder()

		// call handler
		handler.StatusCheck(recorder, req)

		// verify response
		assert.Equal(t, http.StatusNotImplemented, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	})

	t.Run("status check with different request methods", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		handler := &Handler{
			logger: log,
		}

		methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}

		for _, method := range methods {
			t.Run(method, func(t *testing.T) {
				t.Parallel()

				req := httptest.NewRequest(method, "/status", nil)
				recorder := httptest.NewRecorder()

				handler.StatusCheck(recorder, req)

				assert.Equal(t, http.StatusNotImplemented, recorder.Code)
			})
		}
	})
}

func TestHealthCheck(t *testing.T) {
	t.Parallel()

	t.Run("health check returns not implemented", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		handler := &Handler{
			logger: log,
		}

		// create test request
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		recorder := httptest.NewRecorder()

		// call handler
		handler.HealthCheck(recorder, req)

		// verify response
		assert.Equal(t, http.StatusNotImplemented, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	})

	t.Run("health check response format", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		handler := &Handler{
			logger: log,
		}

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		recorder := httptest.NewRecorder()

		handler.HealthCheck(recorder, req)

		assert.Equal(t, http.StatusNotImplemented, recorder.Code)
		assert.NotEmpty(t, recorder.Body.String())
	})
}

func TestHandleMetrics(t *testing.T) {
	t.Parallel()

	t.Run("metrics handler returns not implemented", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		handler := &Handler{
			logger: log,
		}

		// create test request
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		recorder := httptest.NewRecorder()

		// call handler
		handler.HandleMetrics(recorder, req)

		// verify response
		assert.Equal(t, http.StatusNotImplemented, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	})

	t.Run("metrics handler with query parameters", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		handler := &Handler{
			logger: log,
		}

		req := httptest.NewRequest(http.MethodGet, "/metrics?format=json", nil)
		recorder := httptest.NewRecorder()

		handler.HandleMetrics(recorder, req)

		assert.Equal(t, http.StatusNotImplemented, recorder.Code)
	})
}

func TestSystemHandlersIntegration(t *testing.T) {
	t.Parallel()

	t.Run("all system handlers respond correctly", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		handler := &Handler{
			logger: log,
		}

		endpoints := []struct {
			name    string
			path    string
			handler func(http.ResponseWriter, *http.Request)
		}{
			{"status", "/status", handler.StatusCheck},
			{"health", "/health", handler.HealthCheck},
			{"metrics", "/metrics", handler.HandleMetrics},
		}

		for _, endpoint := range endpoints {
			t.Run(endpoint.name, func(t *testing.T) {
				t.Parallel()

				req := httptest.NewRequest(http.MethodGet, endpoint.path, nil)
				recorder := httptest.NewRecorder()

				endpoint.handler(recorder, req)

				assert.Equal(t, http.StatusNotImplemented, recorder.Code)
				assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
			})
		}
	})
}
