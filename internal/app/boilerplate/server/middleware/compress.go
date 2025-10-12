package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// Compress is a middleware that compresses the response.
func Compress(level int, format string) func(next http.Handler) http.Handler {
	return middleware.Compress(level, format)
}
