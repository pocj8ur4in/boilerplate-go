package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/database"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/jwt"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/redis"
)

// setupTestHandler creates a handler for testings.
func setupTestHandler(t *testing.T) *Handler {
	t.Helper()

	log, err := logger.New(&logger.Config{})
	require.NoError(t, err)

	// create JWT for testing
	jwtService, err := jwt.New(&jwt.Config{})
	require.NoError(t, err)

	// try to connect to test database
	dbConn, err := database.New(&database.Config{Port: &[]int{35432}[0]})
	if err != nil {
		t.Logf("failed to connect to test database: %v", err)
	}

	// try to connect to test redis
	redisConn, err := redis.New(&redis.Config{Addrs: []string{"localhost:36379"}})
	if err != nil {
		t.Logf("failed to connect to test redis: %v", err)
	}

	handler := &Handler{
		logger: log,
		db:     dbConn,
		redis:  redisConn,
		jwt:    jwtService,
	}

	return handler
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("create new handler successfully", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		// create JWT for testing
		jwtService, err := jwt.New(&jwt.Config{})
		require.NoError(t, err)

		// try to connect to test database
		dbConn, _ := database.New(&database.Config{Port: &[]int{35432}[0]})

		// try to connect to test redis
		redisConn, _ := redis.New(&redis.Config{Addrs: []string{"localhost:36379"}})

		handler := New(log, dbConn, redisConn, jwtService)

		require.NotNil(t, handler)
		assert.IsType(t, &Handler{}, handler)
	})
}

func TestSendResponse(t *testing.T) {
	t.Parallel()

	t.Run("send response successfully", func(t *testing.T) {
		t.Parallel()

		handler := setupTestHandler(t)

		// create test response recorder
		recorder := httptest.NewRecorder()

		// test data
		testData := map[string]interface{}{
			"message": "success",
			"code":    200,
		}

		// send response
		handler.sendResponse(recorder, http.StatusOK, testData)

		// verify status code
		assert.Equal(t, http.StatusOK, recorder.Code)

		// verify content type
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

		// verify response body contains expected data
		assert.Contains(t, recorder.Body.String(), "success")
		assert.Contains(t, recorder.Body.String(), "200")
	})

	t.Run("send response with different status code", func(t *testing.T) {
		t.Parallel()

		handler := setupTestHandler(t)

		recorder := httptest.NewRecorder()

		testData := map[string]interface{}{
			"error": "not found",
		}

		handler.sendResponse(recorder, http.StatusNotFound, testData)

		assert.Equal(t, http.StatusNotFound, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
		assert.Contains(t, recorder.Body.String(), "not found")
	})

	t.Run("send response with empty data", func(t *testing.T) {
		t.Parallel()

		handler := setupTestHandler(t)

		recorder := httptest.NewRecorder()

		handler.sendResponse(recorder, http.StatusNoContent, map[string]interface{}{})

		assert.Equal(t, http.StatusNoContent, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	})
}

func TestSendError(t *testing.T) {
	t.Parallel()

	t.Run("send error response successfully", func(t *testing.T) {
		t.Parallel()

		handler := setupTestHandler(t)

		recorder := httptest.NewRecorder()

		handler.sendError(recorder, http.StatusBadRequest, "invalid request")

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
		assert.Contains(t, recorder.Body.String(), "error")
		assert.Contains(t, recorder.Body.String(), "invalid request")
	})

	t.Run("send error with different status codes", func(t *testing.T) {
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

				handler := setupTestHandler(t)
				recorder := httptest.NewRecorder()

				handler.sendError(recorder, testCase.statusCode, testCase.message)

				assert.Equal(t, testCase.statusCode, recorder.Code)
				assert.Contains(t, recorder.Body.String(), testCase.message)
			})
		}
	})
}

func TestNewModule(t *testing.T) {
	t.Parallel()

	t.Run("return fx.Option", func(t *testing.T) {
		t.Parallel()

		module := NewModule()

		require.NotNil(t, module)
	})
}
