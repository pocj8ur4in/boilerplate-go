package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	t.Parallel()

	t.Run("init for test instance of handler", func(t *testing.T) {
		t.Parallel()

		handlerClient := InitForTest(t)

		require.NotNil(t, handlerClient)
		require.NotNil(t, handlerClient.log)
		require.NotNil(t, handlerClient.database)
		require.NotNil(t, handlerClient.jwt)
		require.NotNil(t, handlerClient.redis)
	})
}

func TestNewModule(t *testing.T) {
	t.Parallel()

	t.Run("create module for handler", func(t *testing.T) {
		t.Parallel()

		module := NewModule()

		require.NotNil(t, module)
	})
}

func TestSendResponse(t *testing.T) {
	t.Parallel()

	t.Run("send success response", func(t *testing.T) {
		t.Parallel()

		// create test response recorder
		handlerClient := InitForTest(t)
		recorder := httptest.NewRecorder()

		// test data
		testData := map[string]interface{}{
			"message": "success",
			"code":    200,
		}

		// send response
		handlerClient.sendResponse(recorder, http.StatusOK, testData)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
		assert.Contains(t, recorder.Body.String(), "success")
		assert.Contains(t, recorder.Body.String(), "200")
	})
}

func TestSendError(t *testing.T) {
	t.Parallel()

	t.Run("send error response with different status codes", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name       string
			statusCode int
			message    string
		}{
			{"bad request", http.StatusBadRequest, "bad request error"},
			{"unauthorized", http.StatusUnauthorized, "unauthorized error"},
			{"forbidden", http.StatusForbidden, "forbidden error"},
			{"not found", http.StatusNotFound, "not found error"},
			{"internal server error", http.StatusInternalServerError, "internal error"},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				t.Parallel()

				handlerClient := InitForTest(t)
				recorder := httptest.NewRecorder()

				handlerClient.sendError(recorder, testCase.statusCode, "GENERIC_ERROR", testCase.message)

				assert.Equal(t, testCase.statusCode, recorder.Code)
				assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
				assert.Contains(t, recorder.Body.String(), testCase.message)
			})
		}
	})
}
