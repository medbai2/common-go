package middleware

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/medbai2/common-go/config"
	"github.com/medbai2/common-go/logger"
	"github.com/medbai2/common-go/response"
	"github.com/medbai2/common-go/types"
)

var (
	jwksCache *jwk.Cache
	jwksURL   string
)

// init initializes the JWKS cache
func init() {
	cache := jwk.NewCache(context.Background())
	jwksCache = cache
}

// Auth0 validates Auth0 JWT tokens
// It extracts the Bearer token from the Authorization header,
// validates it against Auth0's JWKS, and stores user info in Gin context
func Auth0(cfg *config.Auth0Config, appLogger logger.Logger) gin.HandlerFunc {
	if !cfg.Enabled {
		// If Auth0 is disabled, return a no-op middleware
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// Build JWKS URL
	jwksURL = fmt.Sprintf("https://%s/.well-known/jwks.json", cfg.Domain)

	appLogger.Info("Auth0 middleware enabled", map[string]interface{}{
		"domain":   cfg.Domain,
		"audience": cfg.Audience,
	})

	return func(c *gin.Context) {
		requestLogger := logger.NewContextLogger(c.Request.Context(), "auth0-middleware")

		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			requestLogger.Warn("Missing Authorization header")
			response.Unauthorized(c, "Authorization header required")
			c.Abort()
			return
		}

		// Parse Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			requestLogger.Warn("Invalid Authorization header format")
			response.Unauthorized(c, "Bearer token required")
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Validate token
		user, err := validateToken(tokenString, cfg, requestLogger)
		if err != nil {
			requestLogger.Warn("Token validation failed", map[string]interface{}{
				"error": err.Error(),
			})
			response.Unauthorized(c, "Invalid or expired token")
			c.Abort()
			return
		}

		// Store user in context
		c.Set(string(types.Auth0UserKey), user)
		requestLogger.Info("Auth0 token validated successfully", map[string]interface{}{
			"user_id": user.Sub,
			"email":   user.Email,
		})
		c.Next()
	}
}

// validateToken validates the JWT token against Auth0's JWKS
func validateToken(tokenString string, cfg *config.Auth0Config, appLogger logger.Logger) (*types.Auth0User, error) {
	// Parse token without validation first to get the kid
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get key ID from token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid not found in token header")
		}

		// Fetch JWKS from cache (cache will fetch if not present)
		keySet, err := jwksCache.Get(context.Background(), jwksURL)
		if err != nil {
			return nil, fmt.Errorf("failed to get JWKS: %w", err)
		}

		// Find the key with matching kid
		key, found := keySet.LookupKeyID(kid)
		if !found {
			return nil, fmt.Errorf("key with kid %s not found", kid)
		}

		// Get public key
		var rawKey interface{}
		if err := key.Raw(&rawKey); err != nil {
			return nil, fmt.Errorf("failed to get raw key: %w", err)
		}

		return rawKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Validate claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Validate audience
	aud, ok := claims["aud"].(string)
	if !ok {
		// Try array format
		audArray, ok := claims["aud"].([]interface{})
		if ok && len(audArray) > 0 {
			aud = audArray[0].(string)
		} else {
			return nil, fmt.Errorf("audience not found in token")
		}
	}

	if aud != cfg.Audience {
		return nil, fmt.Errorf("audience mismatch: expected %s, got %s", cfg.Audience, aud)
	}

	// Validate issuer
	iss, ok := claims["iss"].(string)
	if !ok {
		return nil, fmt.Errorf("issuer not found in token")
	}

	expectedIss := fmt.Sprintf("https://%s/", cfg.Domain)
	if iss != expectedIss {
		return nil, fmt.Errorf("issuer mismatch: expected %s, got %s", expectedIss, iss)
	}

	// Extract user information
	sub, ok := claims["sub"].(string)
	if !ok {
		return nil, fmt.Errorf("sub (subject) not found in token")
	}

	email, _ := claims["email"].(string)
	name, _ := claims["name"].(string)
	if name == "" {
		name, _ = claims["nickname"].(string)
	}
	if name == "" {
		name = email
	}
	if name == "" {
		name = sub
	}

	return &types.Auth0User{
		Sub:   sub,
		Email: email,
		Name:  name,
	}, nil
}

// GetAuth0User extracts Auth0User from Gin context
// Returns nil if not found or not authenticated
func GetAuth0User(c *gin.Context) *types.Auth0User {
	user, exists := c.Get(string(types.Auth0UserKey))
	if !exists {
		return nil
	}

	auth0User, ok := user.(*types.Auth0User)
	if !ok {
		return nil
	}

	return auth0User
}

