package jwt

import (
	"testing"
	"time"
)

const (
	// TestIssuer is the test issuer of JWT.
	TestIssuer = "test_issuer"

	// TestAudience is the test audience of JWT.
	TestAudience = "test_audience"

	// TestSecretKey is the test secret key of JWT.
	TestSecretKey = "test_secret_key"

	// TestAccessTokenTTL is the test access token TTL of JWT.
	TestAccessTokenTTL = 1 * time.Hour

	// TestRefreshTokenTTL is the test refresh token TTL of JWT.
	TestRefreshTokenTTL = 24 * time.Hour
)

// TestOption modifies config for testing.
type TestOption func(*Config)

// WithSecretKey sets custom secret key.
func WithSecretKey(secretKey string) TestOption {
	return func(c *Config) {
		c.SecretKey = secretKey
	}
}

// WithAccessTokenTTL sets custom access token TTL.
func WithAccessTokenTTL(accessTokenTTL time.Duration) TestOption {
	return func(c *Config) {
		c.AccessTokenTTL = accessTokenTTL
	}
}

// WithRefreshTokenTTL sets custom refresh token TTL.
func WithRefreshTokenTTL(refreshTokenTTL time.Duration) TestOption {
	return func(c *Config) {
		c.RefreshTokenTTL = refreshTokenTTL
	}
}

// InitForTest initializes for testing.
//
//revive:disable:unexported-return // returns unexported type for testing purposes
func InitForTest(t *testing.T, opts ...TestOption) *client {
	t.Helper()

	// set config
	config := Config{
		Issuer:          TestIssuer,
		Audience:        TestAudience,
		SecretKey:       TestSecretKey,
		AccessTokenTTL:  TestAccessTokenTTL,
		RefreshTokenTTL: TestRefreshTokenTTL,
	}

	// apply custom options
	for _, opt := range opts {
		opt(&config)
	}

	// create instance
	testClient := newInstance(config)

	return testClient
}
