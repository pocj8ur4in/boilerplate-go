package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
)

// InjectLogger is a middleware that injects logger into context with request metadata.
func InjectLogger(log logger.Client) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			requestCtx := request.Context()

			// create logger with request metadata
			requestLogger := log.Ctx(requestCtx)

			// extract request_id from context
			if id := requestCtx.Value(middleware.RequestIDKey); id != nil {
				if idStr, ok := id.(string); ok && idStr != "" {
					requestLogger = requestLogger.With("request_id", idStr)
				}
			}

			// inject logger into context
			next.ServeHTTP(writer, request.WithContext(logger.WithContext(requestCtx, requestLogger)))
		})
	}
}
