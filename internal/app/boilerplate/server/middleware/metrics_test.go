package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsConfigSetDefault(t *testing.T) {
	t.Parallel()

	t.Run("set default values when config is empty", func(t *testing.T) {
		t.Parallel()

		config := &MetricsConfig{}
		config.SetDefault()

		require.NotNil(t, config.Enabled)
		require.NotNil(t, config.Path)
		require.NotNil(t, config.ExcludePaths)

		assert.True(t, *config.Enabled)
		assert.Equal(t, "/metrics", *config.Path)
		assert.Contains(t, config.ExcludePaths, "/health")
		assert.Contains(t, config.ExcludePaths, "/status")
	})

	t.Run("not override existing values", func(t *testing.T) {
		t.Parallel()

		enabled := false
		path := "/test-metrics"
		excludePaths := []string{"/test"}

		config := &MetricsConfig{
			Enabled:      &enabled,
			Path:         &path,
			ExcludePaths: excludePaths,
		}

		config.SetDefault()

		assert.False(t, *config.Enabled)
		assert.Equal(t, "/test-metrics", *config.Path)
		assert.Equal(t, []string{"/test"}, config.ExcludePaths)
	})
}

//nolint:funlen // Multiple test cases in one function
func TestMetrics(t *testing.T) {
	t.Parallel()

	t.Run("metrics middleware records request metrics", func(t *testing.T) {
		t.Parallel()

		registry := prometheus.NewRegistry()
		config := &MetricsConfig{}

		handler := Metrics(config, registry)(testHandler(http.StatusOK, "success"))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "success", recorder.Body.String())

		// verify metrics are registered
		metrics, err := registry.Gather()
		require.NoError(t, err)
		assert.NotEmpty(t, metrics)
	})

	t.Run("metrics middleware skips excluded paths", func(t *testing.T) {
		t.Parallel()

		registry := prometheus.NewRegistry()
		config := &MetricsConfig{
			ExcludePaths: []string{"/health"},
		}

		handler := Metrics(config, registry)(testHandler(http.StatusOK, "success"))

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("metrics middleware skips metrics endpoint", func(t *testing.T) {
		t.Parallel()

		registry := prometheus.NewRegistry()
		config := &MetricsConfig{}

		handler := Metrics(config, registry)(testHandler(http.StatusOK, "success"))

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("metrics middleware respects disabled flag", func(t *testing.T) {
		t.Parallel()

		registry := prometheus.NewRegistry()
		enabled := false
		config := &MetricsConfig{
			Enabled: &enabled,
		}

		handler := Metrics(config, registry)(testHandler(http.StatusOK, "success"))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("metrics middleware with nil config uses defaults", func(t *testing.T) {
		t.Parallel()

		registry := prometheus.NewRegistry()
		handler := Metrics(nil, registry)(testHandler(http.StatusOK, "success"))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("metrics middleware with nil registry uses default registry", func(t *testing.T) {
		t.Parallel()

		config := &MetricsConfig{}
		handler := Metrics(config, nil)(testHandler(http.StatusOK, "success"))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestMetricsWithDifferentStatusCodes(t *testing.T) {
	t.Parallel()

	statusCodes := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusInternalServerError,
	}

	for _, statusCode := range statusCodes {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			t.Parallel()

			registry := prometheus.NewRegistry()
			config := &MetricsConfig{}

			handler := Metrics(config, registry)(testHandler(statusCode, "response"))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			assert.Equal(t, statusCode, recorder.Code)
		})
	}
}

func TestMetricsWithDifferentMethods(t *testing.T) {
	t.Parallel()

	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			registry := prometheus.NewRegistry()
			config := &MetricsConfig{}

			handler := Metrics(config, registry)(testHandler(http.StatusOK, "success"))

			req := httptest.NewRequest(method, "/test", nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			assert.Equal(t, http.StatusOK, recorder.Code)
		})
	}
}

func TestMetricsWithRequestBody(t *testing.T) {
	t.Parallel()

	t.Run("metrics records request size", func(t *testing.T) {
		t.Parallel()

		registry := prometheus.NewRegistry()
		config := &MetricsConfig{}

		handler := Metrics(config, registry)(testHandler(http.StatusOK, "success"))

		body := strings.NewReader(`{"key": "value"}`)
		req := httptest.NewRequest(http.MethodPost, "/test", body)
		req.ContentLength = int64(body.Len())
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)

		// verify metrics are recorded
		metrics, err := registry.Gather()
		require.NoError(t, err)
		assert.NotEmpty(t, metrics)
	})

	t.Run("metrics handles request without body", func(t *testing.T) {
		t.Parallel()

		registry := prometheus.NewRegistry()
		config := &MetricsConfig{}

		handler := Metrics(config, registry)(testHandler(http.StatusOK, "success"))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestMetricsCollectorCreation(t *testing.T) {
	t.Parallel()

	t.Run("create metrics collector with custom registry", func(t *testing.T) {
		t.Parallel()

		registry := prometheus.NewRegistry()
		collector := newMetricsCollector(registry)

		require.NotNil(t, collector)
		require.NotNil(t, collector.requestsTotal)
		require.NotNil(t, collector.requestDuration)
		require.NotNil(t, collector.requestSize)
		require.NotNil(t, collector.responseSize)
		require.NotNil(t, collector.requestsInFlight)
	})
}

func TestShouldSkipMetrics(t *testing.T) {
	t.Parallel()

	t.Run("skip when metrics disabled", func(t *testing.T) {
		t.Parallel()

		enabled := false
		config := &MetricsConfig{
			Enabled: &enabled,
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		assert.True(t, shouldSkipMetrics(config, req))
	})

	t.Run("skip metrics endpoint", func(t *testing.T) {
		t.Parallel()

		path := "/metrics"
		config := &MetricsConfig{
			Path: &path,
		}
		config.SetDefault()

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		assert.True(t, shouldSkipMetrics(config, req))
	})

	t.Run("skip excluded paths", func(t *testing.T) {
		t.Parallel()

		config := &MetricsConfig{
			ExcludePaths: []string{"/health", "/status"},
		}
		config.SetDefault()

		req1 := httptest.NewRequest(http.MethodGet, "/health", nil)
		assert.True(t, shouldSkipMetrics(config, req1))

		req2 := httptest.NewRequest(http.MethodGet, "/status", nil)
		assert.True(t, shouldSkipMetrics(config, req2))
	})

	t.Run("do not skip normal paths", func(t *testing.T) {
		t.Parallel()

		config := &MetricsConfig{}
		config.SetDefault()

		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		assert.False(t, shouldSkipMetrics(config, req))
	})
}

func TestMetricsMiddlewareChaining(t *testing.T) {
	t.Parallel()

	t.Run("chain metrics with other middlewares", func(t *testing.T) {
		t.Parallel()

		registry := prometheus.NewRegistry()
		config := &MetricsConfig{}

		handler := RequestID(
			SecurityHeaders()(
				Metrics(config, registry)(
					testHandler(http.StatusOK, "success"),
				),
			),
		)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "success", recorder.Body.String())
		assert.NotEmpty(t, recorder.Header().Get("X-Content-Type-Options"))
	})
}
