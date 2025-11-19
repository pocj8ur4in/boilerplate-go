// Package jwt provides JWT client.
package jwt

import (
	"errors"
	"fmt"
	"time"

	jwtPkg "github.com/golang-jwt/jwt/v5"
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

// Client defines the interface for JWT operations.
type Client interface {
	// GenerateAccessToken generates an access token.
	GenerateAccessToken(userID, email, role string) (*string, error)

	// GenerateRefreshToken generates a refresh token.
	GenerateRefreshToken(userID, email, role string) (*string, error)

	// ValidateToken validates a JWT token and returns the claims.
	ValidateToken(tokenStr string) (*Claims, error)

	// RefreshAccessToken refreshes an access token using a refresh token.
	RefreshAccessToken(refreshToken string) (*string, error)

	// ExtractClaims extracts claims from a token without validation.
	ExtractClaims(tokenString string) (*Claims, error)
}

// client implements jwt.Client interface.
type client struct {
	// Config provides JWT configuration.
	Config Config
}

// Config represents configuration for JWT.
type Config struct {
	// Issuer is issuer of JWT.
	Issuer string `env:"ISSUER" envDefault:"boilerplate" json:"issuer"`

	// Audience is audience of JWT.
	Audience string `env:"AUDIENCE" envDefault:"boilerplate_audience" json:"audience"`

	// SecretKey is secret key of JWT.
	SecretKey string `env:"SECRET_KEY" envDefault:"boilerplate_secret_key" json:"secret_key"`

	// AccessTokenTTL is access token TTL of JWT.
	AccessTokenTTL time.Duration `env:"ACCESS_TOKEN_TTL" envDefault:"15m" json:"access_token_ttl"`

	// RefreshTokenTTL is refresh token TTL of JWT.
	RefreshTokenTTL time.Duration `env:"REFRESH_TOKEN_TTL" envDefault:"168h" json:"refresh_token_ttl"`
}

// Claims represents JWT claims.
type Claims struct {
	// UserID is user ID of JWT.
	UserID string `json:"user_id"`

	// Email is email of JWT.
	Email string `json:"email"`

	// Role is role of JWT.
	Role string `json:"role"`

	// RegisteredClaims extends jwtPkg.RegisteredClaims.
	jwtPkg.RegisteredClaims
}

// NewModule provides module for JWT.
func NewModule() fx.Option {
	return fx.Module("jwt",
		// provide concrete type for constructor
		fx.Provide(func(config Config) *client {
			// create instance
			instance := newInstance(config)

			return instance
		}),
		// provide interface type for dependency injection
		fx.Provide(fx.Annotate(
			func(instance *client) Client {
				return instance
			},
		)),
	)
}

// newInstance creates a new JWT instance.
func newInstance(config Config) *client {
	return &client{Config: config}
}

// GenerateAccessToken generates an access token.
func (c *client) GenerateAccessToken(userID, email, role string) (*string, error) {
	return c.generateToken(userID, email, role, c.Config.AccessTokenTTL)
}

// GenerateRefreshToken generates a refresh token.
func (c *client) GenerateRefreshToken(userID, email, role string) (*string, error) {
	return c.generateToken(userID, email, role, c.Config.RefreshTokenTTL)
}

// generateToken generates a JWT token.
func (c *client) generateToken(userID, email, role string, ttl time.Duration) (*string, error) {
	now := time.Now()

	// set claims
	claims := &Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwtPkg.RegisteredClaims{
			Issuer:    c.Config.Issuer,
			Subject:   userID,
			Audience:  jwtPkg.ClaimStrings{c.Config.Audience},
			ExpiresAt: jwtPkg.NewNumericDate(now.Add(ttl)),
			NotBefore: jwtPkg.NewNumericDate(now),
			IssuedAt:  jwtPkg.NewNumericDate(now),
		},
	}

	// create token
	token := jwtPkg.NewWithClaims(jwtPkg.SigningMethodHS256, claims)

	// sign token
	signedTokenStr, err := token.SignedString([]byte(c.Config.SecretKey))
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}

	return &signedTokenStr, nil
}

// ValidateToken validates a JWT token and returns the claims.
func (c *client) ValidateToken(tokenStr string) (*Claims, error) {
	// parse token
	token, err := jwtPkg.ParseWithClaims(
		tokenStr,
		&Claims{},
		func(token *jwtPkg.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwtPkg.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("%w: %v", ErrUnexpectedSigningMethod, token.Header["alg"])
			}

			return []byte(c.Config.SecretKey), nil
		},
	)
	if err != nil {
		// return error if token is expired
		if errors.Is(err, jwtPkg.ErrTokenExpired) {
			return nil, ErrExpiredToken
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
func (c *client) RefreshAccessToken(refreshToken string) (*string, error) {
	// validate refresh token
	claims, err := c.ValidateToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	return c.GenerateAccessToken(claims.UserID, claims.Email, claims.Role)
}

// ExtractClaims extracts claims from a token without validation.
func (c *client) ExtractClaims(tokenStr string) (*Claims, error) {
	// parse token
	token, _, err := jwtPkg.NewParser().ParseUnverified(tokenStr, &Claims{})
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
