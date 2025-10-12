package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testIssuer    = "test_issuer"
	testAudience  = "test_audience"
	testSecretKey = "test_secret_key"
)

// createTestJWT creates a JWT instance for testing.
func createTestJWT(t *testing.T) *JWT {
	t.Helper()

	config := &Config{
		Issuer:          &[]string{testIssuer}[0],
		Audience:        &[]string{testAudience}[0],
		SecretKey:       &[]string{testSecretKey}[0],
		AccessTokenTTL:  &[]time.Duration{1 * time.Hour}[0],
		RefreshTokenTTL: &[]time.Duration{24 * time.Hour}[0],
	}

	jwt, err := New(config)
	require.NoError(t, err)

	return jwt
}

func TestConfig_SetDefault(t *testing.T) {
	t.Parallel()

	t.Run("set all defaults", func(t *testing.T) {
		t.Parallel()

		config := &Config{}
		config.SetDefault()

		require.NotNil(t, config.Issuer)
		require.Equal(t, "boilerplate", *config.Issuer)
		require.NotNil(t, config.Audience)
		require.Equal(t, "boilerplate_audience", *config.Audience)
		require.NotNil(t, config.SecretKey)
		require.Equal(t, "boilerplate_secret_key", *config.SecretKey)
		require.NotNil(t, config.AccessTokenTTL)
		require.Equal(t, 1*time.Hour, *config.AccessTokenTTL)
		require.NotNil(t, config.RefreshTokenTTL)
		require.Equal(t, 24*time.Hour, *config.RefreshTokenTTL)
	})

	t.Run("preserve existing values", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			Issuer:          &[]string{testIssuer}[0],
			Audience:        &[]string{testAudience}[0],
			SecretKey:       &[]string{testSecretKey}[0],
			AccessTokenTTL:  &[]time.Duration{30 * time.Minute}[0],
			RefreshTokenTTL: &[]time.Duration{7 * 24 * time.Hour}[0],
		}

		config.SetDefault()

		require.Equal(t, testIssuer, *config.Issuer)
		require.Equal(t, testAudience, *config.Audience)
		require.Equal(t, testSecretKey, *config.SecretKey)
		require.Equal(t, 30*time.Minute, *config.AccessTokenTTL)
		require.Equal(t, 7*24*time.Hour, *config.RefreshTokenTTL)
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("create JWT with valid config", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			Issuer:          &[]string{testIssuer}[0],
			Audience:        &[]string{testAudience}[0],
			SecretKey:       &[]string{testSecretKey}[0],
			AccessTokenTTL:  &[]time.Duration{1 * time.Hour}[0],
			RefreshTokenTTL: &[]time.Duration{24 * time.Hour}[0],
		}

		jwt, err := New(config)
		require.NoError(t, err)
		require.NotNil(t, jwt)
		require.NotNil(t, jwt.config)
	})

	t.Run("create JWT with nil config", func(t *testing.T) {
		t.Parallel()

		jwt, err := New(nil)
		require.Error(t, err)
		require.Nil(t, jwt)
		require.ErrorIs(t, err, ErrConfigRequired)
	})

	t.Run("create JWT with empty config applies defaults", func(t *testing.T) {
		t.Parallel()

		config := &Config{}

		jwt, err := New(config)
		require.NoError(t, err)
		require.NotNil(t, jwt)
		require.Equal(t, "boilerplate", *jwt.config.Issuer)
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

		token, err := jwt.GenerateAccessToken("user123", "test@example.com", "admin")
		require.NoError(t, err)

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

		jwt, err := New(&Config{
			Issuer:          &[]string{testIssuer}[0],
			Audience:        &[]string{testAudience}[0],
			SecretKey:       &[]string{testSecretKey}[0],
			AccessTokenTTL:  &[]time.Duration{1 * time.Millisecond}[0],
			RefreshTokenTTL: &[]time.Duration{24 * time.Hour}[0],
		})
		require.NoError(t, err)

		token, err := jwt.GenerateAccessToken("user123", "test@example.com", "admin")
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond)

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
		token, err := jwt1.GenerateAccessToken("user123", "test@example.com", "admin")
		require.NoError(t, err)

		jwt2, err := New(&Config{
			Issuer:          &[]string{testIssuer}[0],
			Audience:        &[]string{testAudience}[0],
			SecretKey:       &[]string{"different_secret"}[0],
			AccessTokenTTL:  &[]time.Duration{1 * time.Hour}[0],
			RefreshTokenTTL: &[]time.Duration{24 * time.Hour}[0],
		})
		require.NoError(t, err)

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

		refreshToken, err := jwt.GenerateRefreshToken("user123", "test@example.com", "admin")
		require.NoError(t, err)

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

		token, err := jwt.GenerateAccessToken("user123", "test@example.com", "admin")
		require.NoError(t, err)

		claims, err := jwt.ExtractClaims(*token)
		require.NoError(t, err)
		require.NotNil(t, claims)
		require.Equal(t, "user123", claims.UserID)
		require.Equal(t, "test@example.com", claims.Email)
		require.Equal(t, "admin", claims.Role)
	})

	t.Run("extract claims from expired token", func(t *testing.T) {
		t.Parallel()

		jwt, err := New(&Config{
			Issuer:          &[]string{testIssuer}[0],
			Audience:        &[]string{testAudience}[0],
			SecretKey:       &[]string{testSecretKey}[0],
			AccessTokenTTL:  &[]time.Duration{1 * time.Millisecond}[0],
			RefreshTokenTTL: &[]time.Duration{24 * time.Hour}[0],
		})
		require.NoError(t, err)

		token, err := jwt.GenerateAccessToken("user123", "test@example.com", "admin")
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond)

		claims, err := jwt.ExtractClaims(*token)
		require.NoError(t, err)
		require.NotNil(t, claims)
		require.Equal(t, "user123", claims.UserID)
	})

	t.Run("reject malformed token", func(t *testing.T) {
		t.Parallel()

		jwt := createTestJWT(t)

		claims, err := jwt.ExtractClaims("not_a_valid_jwt_token")
		require.Error(t, err)
		require.Nil(t, claims)
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

func TestClaimsCustomFields(t *testing.T) {
	t.Parallel()

	t.Run("claims contain custom fields", func(t *testing.T) {
		t.Parallel()

		jwt := createTestJWT(t)

		userID := "user456"
		email := "custom@example.com"
		role := "moderator"

		token, err := jwt.GenerateAccessToken(userID, email, role)
		require.NoError(t, err)

		claims, err := jwt.ValidateToken(*token)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, role, claims.Role)
		assert.Equal(t, userID, claims.Subject)
		assert.Equal(t, testIssuer, claims.Issuer)
	})
}
