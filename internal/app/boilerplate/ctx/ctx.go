// Package ctx provides context key.
package ctx

// Key represents a context key.
type Key string

const (
	// RequestIDKey is the key for request ID in context.
	RequestIDKey Key = "request_id"

	// UserIDKey is the key for user ID in context.
	UserIDKey Key = "user_id"

	// UserEmailKey is the key for user email in context.
	UserEmailKey Key = "user_email"

	// UserRoleKey is the key for user role in context.
	UserRoleKey Key = "user_role"

	// JwtClaimsKey is the key for JWT claims in context.
	JwtClaimsKey Key = "jwt_claims"
)
