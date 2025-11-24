package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/legacy"

	"github.com/pocj8ur4in/boilerplate-go/internal/app/boilerplate/ctx"
	genAPI "github.com/pocj8ur4in/boilerplate-go/internal/gen/api"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/jwt"
	"github.com/pocj8ur4in/boilerplate-go/internal/pkg/logger"
)

var (
	// ErrUnauthorized is returned when authentication fails.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrInvalidAuthorizationHeader is returned when the authorization header is invalid.
	ErrInvalidAuthorizationHeader = errors.New("invalid authorization header")

	// ErrMissingBearerToken is returned when the bearer token is missing.
	ErrMissingBearerToken = errors.New("missing bearer token")
)

// jwtAuthCache holds the cached OpenAPI spec and router.
type jwtAuthCache struct {
	// mu protects concurrent access.
	mu sync.RWMutex

	// spec holds the cached OpenAPI spec.
	spec *openapi3.T

	// router holds the cached router for spec validation.
	router routers.Router
}

// ValidationError represents a validation error with field information.
type ValidationError struct {
	// Field is the field that caused the validation error.
	Field string `json:"field"`

	// Message is the message of the validation error.
	Message string `json:"message"`
}

// ValidationErrorResponse represents the response for validation errors.
type ValidationErrorResponse struct {
	// Error is the error message.
	Error string `json:"error"`

	// Errors is the list of validation errors.
	Errors []ValidationError `json:"errors,omitempty"`
}

// JwtAuth is a middleware that validates JWT tokens based on OpenAPI spec security requirements.
func JwtAuth(log logger.Client, jwt jwt.Client) func(next http.Handler) http.Handler {
	return openAPIValidationWithSpec(log, jwt, nil)
}

// openAPIValidationWithSpec validates requests against the provided OpenAPI specification.
func openAPIValidationWithSpec(
	log logger.Client,
	jwt jwt.Client,
	spec *openapi3.T,
) func(next http.Handler) http.Handler {
	// create jwt auth cache for middleware
	jwtAuth := &jwtAuthCache{
		spec: spec,
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			// load spec if not cached
			if err := jwtAuth.ensureSpecLoaded(); err != nil {
				log.Error("failed to load OpenAPI spec", "error", err)
				sendValidationError(writer, "internal server error", nil)

				return
			}

			// validate request with authentication
			if err := jwtAuth.validateRequest(request, log, jwt); err != nil {
				log.Debug("request validation failed",
					"path", request.URL.Path,
					"method", request.Method,
					"error", err,
				)

				// parse validation errors
				validationErrors := parseValidationError(err)
				sendValidationError(writer, "validation failed", validationErrors)

				return
			}

			// validation passed, continue to next handler
			next.ServeHTTP(writer, request)
		})
	}
}

// ensureSpecLoaded loads the OpenAPI spec if not already cached.
func (j *jwtAuthCache) ensureSpecLoaded() error {
	if j.router != nil {
		return nil
	}

	var err error

	// use provided spec or load from generated code
	var spec *openapi3.T

	if j.spec != nil {
		spec = j.spec
	} else {
		spec, err = genAPI.GetSwagger()
		if err != nil {
			return fmt.Errorf("failed to load OpenAPI spec: %w", err)
		}

		j.spec = spec
	}

	// create router for matching requests
	router, err := legacy.NewRouter(spec)
	if err != nil {
		return fmt.Errorf("failed to create router from spec: %w", err)
	}

	j.router = router

	return nil
}

// validateRequest validates the HTTP request against the OpenAPI spec.
func (j *jwtAuthCache) validateRequest(request *http.Request, log logger.Client, jwt jwt.Client) error {
	j.mu.RLock()
	defer j.mu.RUnlock()

	// find route in spec
	route, pathParams, err := j.router.FindRoute(request)
	if err != nil {
		// route not found in spec, skip validation
		log.Debug("route not found in OpenAPI spec, skipping validation",
			"path", request.URL.Path,
			"method", request.Method,
		)

		return nil
	}

	// prepare validation input with JWT authentication
	requestValidationInput := &openapi3filter.RequestValidationInput{
		Request:    request,
		PathParams: pathParams,
		Route:      route,
		Options: &openapi3filter.Options{
			AuthenticationFunc: authenticationFunc(log, jwt),
			MultiError:         true,
		},
	}

	// read request body for validation
	if request.Body != nil {
		bodyBytes, readErr := io.ReadAll(request.Body)
		if readErr != nil {
			return fmt.Errorf("failed to read request body: %w", err)
		}

		// restore body for handler
		request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		// create new body for validation
		requestValidationInput.Request = request.Clone(request.Context())
		requestValidationInput.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	// validate request
	if err = openapi3filter.ValidateRequest(request.Context(), requestValidationInput); err != nil {
		return fmt.Errorf("request validation failed: %w", err)
	}

	return nil
}

// authenticationFunc creates an OpenAPI authentication function for JWT validation.
func authenticationFunc(log logger.Client, jwt jwt.Client) openapi3filter.AuthenticationFunc {
	return func(_ context.Context, input *openapi3filter.AuthenticationInput) error {
		// check if the security scheme is bearer
		if input.SecurityScheme.Scheme != "bearer" {
			log.Debug("unsupported security scheme", "scheme", input.SecurityScheme.Scheme)

			return ErrUnauthorized
		}

		// check bearer format if specified
		if input.SecurityScheme.BearerFormat != "" && input.SecurityScheme.BearerFormat != "JWT" {
			log.Debug("unsupported bearer format", "format", input.SecurityScheme.BearerFormat)

			return ErrUnauthorized
		}

		// extract authorization header
		authHeader := input.RequestValidationInput.Request.Header.Get("Authorization")
		if authHeader == "" {
			log.Debug("missing authorization header")

			return ErrMissingBearerToken
		}

		// check if token starts with "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			log.Debug("invalid authorization header format", "header", authHeader)

			return ErrInvalidAuthorizationHeader
		}

		// extract token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			log.Debug("empty bearer token")

			return ErrMissingBearerToken
		}

		// validate token
		claims, err := jwt.ValidateToken(tokenString)
		if err != nil {
			log.Debug("token validation failed", "error", err)

			return ErrUnauthorized
		}

		// add user information to request context
		request := input.RequestValidationInput.Request
		requestCtx := request.Context()
		requestCtx = context.WithValue(requestCtx, ctx.UserIDKey, claims.UserID)
		requestCtx = context.WithValue(requestCtx, ctx.UserEmailKey, claims.Email)
		requestCtx = context.WithValue(requestCtx, ctx.UserRoleKey, claims.Role)
		requestCtx = context.WithValue(requestCtx, ctx.JwtClaimsKey, claims)

		// update request with new context
		//
		//nolint:contextcheck // request.WithContext is the standard way to update request context
		input.RequestValidationInput.Request = request.WithContext(requestCtx)

		return nil
	}
}

// parseValidationError parses the validation error and extracts field-level errors.
func parseValidationError(err error) []ValidationError {
	if err == nil {
		return nil
	}

	var validationErrors []ValidationError

	// try to parse as RequestError
	var reqErr *openapi3filter.RequestError
	if errors.As(err, &reqErr) {
		field := "request"
		if reqErr.Parameter != nil {
			field = reqErr.Parameter.Name
		}

		validationErrors = append(validationErrors, ValidationError{
			Field:   field,
			Message: reqErr.Error(),
		})

		return validationErrors
	}

	// try to parse as SecurityRequirementsError
	var secErr *openapi3filter.SecurityRequirementsError
	if errors.As(err, &secErr) {
		for _, e := range secErr.Errors {
			validationErrors = append(validationErrors, ValidationError{
				Field:   "security",
				Message: e.Error(),
			})
		}

		return validationErrors
	}

	// parse schema validation errors
	var schemaErr *openapi3.SchemaError
	if errors.As(err, &schemaErr) {
		validationErrors = append(validationErrors, ValidationError{
			Field:   schemaErr.SchemaField,
			Message: schemaErr.Reason,
		})

		return validationErrors
	}

	// fallback: generic error
	validationErrors = append(validationErrors, ValidationError{
		Field:   "request",
		Message: err.Error(),
	})

	return validationErrors
}

// sendValidationError sends a validation error response.
func sendValidationError(writer http.ResponseWriter, message string, errors []ValidationError) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusBadRequest)

	response := ValidationErrorResponse{
		Error:  message,
		Errors: errors,
	}

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		// if encoding fails, send plain text error
		http.Error(writer, "internal server error", http.StatusInternalServerError)
	}
}
