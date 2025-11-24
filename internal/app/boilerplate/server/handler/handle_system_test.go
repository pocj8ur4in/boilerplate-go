package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusCheck(t *testing.T) {
	t.Parallel()

	t.Run("StatusCheck returns OK", func(t *testing.T) {
		t.Parallel()

		handlerClient := InitForTest(t)
		request := httptest.NewRequest(http.MethodGet, "/status", nil)
		recorder := httptest.NewRecorder()

		handlerClient.StatusCheck(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	})

	t.Run("StatusCheck with different request methods", func(t *testing.T) {
		t.Parallel()

		handlerClient := InitForTest(t)
		methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}

		for _, method := range methods {
			t.Run(method, func(t *testing.T) {
				t.Parallel()

				request := httptest.NewRequest(method, "/status", nil)
				recorder := httptest.NewRecorder()

				handlerClient.StatusCheck(recorder, request)

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
			})
		}
	})
}

func TestHealthCheck(t *testing.T) {
	t.Parallel()

	t.Run("HealthCheck returns OK", func(t *testing.T) {
		t.Parallel()

		handlerClient := InitForTest(t)
		request := httptest.NewRequest(http.MethodGet, "/health", nil)
		recorder := httptest.NewRecorder()

		handlerClient.HealthCheck(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
		assert.NotEmpty(t, recorder.Body.String())
	})
}

func TestHandleMetrics(t *testing.T) {
	t.Parallel()

	t.Run("HandleMetrics returns prometheus metrics", func(t *testing.T) {
		t.Parallel()

		handlerClient := InitForTest(t)
		request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		recorder := httptest.NewRecorder()

		handlerClient.HandleMetrics(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Contains(t, recorder.Header().Get("Content-Type"), "text/plain")
		assert.NotEmpty(t, recorder.Body.String())
	})
}
