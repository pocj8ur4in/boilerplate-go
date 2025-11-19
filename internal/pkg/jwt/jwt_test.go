package jwt

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestInit(t *testing.T) {
	t.Parallel()

	t.Run("init for test instance of JWT", func(t *testing.T) {
		t.Parallel()

		jwt := InitForTest(t)
		require.NotNil(t, jwt)
		require.Equal(t, TestIssuer, jwt.Config.Issuer)
		require.Equal(t, TestAudience, jwt.Config.Audience)
		require.Equal(t, TestSecretKey, jwt.Config.SecretKey)
		require.Equal(t, TestAccessTokenTTL, jwt.Config.AccessTokenTTL)
		require.Equal(t, TestRefreshTokenTTL, jwt.Config.RefreshTokenTTL)
	})
}

func TestNewAndSetup(t *testing.T) {
	t.Parallel()

	t.Run("new and setup JWT", func(t *testing.T) {
		t.Parallel()

		config := Config{
			Issuer:          TestIssuer,
			Audience:        TestAudience,
			SecretKey:       TestSecretKey,
			AccessTokenTTL:  TestAccessTokenTTL,
			RefreshTokenTTL: TestRefreshTokenTTL,
		}

		jwt := newInstance(config)
		require.NotNil(t, jwt)
		require.Equal(t, TestIssuer, jwt.Config.Issuer)
		require.Equal(t, TestAudience, jwt.Config.Audience)
		require.Equal(t, TestSecretKey, jwt.Config.SecretKey)
		require.Equal(t, TestAccessTokenTTL, jwt.Config.AccessTokenTTL)
		require.Equal(t, TestRefreshTokenTTL, jwt.Config.RefreshTokenTTL)
	})
}

func TestNewModule(t *testing.T) {
	t.Parallel()

	t.Run("create module for JWT", func(t *testing.T) {
		t.Parallel()

		module := NewModule()
		require.NotNil(t, module)
	})

	t.Run("create app with JWT module", func(t *testing.T) {
		t.Parallel()

		jwt := InitForTest(t)
		require.NotNil(t, jwt)

		var forPopulate Client

		app := fx.New(
			fx.Supply(jwt.Config),
			NewModule(),
			fx.Populate(&forPopulate),
		)

		ctx := context.Background()
		err := app.Start(ctx)
		require.NoError(t, err)

		require.NotNil(t, forPopulate)

		err = app.Stop(ctx)
		require.NoError(t, err)
	})
}

func TestGenerateAccessToken(t *testing.T) {
	t.Parallel()

	t.Run("generate valid access token", func(t *testing.T) {
		t.Parallel()

		jwt := InitForTest(t)

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

		jwt := InitForTest(t)

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

		jwt := InitForTest(t)

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

		jwt := InitForTest(t)

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
		jwt := InitForTest(t, WithAccessTokenTTL(10*time.Millisecond), WithRefreshTokenTTL(10*time.Millisecond))

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

		jwt1 := InitForTest(t)

		// generate access token
		token, err := jwt1.GenerateAccessToken("user123", "test@example.com", "admin")
		require.NoError(t, err)

		// create JWT with different secret
		jwt2 := InitForTest(t, WithSecretKey("different_secret"))

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

		jwt := InitForTest(t)

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

		jwt := InitForTest(t)

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

		jwt := InitForTest(t)

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
		jwt := InitForTest(t, WithAccessTokenTTL(10*time.Millisecond))

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

		jwt := InitForTest(t)

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

		jwt := InitForTest(t)

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
		assert.Equal(t, TestIssuer, claims.Issuer)
	})
}
