package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// testIssuer is the issuer of the test JWT.
	testIssuer = "test_issuer"

	// testAudience is the audience of the test JWT.
	testAudience = "test_audience"

	// testSecretKey is the secret key of the test JWT.
	testSecretKey = "test_secret_key"

	// testAccessTokenTTL is the access token TTL of the test JWT.
	testAccessTokenTTL = 1 * time.Hour

	// testRefreshTokenTTL is the refresh token TTL of the test JWT.
	testRefreshTokenTTL = 24 * time.Hour
)

// createTestJWT creates a JWT instance for testing.
func createTestJWT(t *testing.T) *JWT {
	t.Helper()

	issuer := testIssuer
	audience := testAudience
	secretKey := testSecretKey
	accessTokenTTL := testAccessTokenTTL
	refreshTokenTTL := testRefreshTokenTTL

	config := &Config{
		Issuer:          &issuer,
		Audience:        &audience,
		SecretKey:       &secretKey,
		AccessTokenTTL:  &accessTokenTTL,
		RefreshTokenTTL: &refreshTokenTTL,
	}

	jwt, err := New(config)
	require.NoError(t, err)

	return jwt
}

func TestConfig(t *testing.T) {
	t.Parallel()

	t.Run("set default values on jwt config", func(t *testing.T) {
		t.Parallel()

		config := &Config{}
		config.SetDefault()

		require.NotNil(t, config.Issuer)
		require.Equal(t, defaultIssuer, *config.Issuer)
		require.NotNil(t, config.Audience)
		require.Equal(t, defaultAudience, *config.Audience)
		require.NotNil(t, config.SecretKey)
		require.Equal(t, defaultSecretKey, *config.SecretKey)
		require.NotNil(t, config.AccessTokenTTL)
		require.Equal(t, defaultAccessTokenTTL, *config.AccessTokenTTL)
		require.NotNil(t, config.RefreshTokenTTL)
		require.Equal(t, defaultRefreshTokenTTL, *config.RefreshTokenTTL)
	})

	t.Run("preserve existing values on jwt config", func(t *testing.T) {
		t.Parallel()

		issuer := testIssuer
		audience := testAudience
		secretKey := testSecretKey
		accessTokenTTL := testAccessTokenTTL
		refreshTokenTTL := testRefreshTokenTTL

		config := &Config{
			Issuer:          &issuer,
			Audience:        &audience,
			SecretKey:       &secretKey,
			AccessTokenTTL:  &accessTokenTTL,
			RefreshTokenTTL: &refreshTokenTTL,
		}

		config.SetDefault()

		require.Equal(t, testIssuer, *config.Issuer)
		require.Equal(t, testAudience, *config.Audience)
		require.Equal(t, testSecretKey, *config.SecretKey)
		require.Equal(t, testAccessTokenTTL, *config.AccessTokenTTL)
		require.Equal(t, testRefreshTokenTTL, *config.RefreshTokenTTL)
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("create JWT with valid config", func(t *testing.T) {
		t.Parallel()

		issuer := testIssuer
		audience := testAudience
		secretKey := testSecretKey
		accessTokenTTL := testAccessTokenTTL
		refreshTokenTTL := testRefreshTokenTTL

		config := &Config{
			Issuer:          &issuer,
			Audience:        &audience,
			SecretKey:       &secretKey,
			AccessTokenTTL:  &accessTokenTTL,
			RefreshTokenTTL: &refreshTokenTTL,
		}

		jwt, err := New(config)
		require.NoError(t, err)
		require.NotNil(t, jwt)
		require.NotNil(t, jwt.config)
		require.Equal(t, issuer, *jwt.config.Issuer)
		require.Equal(t, audience, *jwt.config.Audience)
		require.Equal(t, secretKey, *jwt.config.SecretKey)
		require.Equal(t, accessTokenTTL, *jwt.config.AccessTokenTTL)
		require.Equal(t, refreshTokenTTL, *jwt.config.RefreshTokenTTL)
	})

	t.Run("create JWT with nil config", func(t *testing.T) {
		t.Parallel()

		jwt, err := New(nil)
		require.NoError(t, err)
		require.NotNil(t, jwt)
		require.Equal(t, defaultIssuer, *jwt.config.Issuer)
		require.Equal(t, defaultAudience, *jwt.config.Audience)
		require.Equal(t, defaultSecretKey, *jwt.config.SecretKey)
		require.Equal(t, defaultAccessTokenTTL, *jwt.config.AccessTokenTTL)
		require.Equal(t, defaultRefreshTokenTTL, *jwt.config.RefreshTokenTTL)
	})
}

func TestGenerateAccessToken(t *testing.T) {
	t.Parallel()

	t.Run("generate valid access token", func(t *testing.T) {
		t.Parallel()

		jwt := createTestJWT(t)

		token, err := jwt.GenerateAccessToken("user123", "test@example.com", "admin")
		require.NoError(t, err)
		require.NotNil(t, token)
		require.NotEmpty(t, *token)
	})
}

func TestGenerateRefreshToken(t *testing.T) {
	t.Parallel()

	t.Run("generate valid refresh token", func(t *testing.T) {
		t.Parallel()

		jwt := createTestJWT(t)

		token, err := jwt.GenerateRefreshToken("user123", "test@example.com", "user")
		require.NoError(t, err)
		require.NotNil(t, token)
		require.NotEmpty(t, *token)
	})
}

func TestValidateToken(t *testing.T) {
	t.Parallel()

	t.Run("validate valid token", func(t *testing.T) {
		t.Parallel()

		jwt := createTestJWT(t)

		// generate access token
		token, err := jwt.GenerateAccessToken("user123", "test@example.com", "admin")
		require.NoError(t, err)

		// validate token
		claims, err := jwt.ValidateToken(*token)
		require.NoError(t, err)
		require.NotNil(t, claims)
		require.Equal(t, "user123", claims.UserID)
		require.Equal(t, "test@example.com", claims.Email)
		require.Equal(t, "admin", claims.Role)
	})

	t.Run("reject invalid token", func(t *testing.T) {
		t.Parallel()

		jwt := createTestJWT(t)

		claims, err := jwt.ValidateToken("invalid_token")
		require.Error(t, err)
		require.Nil(t, claims)
		require.ErrorIs(t, err, ErrInvalidToken)
	})
}

func TestValidateTokenExpired(t *testing.T) {
	t.Parallel()

	t.Run("reject expired token", func(t *testing.T) {
		t.Parallel()

		// create JWT with very short TTL for testing expiration
		issuer := testIssuer
		audience := testAudience
		secretKey := testSecretKey
		accessTokenTTL := 10 * time.Millisecond
		refreshTokenTTL := testRefreshTokenTTL

		jwt, err := New(&Config{
			Issuer:          &issuer,
			Audience:        &audience,
			SecretKey:       &secretKey,
			AccessTokenTTL:  &accessTokenTTL,
			RefreshTokenTTL: &refreshTokenTTL,
		})
		require.NoError(t, err)

		// generate access token
		token, err := jwt.GenerateAccessToken("user123", "test@example.com", "admin")
		require.NoError(t, err)

		// sleep for access token TTL
		time.Sleep(20 * time.Millisecond)

		// validate expired token
		claims, err := jwt.ValidateToken(*token)
		require.Error(t, err)
		require.Nil(t, claims)
		require.ErrorIs(t, err, ErrExpiredToken)
	})
}

func TestValidateTokenWrongSecret(t *testing.T) {
	t.Parallel()

	t.Run("reject token with wrong secret", func(t *testing.T) {
		t.Parallel()

		jwt1 := createTestJWT(t)

		// generate access token
		token, err := jwt1.GenerateAccessToken("user123", "test@example.com", "admin")
		require.NoError(t, err)

		// create JWT with different secret
		issuer := testIssuer
		audience := testAudience
		secretKey := "different_secret"
		accessTokenTTL := testAccessTokenTTL
		refreshTokenTTL := testRefreshTokenTTL

		jwt2, err := New(&Config{
			Issuer:          &issuer,
			Audience:        &audience,
			SecretKey:       &secretKey,
			AccessTokenTTL:  &accessTokenTTL,
			RefreshTokenTTL: &refreshTokenTTL,
		})
		require.NoError(t, err)

		// validate token with different secret
		claims, err := jwt2.ValidateToken(*token)
		require.Error(t, err)
		require.Nil(t, claims)
		require.ErrorIs(t, err, ErrInvalidToken)
	})
}

func TestRefreshAccessToken(t *testing.T) {
	t.Parallel()

	t.Run("refresh access token with valid refresh token", func(t *testing.T) {
		t.Parallel()

		jwt := createTestJWT(t)

		// generate refresh token
		refreshToken, err := jwt.GenerateRefreshToken("user123", "test@example.com", "admin")
		require.NoError(t, err)

		// generate new access token
		newAccessToken, err := jwt.RefreshAccessToken(*refreshToken)
		require.NoError(t, err)
		require.NotNil(t, newAccessToken)
		require.NotEmpty(t, *newAccessToken)

		// validate new access token
		claims, err := jwt.ValidateToken(*newAccessToken)
		require.NoError(t, err)
		require.Equal(t, "user123", claims.UserID)
		require.Equal(t, "test@example.com", claims.Email)
		require.Equal(t, "admin", claims.Role)
	})

	t.Run("reject invalid refresh token", func(t *testing.T) {
		t.Parallel()

		jwt := createTestJWT(t)

		// refresh invalid refresh token
		newAccessToken, err := jwt.RefreshAccessToken("invalid_refresh_token")
		require.Error(t, err)
		require.Nil(t, newAccessToken)
	})
}

func TestExtractClaims(t *testing.T) {
	t.Parallel()

	t.Run("extract claims from valid token", func(t *testing.T) {
		t.Parallel()

		jwt := createTestJWT(t)

		// generate access token
		token, err := jwt.GenerateAccessToken("user123", "test@example.com", "admin")
		require.NoError(t, err)

		// extract claims from token
		claims, err := jwt.ExtractClaims(*token)
		require.NoError(t, err)
		require.NotNil(t, claims)
		require.Equal(t, "user123", claims.UserID)
		require.Equal(t, "test@example.com", claims.Email)
		require.Equal(t, "admin", claims.Role)
	})

	t.Run("extract claims from expired token", func(t *testing.T) {
		t.Parallel()

		// create JWT with short TTL for testing expiration
		issuer := testIssuer
		audience := testAudience
		secretKey := testSecretKey
		accessTokenTTL := 10 * time.Millisecond
		refreshTokenTTL := testRefreshTokenTTL

		jwt, err := New(&Config{
			Issuer:          &issuer,
			Audience:        &audience,
			SecretKey:       &secretKey,
			AccessTokenTTL:  &accessTokenTTL,
			RefreshTokenTTL: &refreshTokenTTL,
		})
		require.NoError(t, err)

		// generate access token
		token, err := jwt.GenerateAccessToken("user123", "test@example.com", "admin")
		require.NoError(t, err)

		// sleep for access token TTL
		time.Sleep(20 * time.Millisecond)

		// extract claims from expired token
		claims, err := jwt.ExtractClaims(*token)
		require.NoError(t, err)
		require.NotNil(t, claims)
		require.Equal(t, "user123", claims.UserID)
	})

	t.Run("reject malformed token", func(t *testing.T) {
		t.Parallel()

		jwt := createTestJWT(t)

		// extract claims from malformed token
		claims, err := jwt.ExtractClaims("not_a_valid_jwt_token")
		require.Error(t, err)
		require.Nil(t, claims)
	})
}

func TestClaimsCustomFields(t *testing.T) {
	t.Parallel()

	t.Run("claims contain custom fields", func(t *testing.T) {
		t.Parallel()

		jwt := createTestJWT(t)

		// generate access token with custom fields
		userID := "user456"
		email := "custom@example.com"
		role := "moderator"

		token, err := jwt.GenerateAccessToken(userID, email, role)
		require.NoError(t, err)

		// validate token with custom fields
		claims, err := jwt.ValidateToken(*token)
		require.NoError(t, err)

		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, role, claims.Role)
		assert.Equal(t, userID, claims.Subject)
		assert.Equal(t, testIssuer, claims.Issuer)
	})
}

func TestNewModule(t *testing.T) {
	t.Parallel()

	t.Run("create JWT module", func(t *testing.T) {
		t.Parallel()

		module := NewModule()
		require.NotNil(t, module)
	})
}
