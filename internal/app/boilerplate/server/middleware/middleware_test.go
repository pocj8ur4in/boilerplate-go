package middleware

import (
	"compress/flate"
	"compress/gzip"
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

const (
	// contentEncodingGzip is the gzip compression format.
	contentEncodingGzip = "gzip"
)

// testHandler is a simple handler that returns given status code and message.
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

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

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

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		request.Header.Set("X-Request-Id", "test-request-id")

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.NotEmpty(t, capturedID)
	})
}

func TestRealIP(t *testing.T) {
	t.Parallel()

	t.Run("extract real IP from request", func(t *testing.T) {
		t.Parallel()

		handler := RealIP(testHandler(http.StatusOK, "test"))

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		request.Header.Set("X-Real-IP", "192.168.1.1")

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("handle X-Forwarded-For header", func(t *testing.T) {
		t.Parallel()

		handler := RealIP(testHandler(http.StatusOK, "test"))

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		request.Header.Set("X-Forwarded-For", "10.0.0.1, 192.168.1.1")

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

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

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		require.NotPanics(t, func() {
			handler.ServeHTTP(recorder, request)
		})

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})

	t.Run("pass through normal request", func(t *testing.T) {
		t.Parallel()

		handler := Recoverer(testHandler(http.StatusOK, "success"))

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "success", recorder.Body.String())
	})
}

func TestSecurityHeaders(t *testing.T) {
	t.Parallel()

	t.Run("add all security headers", func(t *testing.T) {
		t.Parallel()

		handler := SecurityHeaders()(testHandler(http.StatusOK, "test"))

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

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

		for _, statusCode := range statusCodes {
			handler := SecurityHeaders()(testHandler(statusCode, "test"))

			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			assert.Equal(t, statusCode, recorder.Code)
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
		request := httptest.NewRequest(http.MethodPost, "/test", body)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("reject request exceeding size limit", func(t *testing.T) {
		t.Parallel()

		maxBytes := int64(10) // 10 bytes
		handler := RequestSize(maxBytes)(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			body, err := io.ReadAll(request.Body)
			if err != nil {
				writer.WriteHeader(http.StatusBadRequest)

				return
			}

			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write(body)
		}))

		largeBody := strings.NewReader("this is a very large body that exceeds the limit")
		request := httptest.NewRequest(http.MethodPost, "/test", largeBody)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.NotEqual(t, http.StatusOK, recorder.Code)
	})
}

func TestLogRequest(t *testing.T) {
	t.Parallel()

	t.Run("log request", func(t *testing.T) {
		t.Parallel()

		log := logger.InitForTest(t)

		handler := LogRequest(log)(testHandler(http.StatusOK, "test"))

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestLogRequestHTTPMethods(t *testing.T) {
	t.Parallel()

	t.Run("log different HTTP methods", func(t *testing.T) {
		t.Parallel()

		log := logger.InitForTest(t)

		methods := []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodPatch,
		}

		for _, method := range methods {
			handler := LogRequest(log)(testHandler(http.StatusOK, "test"))

			request := httptest.NewRequest(method, "/test", nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			assert.Equal(t, http.StatusOK, recorder.Code)
		}
	})
}

func TestLogRequestStatusCodes(t *testing.T) {
	t.Parallel()

	t.Run("log different status codes", func(t *testing.T) {
		t.Parallel()

		log := logger.InitForTest(t)

		statusCodes := []int{
			http.StatusOK,
			http.StatusCreated,
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusNotFound,
			http.StatusInternalServerError,
		}

		for _, statusCode := range statusCodes {
			handler := LogRequest(log)(testHandler(statusCode, "test"))

			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			assert.Equal(t, statusCode, recorder.Code)
		}
	})
}

func TestTimeout(t *testing.T) {
	t.Parallel()

	t.Run("complete request within timeout", func(t *testing.T) {
		t.Parallel()

		handler := Timeout(2 * time.Second)(testHandler(http.StatusOK, "success"))

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

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

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusGatewayTimeout, recorder.Code)
	})
}

func TestCompressFormat(t *testing.T) {
	t.Parallel()

	largeBody := strings.Repeat("test response body that is large enough to be compressed by the middleware", 10)

	t.Run("compress response with gzip", func(t *testing.T) {
		t.Parallel()

		handler := Compress(gzip.DefaultCompression, contentEncodingGzip)(testHandler(http.StatusOK, largeBody))

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		request.Header.Set("Accept-Encoding", contentEncodingGzip)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)

		if recorder.Header().Get("Content-Encoding") == contentEncodingGzip {
			gzipReader, err := gzip.NewReader(recorder.Body)
			require.NoError(t, err)

			defer func() {
				_ = gzipReader.Close()
			}()

			decompressed, err := io.ReadAll(gzipReader)
			require.NoError(t, err)
			assert.Equal(t, largeBody, string(decompressed))
		}
	})

	t.Run("compress response with deflate", func(t *testing.T) {
		t.Parallel()

		handler := Compress(flate.DefaultCompression, "")(testHandler(http.StatusOK, largeBody))

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		request.Header.Set("Accept-Encoding", "deflate")

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)

		if recorder.Header().Get("Content-Encoding") == "deflate" {
			deflateReader := flate.NewReader(recorder.Body)

			defer func() {
				_ = deflateReader.Close()
			}()

			decompressed, err := io.ReadAll(deflateReader)
			require.NoError(t, err)
			assert.Equal(t, largeBody, string(decompressed))
		}
	})
}

func TestCompressWithLevels(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		level    int
		bodyChar string
	}{
		{
			name:     "compress with best compression level",
			level:    gzip.BestCompression,
			bodyChar: "aaaaaaaaaa",
		},
		{
			name:     "compress with best speed level",
			level:    gzip.BestSpeed,
			bodyChar: "bbbbbbbbbb",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			largeBody := strings.Repeat(testCase.bodyChar, 100) // 1000 chars
			handler := Compress(testCase.level, contentEncodingGzip)(testHandler(http.StatusOK, largeBody))

			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			request.Header.Set("Accept-Encoding", contentEncodingGzip)

			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			assert.Equal(t, http.StatusOK, recorder.Code)

			if recorder.Header().Get("Content-Encoding") == contentEncodingGzip {
				gzipReader, err := gzip.NewReader(recorder.Body)
				require.NoError(t, err)

				defer func() {
					_ = gzipReader.Close()
				}()

				decompressed, err := io.ReadAll(gzipReader)
				require.NoError(t, err)
				assert.Equal(t, largeBody, string(decompressed))
			}
		})
	}
}

func TestCompressWithResponseSize(t *testing.T) {
	t.Parallel()

	t.Run("compress very large response", func(t *testing.T) {
		t.Parallel()

		largeBody := strings.Repeat("large response body that is large enough to be compressed by the middleware", 1000)
		handler := Compress(gzip.DefaultCompression, contentEncodingGzip)(testHandler(http.StatusOK, largeBody))

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		request.Header.Set("Accept-Encoding", contentEncodingGzip)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)

		// large responses should be compressed
		if recorder.Header().Get("Content-Encoding") == contentEncodingGzip {
			assert.Less(t, recorder.Body.Len(), len(largeBody))

			gzipReader, err := gzip.NewReader(recorder.Body)
			require.NoError(t, err)

			defer func() {
				_ = gzipReader.Close()
			}()

			decompressed, err := io.ReadAll(gzipReader)
			require.NoError(t, err)
			assert.Equal(t, largeBody, string(decompressed))
		}
	})

	t.Run("compress small response should not be compressed", func(t *testing.T) {
		t.Parallel()

		smallBody := "small response body that is not large enough to be compressed by the middleware"
		handler := Compress(gzip.DefaultCompression, contentEncodingGzip)(testHandler(http.StatusOK, smallBody))

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		request.Header.Set("Accept-Encoding", contentEncodingGzip)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, smallBody, recorder.Body.String())
	})
}

func TestCompressHeaders(t *testing.T) {
	t.Parallel()

	t.Run("no compression without Accept-Encoding header", func(t *testing.T) {
		t.Parallel()

		handler := Compress(gzip.DefaultCompression, contentEncodingGzip)(testHandler(http.StatusOK, "test response body"))

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Empty(t, recorder.Header().Get("Content-Encoding"))
		assert.Equal(t, "test response body", recorder.Body.String())
	})
}

func TestCompressDifferentStatusCodes(t *testing.T) {
	t.Parallel()

	statusCodes := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusAccepted,
		http.StatusNoContent,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	}

	for _, statusCode := range statusCodes {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			t.Parallel()

			handler := Compress(gzip.DefaultCompression, contentEncodingGzip)(testHandler(statusCode, "test"))

			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			request.Header.Set("Accept-Encoding", contentEncodingGzip)

			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			assert.Equal(t, statusCode, recorder.Code)
		})
	}
}

func TestCompressDifferentHTTPMethods(t *testing.T) {
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

			handler := Compress(gzip.DefaultCompression, contentEncodingGzip)(testHandler(http.StatusOK, "test"))

			request := httptest.NewRequest(method, "/test", nil)
			request.Header.Set("Accept-Encoding", contentEncodingGzip)

			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			assert.Equal(t, http.StatusOK, recorder.Code)
		})
	}
}

func TestMiddlewareChaining(t *testing.T) {
	t.Parallel()

	t.Run("chain multiple middlewares", func(t *testing.T) {
		t.Parallel()

		log := logger.InitForTest(t)

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

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "success", recorder.Body.String())
		assert.NotEmpty(t, recorder.Header().Get("X-Content-Type-Options"))
	})

	t.Run("middleware order matters", func(t *testing.T) {
		t.Parallel()

		log := logger.InitForTest(t)

		// recoverer should be before panic handler
		handler := Recoverer(
			http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				panic("test panic")
			}),
		)

		wrappedHandler := LogRequest(log)(handler)

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		require.NotPanics(t, func() {
			wrappedHandler.ServeHTTP(recorder, request)
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
		request := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)
	})
}

func TestMiddlewareWithLargePayload(t *testing.T) {
	t.Parallel()

	t.Run("handle large payload within limit", func(t *testing.T) {
		t.Parallel()

		log := logger.InitForTest(t)

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
		request := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(payload))
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}
