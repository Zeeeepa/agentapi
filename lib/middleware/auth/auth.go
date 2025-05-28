package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coder/agentapi/lib/middleware"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware handles authentication for AgentAPI
type AuthMiddleware struct {
	config   middleware.AuthConfig
	enabled  bool
	jwtKey   []byte
	apiKeys  map[string]string // API key -> user ID mapping
}

// NewAuthMiddleware creates a new authentication middleware instance
func NewAuthMiddleware(config middleware.AuthConfig) *AuthMiddleware {
	return &AuthMiddleware{
		config:  config,
		enabled: config.Enabled,
		jwtKey:  []byte(config.Secret),
		apiKeys: make(map[string]string),
	}
}

// Name returns the middleware name
func (a *AuthMiddleware) Name() string {
	return "auth"
}

// IsEnabled returns whether the middleware is enabled
func (a *AuthMiddleware) IsEnabled() bool {
	return a.enabled
}

// Configure configures the middleware with the provided config
func (a *AuthMiddleware) Configure(config interface{}) error {
	if authConfig, ok := config.(middleware.AuthConfig); ok {
		a.config = authConfig
		a.enabled = authConfig.Enabled
		a.jwtKey = []byte(authConfig.Secret)
		return nil
	}
	return fmt.Errorf("invalid auth config type")
}

// Handler returns the HTTP middleware handler
func (a *AuthMiddleware) Handler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !a.enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Extract authentication token
			token, err := a.extractToken(r)
			if err != nil {
				a.writeUnauthorized(w, err.Error())
				return
			}

			// Validate token based on auth type
			userID, err := a.validateToken(token)
			if err != nil {
				a.writeUnauthorized(w, err.Error())
				return
			}

			// Add user context to request
			ctx := context.WithValue(r.Context(), middleware.UserIDKey, userID)
			
			// Get or create request context
			reqCtx := middleware.GetRequestContext(ctx)
			if reqCtx == nil {
				reqCtx = &middleware.RequestContext{
					RequestID: a.generateRequestID(),
					Timestamp: time.Now(),
					Metadata:  make(map[string]interface{}),
				}
			}
			reqCtx.UserID = userID
			
			ctx = middleware.SetRequestContext(ctx, reqCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractToken extracts the authentication token from the request
func (a *AuthMiddleware) extractToken(r *http.Request) (string, error) {
	switch a.config.Type {
	case "bearer":
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			return "", fmt.Errorf("missing authorization header")
		}
		
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return "", fmt.Errorf("invalid authorization header format")
		}
		
		return parts[1], nil
		
	case "api_key":
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			apiKey = r.URL.Query().Get("api_key")
		}
		if apiKey == "" {
			return "", fmt.Errorf("missing API key")
		}
		return apiKey, nil
		
	default:
		return "", fmt.Errorf("unsupported auth type: %s", a.config.Type)
	}
}

// validateToken validates the extracted token
func (a *AuthMiddleware) validateToken(token string) (string, error) {
	switch a.config.Type {
	case "bearer":
		return a.validateJWT(token)
	case "api_key":
		return a.validateAPIKey(token)
	default:
		return "", fmt.Errorf("unsupported auth type: %s", a.config.Type)
	}
}

// validateJWT validates a JWT token
func (a *AuthMiddleware) validateJWT(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.jwtKey, nil
	})
	
	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}
	
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if userID, exists := claims["user_id"]; exists {
			if userIDStr, ok := userID.(string); ok {
				return userIDStr, nil
			}
		}
		return "", fmt.Errorf("invalid token claims")
	}
	
	return "", fmt.Errorf("invalid token")
}

// validateAPIKey validates an API key
func (a *AuthMiddleware) validateAPIKey(apiKey string) (string, error) {
	if userID, exists := a.apiKeys[apiKey]; exists {
		return userID, nil
	}
	return "", fmt.Errorf("invalid API key")
}

// GenerateJWT generates a JWT token for a user
func (a *AuthMiddleware) GenerateJWT(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"iss":     a.config.Issuer,
		"exp":     time.Now().Add(a.config.Expiration).Unix(),
		"iat":     time.Now().Unix(),
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtKey)
}

// AddAPIKey adds an API key for a user
func (a *AuthMiddleware) AddAPIKey(userID string) (string, error) {
	apiKey, err := a.generateAPIKey()
	if err != nil {
		return "", err
	}
	
	a.apiKeys[apiKey] = userID
	return apiKey, nil
}

// RemoveAPIKey removes an API key
func (a *AuthMiddleware) RemoveAPIKey(apiKey string) {
	delete(a.apiKeys, apiKey)
}

// generateAPIKey generates a random API key
func (a *AuthMiddleware) generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// generateRequestID generates a unique request ID
func (a *AuthMiddleware) generateRequestID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// writeUnauthorized writes an unauthorized response
func (a *AuthMiddleware) writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	
	response := middleware.StandardResponse{
		Success:   false,
		Error:     &middleware.ErrorInfo{
			Code:    "UNAUTHORIZED",
			Message: message,
		},
		RequestID: a.generateRequestID(),
		Timestamp: time.Now(),
	}
	
	// Simple JSON encoding (avoiding external dependencies)
	w.Write([]byte(fmt.Sprintf(`{"success":false,"error":{"code":"UNAUTHORIZED","message":"%s"},"request_id":"%s","timestamp":"%s"}`,
		message, response.RequestID, response.Timestamp.Format(time.RFC3339))))
}

// AuthenticateRequest is a helper function for manual authentication
func (a *AuthMiddleware) AuthenticateRequest(r *http.Request) (string, error) {
	token, err := a.extractToken(r)
	if err != nil {
		return "", err
	}
	return a.validateToken(token)
}

