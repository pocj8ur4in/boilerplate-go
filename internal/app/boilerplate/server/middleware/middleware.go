// Package middleware provides middleware for server.
package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
)

// RequestID is a middleware that adds a request ID to the request.
func RequestID(next http.Handler) http.Handler {
	return middleware.RequestID(next)
}

// RealIP is a middleware that adds the real IP address to the request.
func RealIP(next http.Handler) http.Handler {
	return middleware.RealIP(next)
}

// Recoverer is a middleware that recovers from panics.
func Recoverer(next http.Handler) http.Handler {
	return middleware.Recoverer(next)
}

// SecurityHeaders is a middleware that adds security headers to responses.
func SecurityHeaders() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			// prevent MIME type sniffing
			writer.Header().Set("X-Content-Type-Options", "nosniff")

			// prevent clickjacking attacks
			writer.Header().Set("X-Frame-Options", "DENY")

			// enable XSS protection
			writer.Header().Set("X-XSS-Protection", "1; mode=block")

			// force HTTPS (adjust max-age as needed)
			writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

			// control referrer information
			writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// prevent DNS prefetching
			writer.Header().Set("X-DNS-Prefetch-Control", "off")

			// control browser features
			writer.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			next.ServeHTTP(writer, request)
		})
	}
}

// RequestSize is a middleware that sets a maximum request size.
func RequestSize(maxBytes int64) func(next http.Handler) http.Handler {
	return middleware.RequestSize(maxBytes)
}

// LogRequest is a middleware that logs HTTP requests.
func LogRequest(log logger.Client) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			// skip if log level is not debug
			if log.Level() > slog.LevelDebug {
				next.ServeHTTP(writer, request)

				return
			}

			start := time.Now()

			// wrap response writer to capture status code
			wrappedWriter := middleware.NewWrapResponseWriter(writer, request.ProtoMajor)

			// process request
			next.ServeHTTP(wrappedWriter, request)

			// get logger from context
			logWithCtx := log.Ctx(request.Context())

			// build log message for request
			logWithCtx.Debug("http request",
				"method", request.Method,
				"path", request.URL.Path,
				"remote_addr", request.RemoteAddr,
				"user_agent", request.UserAgent(),
				"status", wrappedWriter.Status(),
				"bytes", wrappedWriter.BytesWritten(),
				"duration", time.Since(start),
			)
		})
	}
}

// Timeout is a middleware that sets a timeout for the request.
func Timeout(timeout time.Duration) func(next http.Handler) http.Handler {
	return middleware.Timeout(timeout)
}

// Compress is a middleware that compresses the response.
func Compress(level int, format string) func(next http.Handler) http.Handler {
	return middleware.Compress(level, format)
}
