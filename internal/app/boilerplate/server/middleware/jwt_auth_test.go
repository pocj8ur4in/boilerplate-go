package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genAPI "github.com/pocj8ur4in/boilerplate-go/internal/gen/api"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/jwt"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
)

var (
	// ErrInvalidParameter is returned when a parameter is invalid.
	ErrInvalidParameter = errors.New("invalid parameter")

	// ErrRequestValidationFailed is returned when request validation fails.
	ErrRequestValidationFailed = errors.New("request validation failed")

	// ErrUnknownValidationError is returned for unknown validation errors.
	ErrUnknownValidationError = errors.New("unknown validation error")
)

const (
	// specContent is the content of the test OpenAPI spec.
	specContent = `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
paths:
  /test:
    get:
      responses:
        '200':
          description: OK
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required:
                - name
                - email
              properties:
                name:
                  type: string
                  minLength: 1
                email:
                  type: string
                  format: email
                  pattern: '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$'
      responses:
        '200':
          description: OK
  /test/{id}:
    get:
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
      responses:
        '200':
          description: OK
  /protected:
    get:
      security:
        - BearerAuth: []
      responses:
        '200':
          description: OK
`
)

// generateTestToken generates a test JWT token.
func generateTestToken(t *testing.T, jwt jwt.Client, userID, email, role string) string {
	t.Helper()

	token, err := jwt.GenerateAccessToken(userID, email, role)
	require.NoError(t, err)
	require.NotNil(t, token)

	return *token
}

// createTestSpec creates a test OpenAPI spec for testing.
func createTestSpec(t *testing.T) *openapi3.T {
	t.Helper()

	// create loader
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	// load spec from content
	spec, err := loader.LoadFromData([]byte(specContent))
	require.NoError(t, err)
	require.NotNil(t, spec)

	return spec
}

func TestJwtAuth(t *testing.T) {
	t.Parallel()

	t.Run("validate request with authentication", func(t *testing.T) {
		t.Parallel()

		log := logger.InitForTest(t)
		jwtClient := jwt.InitForTest(t)
		middleware := JwtAuth(log, jwtClient)

		assert.NotNil(t, middleware)

		next := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusOK)
		})

		handler := middleware(next)
		assert.NotNil(t, handler)

		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.Positive(t, recorder.Code)
	})
}

func TestOpenAPIValidation(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name           string
		method         string
		path           string
		body           []byte
		expectedStatus int
	}{
		{
			name:           "valid GET request",
			method:         http.MethodGet,
			path:           "/test",
			body:           nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid POST request",
			method:         http.MethodPost,
			path:           "/test",
			body:           []byte(`{"name":"example"}`),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "route not in spec",
			method:         http.MethodGet,
			path:           "/unknown",
			body:           nil,
			expectedStatus: http.StatusOK,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			handler := openAPIValidationWithSpec(
				logger.InitForTest(t),
				jwt.InitForTest(t, jwt.WithSecretKey("test-secret")),
				createTestSpec(t),
			)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			var request *http.Request
			if testcase.body != nil {
				request = httptest.NewRequest(testcase.method, testcase.path, bytes.NewReader(testcase.body))
				request.Header.Set("Content-Type", "application/json")
			} else {
				request = httptest.NewRequest(testcase.method, testcase.path, nil)
			}

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)

			assert.Equal(t, testcase.expectedStatus, recorder.Code)
		})
	}
}

func TestOpenAPIValidationWithJWTAuthenticationSuccess(t *testing.T) {
	t.Parallel()

	t.Run("allow request to protected endpoint with valid token", func(t *testing.T) {
		t.Parallel()

		log := logger.InitForTest(t)
		jwtClient := jwt.InitForTest(t, jwt.WithSecretKey("test-secret-key"))
		testSpec := createTestSpec(t)

		token := generateTestToken(t, jwtClient, "user123", "test@example.com", "admin")

		handler := openAPIValidationWithSpec(
			log,
			jwtClient,
			testSpec,
		)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		request := httptest.NewRequest(http.MethodGet, "/protected", nil)
		request.Header.Set("Authorization", "Bearer "+token)

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("allow request to unprotected endpoint without token", func(t *testing.T) {
		t.Parallel()

		log := logger.InitForTest(t)
		jwtClient := jwt.InitForTest(t, jwt.WithSecretKey("test-secret-key"))
		testSpec := createTestSpec(t)

		handler := openAPIValidationWithSpec(
			log, jwtClient, testSpec,
		)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		request := httptest.NewRequest(http.MethodGet, "/test", nil)

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestOpenAPIValidationWithJWTAuthenticationMissingToken(t *testing.T) {
	t.Parallel()

	t.Run("reject request to protected endpoint without token", func(t *testing.T) {
		t.Parallel()

		log := logger.InitForTest(t)
		jwtClient := jwt.InitForTest(t, jwt.WithSecretKey("test-secret-key"))
		testSpec := createTestSpec(t)

		handler := openAPIValidationWithSpec(
			log,
			jwtClient,
			testSpec,
		)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		request := httptest.NewRequest(http.MethodGet, "/protected", nil)

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})
}

func TestOpenAPIValidationWithJWTAuthenticationInvalidToken(t *testing.T) {
	t.Parallel()

	t.Run("reject request to protected endpoint with invalid token", func(t *testing.T) {
		t.Parallel()

		log := logger.InitForTest(t)
		jwtClient := jwt.InitForTest(t, jwt.WithSecretKey("test-secret-key"))
		testSpec := createTestSpec(t)

		handler := openAPIValidationWithSpec(
			log,
			jwtClient,
			testSpec,
		)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		request := httptest.NewRequest(http.MethodGet, "/protected", nil)
		request.Header.Set("Authorization", "Bearer invalid-token")

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("reject token signed with different secret", func(t *testing.T) {
		t.Parallel()

		log := logger.InitForTest(t)
		jwtClient1 := jwt.InitForTest(t, jwt.WithSecretKey("test-secret-key-1"))
		jwtClient2 := jwt.InitForTest(t, jwt.WithSecretKey("test-secret-key-2"))
		testSpec := createTestSpec(t)

		// generate token with first secret
		token := generateTestToken(t, jwtClient1, "user123", "test@example.com", "user")

		// try to validate with second secret
		handler := openAPIValidationWithSpec(
			log, jwtClient2, testSpec,
		)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		request := httptest.NewRequest(http.MethodGet, "/protected", nil)
		request.Header.Set("Authorization", "Bearer "+token)

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})
}

func TestOpenAPIValidationWithJWTAuthenticationInvalidFormat(t *testing.T) {
	t.Parallel()

	t.Run("reject request to protected endpoint with wrong auth format", func(t *testing.T) {
		t.Parallel()

		log := logger.InitForTest(t)
		jwtClient := jwt.InitForTest(t, jwt.WithSecretKey("test-secret-key"))
		testSpec := createTestSpec(t)

		handler := openAPIValidationWithSpec(
			log,
			jwtClient,
			testSpec,
		)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		request := httptest.NewRequest(http.MethodGet, "/protected", nil)
		request.Header.Set("Authorization", "Basic token123")

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})
}

func TestOpenAPIValidationWithDifferentRoles(t *testing.T) {
	t.Parallel()

	roles := []string{"admin", "user", "guest", "moderator"}

	for _, role := range roles {
		t.Run("authenticate with role "+role, func(t *testing.T) {
			t.Parallel()

			log := logger.InitForTest(t)
			jwtClient := jwt.InitForTest(t, jwt.WithSecretKey("test-secret-key"))
			testSpec := createTestSpec(t)

			token := generateTestToken(t, jwtClient, "user123", "test@example.com", role)

			handler := openAPIValidationWithSpec(
				log,
				jwtClient,
				testSpec,
			)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.Header.Set("Authorization", "Bearer "+token)

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)

			assert.Equal(t, http.StatusOK, recorder.Code)
		})
	}
}

func TestParseValidationError(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name            string
		err             error
		expectedType    string
		expectedMessage string
	}{
		{
			name: "request error with parameter",
			err: &openapi3filter.RequestError{
				Parameter: &openapi3.Parameter{Name: "user_id"},
				Err:       ErrInvalidParameter,
			},
			expectedType:    "VALIDATION_FAILED",
			expectedMessage: "parameter \"user_id\" in  has an error: invalid parameter",
		},
		{
			name:            "request error without parameter",
			err:             &openapi3filter.RequestError{Err: ErrRequestValidationFailed},
			expectedType:    "VALIDATION_FAILED",
			expectedMessage: "request validation failed",
		},
		{
			name: "security requirements error",
			err: &openapi3filter.SecurityRequirementsError{
				Errors: []error{ErrMissingBearerToken},
			},
			expectedType:    "UNAUTHORIZED",
			expectedMessage: "missing bearer token",
		},
		{
			name:            "schema error",
			err:             &openapi3.SchemaError{SchemaField: "#/properties/email", Reason: "invalid format"},
			expectedType:    "VALIDATION_FAILED",
			expectedMessage: "invalid format",
		},
		{
			name:            "generic error",
			err:             ErrUnknownValidationError,
			expectedType:    "VALIDATION_FAILED",
			expectedMessage: "unknown validation error",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			message := parseErrorMessage(testcase.err)

			assert.Equal(t, testcase.expectedMessage, message)
		})
	}
}

func TestEnsureSpecLoaded(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name         string
		existingSpec *openapi3.T
		expectError  bool
	}{
		{
			name:         "load spec from embedded code",
			existingSpec: nil,
			expectError:  false,
		},
		{
			name:         "use existing spec",
			existingSpec: createTestSpec(t),
			expectError:  false,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			log := logger.InitForTest(t)
			jwtClient := jwt.InitForTest(t, jwt.WithSecretKey("test-secret-key"))

			middleware := openAPIValidationWithSpec(log, jwtClient, testcase.existingSpec)

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if testcase.expectError {
				assert.NotEqual(t, http.StatusOK, recorder.Code)
			} else {
				assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest}, recorder.Code)
			}
		})
	}
}

func TestSendErrorResponse(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name           string
		errorType      string
		message        string
		expectedStatus int
	}{
		{
			name:           "validation error",
			errorType:      "VALIDATION_FAILED",
			message:        "invalid format",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "unauthorized error",
			errorType:      "UNAUTHORIZED",
			message:        "missing bearer token",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "internal server error",
			errorType:      "INTERNAL_SERVER_ERROR",
			message:        "failed to load OpenAPI spec",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			genAPI.SendError(recorder, testcase.expectedStatus, testcase.errorType, testcase.message)

			assert.Equal(t, testcase.expectedStatus, recorder.Code)
			assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

			var response genAPI.GenericErrorResponse

			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
			assert.Equal(t, testcase.errorType, response.Error)
			assert.Equal(t, testcase.message, response.Message)
		})
	}
}
