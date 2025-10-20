// Package jwt provides JWT token management.
package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/fx"
)

var (
	// ErrInvalidToken returned when the token is invalid.
	ErrInvalidToken = errors.New("invalid token")

	// ErrExpiredToken returned when the token is expired.
	ErrExpiredToken = errors.New("expired token")

	// ErrInvalidClaims returned when the claims are invalid.
	ErrInvalidClaims = errors.New("invalid claims")

	// ErrUnexpectedSigningMethod returned when the signing method is unexpected.
	ErrUnexpectedSigningMethod = errors.New("unexpected signing method")
)

// JWT provides JWT token management.
type JWT struct {
	// config provides JWT configuration.
	config *Config
}

// Config represents configuration for JWT.
type Config struct {
	// Issuer is issuer of JWT.
	Issuer *string `json:"issuer"`

	// Audience is audience of JWT.
	Audience *string `json:"audience"`

	// SecretKey is secret key of JWT.
	SecretKey *string `json:"secret_key"`

	// AccessTokenTTL is access token TTL of JWT.
	AccessTokenTTL *time.Duration `json:"access_token_ttl"`

	// RefreshTokenTTL is refresh token TTL of JWT.
	RefreshTokenTTL *time.Duration `json:"refresh_token_ttl"`
}

const (
	// defaultIssuer is default issuer of JWT.
	defaultIssuer = "boilerplate"

	// defaultAudience is default audience of JWT.
	defaultAudience = "boilerplate_audience"

	// defaultSecretKey is default secret key of JWT.
	defaultSecretKey = "boilerplate_secret_key"

	// defaultAccessTokenTTL is default access token TTL of JWT.
	defaultAccessTokenTTL = 1 * time.Hour

	// defaultRefreshTokenTTL is default refresh token TTL of JWT.
	defaultRefreshTokenTTL = 24 * time.Hour
)

// SetDefault sets default values.
func (c *Config) SetDefault() {
	if c.Issuer == nil {
		issuer := defaultIssuer
		c.Issuer = &issuer
	}

	if c.Audience == nil {
		audience := defaultAudience
		c.Audience = &audience
	}

	if c.SecretKey == nil {
		secretKey := defaultSecretKey
		c.SecretKey = &secretKey
	}

	if c.AccessTokenTTL == nil {
		accessTokenTTL := defaultAccessTokenTTL
		c.AccessTokenTTL = &accessTokenTTL
	}

	if c.RefreshTokenTTL == nil {
		refreshTokenTTL := defaultRefreshTokenTTL
		c.RefreshTokenTTL = &refreshTokenTTL
	}
}

// Claims represents JWT claims.
type Claims struct {
	// UserID is user ID of JWT.
	UserID string `json:"user_id"`

	// Email is email of JWT.
	Email string `json:"email"`

	// Role is role of JWT.
	Role string `json:"role"`

	// RegisteredClaims provides registered claims of JWT.
	jwt.RegisteredClaims
}

// NewModule provides module for JWT.
func NewModule() fx.Option {
	return fx.Module("jwt",
		fx.Provide(New),
	)
}

// New creates a new JWT instance.
func New(config *Config) (*JWT, error) {
	if config == nil {
		config = &Config{}
	}

	config.SetDefault()

	return &JWT{
		config: config,
	}, nil
}

// GenerateAccessToken generates an access token.
func (j *JWT) GenerateAccessToken(userID, email, role string) (*string, error) {
	return j.generateToken(userID, email, role, *j.config.AccessTokenTTL)
}

// GenerateRefreshToken generates a refresh token.
func (j *JWT) GenerateRefreshToken(userID, email, role string) (*string, error) {
	return j.generateToken(userID, email, role, *j.config.RefreshTokenTTL)
}

// generateToken generates a JWT token.
func (j *JWT) generateToken(userID, email, role string, ttl time.Duration) (*string, error) {
	now := time.Now()

	// set claims
	claims := &Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    *j.config.Issuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{*j.config.Audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	// create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// sign token
	signedTokenStr, err := token.SignedString([]byte(*j.config.SecretKey))
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}

	return &signedTokenStr, nil
}

// ValidateToken validates a JWT token and returns the claims.
func (j *JWT) ValidateToken(tokenStr string) (*Claims, error) {
	// parse token
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("%w: %v", ErrUnexpectedSigningMethod, token.Header["alg"])
			}

			return []byte(*j.config.SecretKey), nil
		},
	)
	if err != nil {
		// return error if token is expired
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, fmt.Errorf("%w: %w", ErrExpiredToken, err)
		}

		return nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}

	// check if token is valid
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}

	return claims, nil
}

// RefreshAccessToken refreshes an access token using a refresh token.
func (j *JWT) RefreshAccessToken(refreshToken string) (*string, error) {
	// validate refresh token
	claims, err := j.ValidateToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	return j.GenerateAccessToken(claims.UserID, claims.Email, claims.Role)
}

// ExtractClaims extracts claims from a token without validation.
func (j *JWT) ExtractClaims(tokenString string) (*Claims, error) {
	// parse token
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// check if claims are valid
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidClaims
	}

	return claims, nil
}
