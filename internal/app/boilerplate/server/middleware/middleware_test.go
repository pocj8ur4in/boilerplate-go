package middleware

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
)

// testHandler is a simple handler that returns 200 OK.
func testHandler(statusCode int, message string) http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(statusCode)
		_, _ = writer.Write([]byte(message))
	}
}

func TestRequestID(t *testing.T) {
	t.Parallel()

	t.Run("add request ID to request", func(t *testing.T) {
		t.Parallel()

		handler := RequestID(testHandler(http.StatusOK, "test"))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("preserve existing request ID", func(t *testing.T) {
		t.Parallel()

		var capturedID string

		handler := RequestID(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			if id := request.Context().Value(middleware.RequestIDKey); id != nil {
				if idStr, ok := id.(string); ok {
					capturedID = idStr
				}
			}
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-Id", "test-request-id")

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.NotEmpty(t, capturedID)
	})
}

func TestRealIP(t *testing.T) {
	t.Parallel()

	t.Run("extract real IP from request", func(t *testing.T) {
		t.Parallel()

		handler := RealIP(testHandler(http.StatusOK, "test"))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Real-IP", "192.168.1.1")

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("handle X-Forwarded-For header", func(t *testing.T) {
		t.Parallel()

		handler := RealIP(testHandler(http.StatusOK, "test"))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "10.0.0.1, 192.168.1.1")

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestRecoverer(t *testing.T) {
	t.Parallel()

	t.Run("recover from panic", func(t *testing.T) {
		t.Parallel()

		handler := Recoverer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			panic("test panic")
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		// should not panic
		require.NotPanics(t, func() {
			handler.ServeHTTP(recorder, req)
		})

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})

	t.Run("pass through normal request", func(t *testing.T) {
		t.Parallel()

		handler := Recoverer(testHandler(http.StatusOK, "success"))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "success", recorder.Body.String())
	})
}

func TestSecurityHeaders(t *testing.T) {
	t.Parallel()

	t.Run("add all security headers", func(t *testing.T) {
		t.Parallel()

		handler := SecurityHeaders()(testHandler(http.StatusOK, "test"))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "nosniff", recorder.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "DENY", recorder.Header().Get("X-Frame-Options"))
		assert.Equal(t, "1; mode=block", recorder.Header().Get("X-XSS-Protection"))
		assert.Equal(t, "max-age=31536000; includeSubDomains; preload",
			recorder.Header().Get("Strict-Transport-Security"))
		assert.Equal(t, "strict-origin-when-cross-origin", recorder.Header().Get("Referrer-Policy"))
		assert.Equal(t, "off", recorder.Header().Get("X-DNS-Prefetch-Control"))
		assert.Equal(t, "geolocation=(), microphone=(), camera=()",
			recorder.Header().Get("Permissions-Policy"))
	})

	t.Run("headers are present for different status codes", func(t *testing.T) {
		t.Parallel()

		statusCodes := []int{
			http.StatusOK,
			http.StatusCreated,
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusInternalServerError,
		}

		for _, code := range statusCodes {
			handler := SecurityHeaders()(testHandler(code, "test"))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			assert.Equal(t, code, recorder.Code)
			assert.NotEmpty(t, recorder.Header().Get("X-Content-Type-Options"))
		}
	})
}

func TestRequestSize(t *testing.T) {
	t.Parallel()

	t.Run("allow request within size limit", func(t *testing.T) {
		t.Parallel()

		maxBytes := int64(1024) // 1KB
		handler := RequestSize(maxBytes)(testHandler(http.StatusOK, "success"))

		body := strings.NewReader("small body")
		req := httptest.NewRequest(http.MethodPost, "/test", body)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("reject request exceeding size limit", func(t *testing.T) {
		t.Parallel()

		maxBytes := int64(10) // 10 bytes
		handler := RequestSize(maxBytes)(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			// try to read the body
			body, err := io.ReadAll(request.Body)
			if err != nil {
				writer.WriteHeader(http.StatusBadRequest)

				return
			}

			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write(body)
		}))

		largeBody := strings.NewReader("this is a very large body that exceeds the limit")
		req := httptest.NewRequest(http.MethodPost, "/test", largeBody)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		// should reject the request
		assert.NotEqual(t, http.StatusOK, recorder.Code)
	})
}

func TestLogRequest(t *testing.T) {
	t.Parallel()

	t.Run("log request successfully", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{})
		require.NoError(t, err)

		handler := LogRequest(log)(testHandler(http.StatusOK, "test"))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("log request with request ID", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{})
		require.NoError(t, err)

		handler := RequestID(LogRequest(log)(testHandler(http.StatusOK, "test")))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestLogRequestHTTPMethods(t *testing.T) {
	t.Parallel()

	t.Run("log different HTTP methods", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{})
		require.NoError(t, err)

		methods := []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodPatch,
		}

		for _, method := range methods {
			handler := LogRequest(log)(testHandler(http.StatusOK, "test"))

			req := httptest.NewRequest(method, "/test", nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			assert.Equal(t, http.StatusOK, recorder.Code)
		}
	})
}

func TestLogRequestStatusCodes(t *testing.T) {
	t.Parallel()

	t.Run("log different status codes", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{})
		require.NoError(t, err)

		statusCodes := []int{
			http.StatusOK,
			http.StatusCreated,
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusNotFound,
			http.StatusInternalServerError,
		}

		for _, code := range statusCodes {
			handler := LogRequest(log)(testHandler(code, "test"))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			assert.Equal(t, code, recorder.Code)
		}
	})
}

func TestTimeout(t *testing.T) {
	t.Parallel()

	t.Run("complete request within timeout", func(t *testing.T) {
		t.Parallel()

		handler := Timeout(2 * time.Second)(testHandler(http.StatusOK, "success"))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("timeout slow request", func(t *testing.T) {
		t.Parallel()

		slowHandler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			select {
			case <-time.After(200 * time.Millisecond):
				writer.WriteHeader(http.StatusOK)
			case <-request.Context().Done():
				// don't write anything
				return
			}
		})

		handler := Timeout(50 * time.Millisecond)(slowHandler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusGatewayTimeout, recorder.Code)
	})
}

func TestMiddlewareChaining(t *testing.T) {
	t.Parallel()

	t.Run("chain multiple middlewares", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{})
		require.NoError(t, err)

		handler := RequestID(
			RealIP(
				Recoverer(
					SecurityHeaders()(
						LogRequest(log)(
							testHandler(http.StatusOK, "success"),
						),
					),
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

	t.Run("middleware order matters", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{})
		require.NoError(t, err)

		// recoverer should be before panic handler
		handler := Recoverer(
			http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				panic("test panic")
			}),
		)

		wrappedHandler := LogRequest(log)(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		require.NotPanics(t, func() {
			wrappedHandler.ServeHTTP(recorder, req)
		})
	})
}

func TestMiddlewareWithContext(t *testing.T) {
	t.Parallel()

	t.Run("middleware preserves context", func(t *testing.T) {
		t.Parallel()

		type contextKey string

		const testKey contextKey = "test"

		handler := RequestID(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			// verify context value is preserved
			if val := request.Context().Value(testKey); val != nil {
				if strVal, ok := val.(string); ok {
					_, _ = request.Context().Value(testKey).(string)

					assert.Equal(t, "test-value", strVal)
				}
			}
		}))

		ctx := context.WithValue(context.Background(), testKey, "test-value")
		req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)
	})
}

func TestMiddlewareWithLargePayload(t *testing.T) {
	t.Parallel()

	t.Run("handle large payload within limit", func(t *testing.T) {
		t.Parallel()

		log, err := logger.New(&logger.Config{})
		require.NoError(t, err)

		maxBytes := int64(1024 * 1024) // 1MB
		handler := RequestSize(maxBytes)(
			LogRequest(log)(
				http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					body, err := io.ReadAll(request.Body)
					if err != nil {
						writer.WriteHeader(http.StatusBadRequest)

						return
					}

					writer.WriteHeader(http.StatusOK)
					_, _ = writer.Write([]byte("received: " + strconv.Itoa(len(body))))
				}),
			),
		)

		// create 100KB payload
		payload := strings.Repeat("a", 100*1024)
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(payload))
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}
