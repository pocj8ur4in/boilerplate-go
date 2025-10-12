package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/pocj8ur4in/boilerplate-go/internal/gen/api"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/jwt"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
)

// ContextKey represents a context key.
type ContextKey string

const (
	// UserIDKey is the key for user ID in context.
	UserIDKey ContextKey = "user_id"

	// UserEmailKey is the key for user email in context.
	UserEmailKey ContextKey = "user_email"

	// UserRoleKey is the key for user role in context.
	UserRoleKey ContextKey = "user_role"

	// ClaimsKey is the key for JWT claims in context.
	ClaimsKey ContextKey = "claims"
)

// JWTAuth is a middleware that validates JWT tokens based on OpenAPI spec security requirements.
func JWTAuth(jwt *jwt.JWT, logger *logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			_, requiresAuth := request.Context().Value(api.BearerAuthScopes).([]string)

			// if endpoint doesn't require auth, skip
			if !requiresAuth {
				logger.Debug().Str("path", request.URL.Path).Msg("endpoint does not require authentication")
				next.ServeHTTP(writer, request)

				return
			}

			// extract token from Authorization header
			authHeader := request.Header.Get("Authorization")
			if authHeader == "" {
				logger.Debug().Msg("missing authorization header")
				http.Error(writer, "Unauthorized", http.StatusUnauthorized)

				return
			}

			// check if token starts with "Bearer "
			if !strings.HasPrefix(authHeader, "Bearer ") {
				logger.Debug().Str("auth_header", authHeader).Msg("invalid authorization header format")
				http.Error(writer, "Unauthorized", http.StatusUnauthorized)

				return
			}

			// extract token
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == "" {
				logger.Debug().Msg("empty token")
				http.Error(writer, "Unauthorized", http.StatusUnauthorized)

				return
			}

			// validate token
			claims, err := jwt.ValidateToken(tokenString)
			if err != nil {
				logger.Debug().Err(err).Msg("token validation failed")
				http.Error(writer, "Unauthorized", http.StatusUnauthorized)

				return
			}

			// add user information to context
			ctx := context.WithValue(request.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
			ctx = context.WithValue(ctx, UserRoleKey, claims.Role)
			ctx = context.WithValue(ctx, ClaimsKey, claims)

			// create new request with updated context
			request = request.WithContext(ctx)

			// continue to next handler
			next.ServeHTTP(writer, request)
		})
	}
}
