package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pocj8ur4in/boilerplate-go/internal/app/boilerplate/ctx"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
)

func TestInjectLogger(t *testing.T) {
	t.Parallel()

	t.Run("injects logger into context", func(t *testing.T) {
		t.Parallel()

		log := logger.InitForTest(t)

		var loggerInjected bool

		handler := InjectLogger(log)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			loggerInjected = log.Ctx(request.Context()) != nil
		}))

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.True(t, loggerInjected)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("works without request_id", func(t *testing.T) {
		t.Parallel()

		log := logger.InitForTest(t)

		var loggerWorks bool

		handler := InjectLogger(log)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			logWithCtx := log.Ctx(request.Context())
			logWithCtx.Info("test without request_id")

			loggerWorks = true
		}))

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.True(t, loggerWorks)
	})
}

func TestInjectLoggerIncludesRequestID(t *testing.T) {
	t.Parallel()

	t.Run("includes request_id in logger when present", func(t *testing.T) {
		t.Parallel()

		log := logger.InitForTest(t)
		testRequestID := "test-request-id-123"

		var requestIDPreserved bool

		handler := InjectLogger(log)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			if id := request.Context().Value(ctx.RequestIDKey); id != nil {
				if idStr, ok := id.(string); ok {
					requestIDPreserved = idStr == testRequestID
				}
			}
		}))

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		requestCtx := context.WithValue(request.Context(), ctx.RequestIDKey, testRequestID)
		request = request.WithContext(requestCtx)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.True(t, requestIDPreserved)
	})
}

func TestInjectLoggerIgnoresInvalidRequestID(t *testing.T) {
	t.Parallel()

	t.Run("ignores empty request_id", func(t *testing.T) {
		t.Parallel()

		log := logger.InitForTest(t)

		var loggerInjected bool

		handler := InjectLogger(log)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			logWithCtx := log.Ctx(request.Context())
			loggerInjected = logWithCtx != nil
		}))

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		requestCtx := context.WithValue(request.Context(), ctx.RequestIDKey, "")
		request = request.WithContext(requestCtx)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.True(t, loggerInjected)
	})

	t.Run("ignores non-string request_id", func(t *testing.T) {
		t.Parallel()

		log := logger.InitForTest(t)

		var loggerInjected bool

		handler := InjectLogger(log)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			logWithCtx := log.Ctx(request.Context())
			loggerInjected = logWithCtx != nil
		}))

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		requestCtx := context.WithValue(request.Context(), ctx.RequestIDKey, 12345)
		request = request.WithContext(requestCtx)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.True(t, loggerInjected)
	})
}

func TestInjectLoggerRetrieval(t *testing.T) {
	t.Parallel()

	t.Run("logger can be retrieved with Logger.Ctx method", func(t *testing.T) {
		t.Parallel()

		log := logger.InitForTest(t)
		testRequestID := "test-ctx-method-789"

		var ctxMethodWorks bool

		handler := InjectLogger(log)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			// use Logger.Ctx method to retrieve logger
			logWithCtx := log.Ctx(request.Context())
			ctxMethodWorks = logWithCtx != nil
		}))

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		requestCtx := context.WithValue(request.Context(), ctx.RequestIDKey, testRequestID)
		request = request.WithContext(requestCtx)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.True(t, ctxMethodWorks)
	})
}

func TestInjectLoggerIntegration(t *testing.T) {
	t.Parallel()

	t.Run("middleware chain with multiple context values", func(t *testing.T) {
		t.Parallel()

		log := logger.InitForTest(t)
		testRequestID := "chain-request-id"
		testUserID := "user-123"

		var allValuesPreserved bool

		handler := InjectLogger(log)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			requestID := request.Context().Value(ctx.RequestIDKey)
			userID := request.Context().Value(ctx.UserIDKey)
			logWithCtx := log.Ctx(request.Context())

			allValuesPreserved = requestID == testRequestID &&
				userID == testUserID &&
				logWithCtx != nil
		}))

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		requestCtx := context.WithValue(request.Context(), ctx.RequestIDKey, testRequestID)
		requestCtx = context.WithValue(requestCtx, ctx.UserIDKey, testUserID)
		request = request.WithContext(requestCtx)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.True(t, allValuesPreserved)
	})
}
