package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocj8ur4in/boilerplate-go/internal/gen/api"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/jwt"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
)

// setupTestJWT creates a test JWT.
func setupTestJWT(t *testing.T) *jwt.JWT {
	t.Helper()

	secretKey := "test-secret-key"
	jwtConfig := &jwt.Config{
		SecretKey: &secretKey,
	}

	jwtService, err := jwt.New(jwtConfig)
	require.NoError(t, err)

	return jwtService
}

// generateTestToken generates a test JWT token.
//
//nolint:unparam // userID is kept as parameter for clarity
func generateTestToken(t *testing.T, jwtService *jwt.JWT, userID, email, role string) string {
	t.Helper()

	token, err := jwtService.GenerateAccessToken(userID, email, role)
	require.NoError(t, err)
	require.NotNil(t, token)

	return *token
}

//nolint:funlen // Multiple test cases in one function
func TestJWTAuth(t *testing.T) {
	t.Parallel()

	t.Run("allow request without authentication when endpoint doesn't require auth", func(t *testing.T) {
		t.Parallel()

		jwtService := setupTestJWT(t)
		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		handler := JWTAuth(jwtService, log)(testHandler(http.StatusOK, "success"))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "success", recorder.Body.String())
	})

	t.Run("require authentication when endpoint requires auth", func(t *testing.T) {
		t.Parallel()

		jwtService := setupTestJWT(t)
		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		handler := JWTAuth(jwtService, log)(testHandler(http.StatusOK, "success"))

		// create request with BearerAuth context (simulating protected endpoint)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		//nolint:staticcheck // Using api.BearerAuthScopes as context key
		ctx := context.WithValue(req.Context(), api.BearerAuthScopes, []string{})
		req = req.WithContext(ctx)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	})

	t.Run("reject request with missing authorization header", func(t *testing.T) {
		t.Parallel()

		jwtService := setupTestJWT(t)
		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		handler := JWTAuth(jwtService, log)(testHandler(http.StatusOK, "success"))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		//nolint:staticcheck // Using api.BearerAuthScopes as context key
		ctx := context.WithValue(req.Context(), api.BearerAuthScopes, []string{})
		req = req.WithContext(ctx)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	})

	t.Run("reject request with invalid authorization header format", func(t *testing.T) {
		t.Parallel()

		jwtService := setupTestJWT(t)
		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		handler := JWTAuth(jwtService, log)(testHandler(http.StatusOK, "success"))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "InvalidFormat token")
		//nolint:staticcheck // Using api.BearerAuthScopes as context key
		ctx := context.WithValue(req.Context(), api.BearerAuthScopes, []string{})
		req = req.WithContext(ctx)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	})

	t.Run("reject request with empty token", func(t *testing.T) {
		t.Parallel()

		jwtService := setupTestJWT(t)
		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		handler := JWTAuth(jwtService, log)(testHandler(http.StatusOK, "success"))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer ")
		//nolint:staticcheck // Using api.BearerAuthScopes as context key
		ctx := context.WithValue(req.Context(), api.BearerAuthScopes, []string{})
		req = req.WithContext(ctx)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	})

	t.Run("reject request with invalid token", func(t *testing.T) {
		t.Parallel()

		jwtService := setupTestJWT(t)
		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		handler := JWTAuth(jwtService, log)(testHandler(http.StatusOK, "success"))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		//nolint:staticcheck // Using api.BearerAuthScopes as context key
		ctx := context.WithValue(req.Context(), api.BearerAuthScopes, []string{})
		req = req.WithContext(ctx)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	})

	t.Run("allow request with valid token", func(t *testing.T) {
		t.Parallel()

		jwtService := setupTestJWT(t)
		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		token := generateTestToken(t, jwtService, "user123", "test@example.com", "user")

		handler := JWTAuth(jwtService, log)(testHandler(http.StatusOK, "success"))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		//nolint:staticcheck // Using api.BearerAuthScopes as context key
		ctx := context.WithValue(req.Context(), api.BearerAuthScopes, []string{})
		req = req.WithContext(ctx)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "success", recorder.Body.String())
	})

	t.Run("add user information to context with valid token", func(t *testing.T) {
		t.Parallel()

		jwtService := setupTestJWT(t)
		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		userID := "user123"
		email := "test@example.com"
		role := "admin"
		token := generateTestToken(t, jwtService, userID, email, role)

		var capturedUserID, capturedEmail, capturedRole string

		handler := JWTAuth(jwtService, log)(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if id := request.Context().Value(UserIDKey); id != nil {
				capturedUserID, _ = id.(string)
			}

			if em := request.Context().Value(UserEmailKey); em != nil {
				capturedEmail, _ = em.(string)
			}

			if r := request.Context().Value(UserRoleKey); r != nil {
				capturedRole, _ = r.(string)
			}

			writer.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		//nolint:staticcheck // Using api.BearerAuthScopes as context key
		ctx := context.WithValue(req.Context(), api.BearerAuthScopes, []string{})
		req = req.WithContext(ctx)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, userID, capturedUserID)
		assert.Equal(t, email, capturedEmail)
		assert.Equal(t, role, capturedRole)
	})

	t.Run("add claims to context with valid token", func(t *testing.T) {
		t.Parallel()

		jwtService := setupTestJWT(t)
		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		token := generateTestToken(t, jwtService, "user123", "test@example.com", "user")

		var capturedClaims *jwt.Claims

		handler := JWTAuth(jwtService, log)(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if claims := request.Context().Value(ClaimsKey); claims != nil {
				capturedClaims, _ = claims.(*jwt.Claims)
			}

			writer.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		//nolint:staticcheck // Using api.BearerAuthScopes as context key
		ctx := context.WithValue(req.Context(), api.BearerAuthScopes, []string{})
		req = req.WithContext(ctx)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		require.NotNil(t, capturedClaims)
		assert.Equal(t, "user123", capturedClaims.UserID)
		assert.Equal(t, "test@example.com", capturedClaims.Email)
		assert.Equal(t, "user", capturedClaims.Role)
	})
}

func TestJWTAuthContextKeys(t *testing.T) {
	t.Parallel()

	t.Run("context keys are defined correctly", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, UserIDKey, ContextKey("user_id"))
		assert.Equal(t, UserEmailKey, ContextKey("user_email"))
		assert.Equal(t, UserRoleKey, ContextKey("user_role"))
		assert.Equal(t, ClaimsKey, ContextKey("claims"))
	})
}

func TestJWTAuthWithDifferentRoles(t *testing.T) {
	t.Parallel()

	roles := []string{"admin", "user", "guest", "moderator"}

	for _, role := range roles {
		t.Run("authenticate with role "+role, func(t *testing.T) {
			t.Parallel()

			jwtService := setupTestJWT(t)
			log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
			require.NoError(t, err)

			token := generateTestToken(t, jwtService, "user123", "test@example.com", role)

			var capturedRole string

			handler := JWTAuth(jwtService, log)(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if r := request.Context().Value(UserRoleKey); r != nil {
					capturedRole, _ = r.(string)
				}

				writer.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			//nolint:staticcheck // Using api.BearerAuthScopes as context key
			ctx := context.WithValue(req.Context(), api.BearerAuthScopes, []string{})
			req = req.WithContext(ctx)

			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, role, capturedRole)
		})
	}
}

func TestJWTAuthTokenFromDifferentSecret(t *testing.T) {
	t.Parallel()

	t.Run("reject token signed with different secret", func(t *testing.T) {
		t.Parallel()

		// create JWT service with one secret
		secretKey1 := "secret-key-1"
		jwtConfig1 := &jwt.Config{
			SecretKey: &secretKey1,
		}
		jwtService1, err := jwt.New(jwtConfig1)
		require.NoError(t, err)

		// create JWT service with different secret
		secretKey2 := "secret-key-2"
		jwtConfig2 := &jwt.Config{
			SecretKey: &secretKey2,
		}
		jwtService2, err := jwt.New(jwtConfig2)
		require.NoError(t, err)

		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		// generate token with first secret
		token := generateTestToken(t, jwtService1, "user123", "test@example.com", "user")

		// try to validate with second secret
		handler := JWTAuth(jwtService2, log)(testHandler(http.StatusOK, "success"))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		//nolint:staticcheck // Using api.BearerAuthScopes as context key
		ctx := context.WithValue(req.Context(), api.BearerAuthScopes, []string{})
		req = req.WithContext(ctx)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	})
}

func TestJWTAuthMiddlewareChaining(t *testing.T) {
	t.Parallel()

	t.Run("chain JWT auth with other middlewares", func(t *testing.T) {
		t.Parallel()

		jwtService := setupTestJWT(t)
		log, err := logger.New(&logger.Config{Level: &[]string{"info"}[0]})
		require.NoError(t, err)

		token := generateTestToken(t, jwtService, "user123", "test@example.com", "user")

		handler := RequestID(
			SecurityHeaders()(
				JWTAuth(jwtService, log)(
					testHandler(http.StatusOK, "success"),
				),
			),
		)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		//nolint:staticcheck // Using api.BearerAuthScopes as context key
		ctx := context.WithValue(req.Context(), api.BearerAuthScopes, []string{})
		req = req.WithContext(ctx)

		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "success", recorder.Body.String())
		assert.NotEmpty(t, recorder.Header().Get("X-Content-Type-Options"))
	})
}
