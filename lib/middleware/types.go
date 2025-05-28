package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// MiddlewareConfig holds configuration for all middleware components
type MiddlewareConfig struct {
	Auth       AuthConfig       `json:"auth"`
	Validation ValidationConfig `json:"validation"`
	Response   ResponseConfig   `json:"response"`
	Sync       SyncConfig       `json:"sync"`
	Claude     ClaudeConfig     `json:"claude"`
	Errors     ErrorConfig      `json:"errors"`
}

// AuthConfig configures authentication middleware
type AuthConfig struct {
	Enabled    bool          `json:"enabled"`
	Type       string        `json:"type"` // "bearer", "api_key", "oauth"
	Secret     string        `json:"secret"`
	Expiration time.Duration `json:"expiration"`
	Issuer     string        `json:"issuer"`
}

// ValidationConfig configures request validation middleware
type ValidationConfig struct {
	Enabled         bool `json:"enabled"`
	StrictMode      bool `json:"strict_mode"`
	MaxRequestSize  int  `json:"max_request_size"`
	ValidateHeaders bool `json:"validate_headers"`
}

// ResponseConfig configures response formatting middleware
type ResponseConfig struct {
	Enabled           bool   `json:"enabled"`
	StandardFormat    bool   `json:"standard_format"`
	IncludeTimestamp  bool   `json:"include_timestamp"`
	IncludeRequestID  bool   `json:"include_request_id"`
	CompressionLevel  int    `json:"compression_level"`
	CacheControl      string `json:"cache_control"`
}

// SyncConfig configures real-time synchronization middleware
type SyncConfig struct {
	Enabled           bool          `json:"enabled"`
	WebSocketEnabled  bool          `json:"websocket_enabled"`
	SSEEnabled        bool          `json:"sse_enabled"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
	BufferSize        int           `json:"buffer_size"`
}

// ClaudeConfig configures Claude Code communication protocols
type ClaudeConfig struct {
	Enabled         bool   `json:"enabled"`
	APIEndpoint     string `json:"api_endpoint"`
	Version         string `json:"version"`
	MaxRetries      int    `json:"max_retries"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
	RateLimitRPS    int    `json:"rate_limit_rps"`
	EnableStreaming bool   `json:"enable_streaming"`
}

// ErrorConfig configures error handling middleware
type ErrorConfig struct {
	Enabled         bool `json:"enabled"`
	DetailedErrors  bool `json:"detailed_errors"`
	LogErrors       bool `json:"log_errors"`
	IncludeStack    bool `json:"include_stack"`
	SanitizeSecrets bool `json:"sanitize_secrets"`
}

// MiddlewareChain represents a chain of middleware functions
type MiddlewareChain struct {
	middlewares []func(http.Handler) http.Handler
	config      MiddlewareConfig
}

// RequestContext holds request-specific data
type RequestContext struct {
	RequestID   string                 `json:"request_id"`
	UserID      string                 `json:"user_id,omitempty"`
	AgentID     string                 `json:"agent_id,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ClaudeState *ClaudeState           `json:"claude_state,omitempty"`
}

// ClaudeState holds Claude Code specific state
type ClaudeState struct {
	SessionID     string            `json:"session_id"`
	ConversationID string           `json:"conversation_id"`
	Model         string            `json:"model"`
	Tools         []string          `json:"tools"`
	Context       map[string]string `json:"context"`
}

// StandardResponse represents the unified API response format
type StandardResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorInfo  `json:"error,omitempty"`
	Meta      *MetaInfo   `json:"meta,omitempty"`
	RequestID string      `json:"request_id"`
	Timestamp time.Time   `json:"timestamp"`
}

// ErrorInfo provides detailed error information
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Stack   string `json:"stack,omitempty"`
}

// MetaInfo provides additional response metadata
type MetaInfo struct {
	Version     string `json:"version"`
	ProcessTime string `json:"process_time"`
	RateLimit   *RateLimit `json:"rate_limit,omitempty"`
}

// RateLimit provides rate limiting information
type RateLimit struct {
	Limit     int `json:"limit"`
	Remaining int `json:"remaining"`
	Reset     int `json:"reset"`
}

// Middleware interface that all middleware components must implement
type Middleware interface {
	Name() string
	Handler() func(http.Handler) http.Handler
	Configure(config interface{}) error
	IsEnabled() bool
}

// ContextKey type for context keys
type ContextKey string

const (
	RequestContextKey ContextKey = "request_context"
	ClaudeStateKey    ContextKey = "claude_state"
	UserIDKey         ContextKey = "user_id"
	AgentIDKey        ContextKey = "agent_id"
)

// GetRequestContext retrieves the request context from the HTTP request context
func GetRequestContext(ctx context.Context) *RequestContext {
	if reqCtx, ok := ctx.Value(RequestContextKey).(*RequestContext); ok {
		return reqCtx
	}
	return nil
}

// SetRequestContext sets the request context in the HTTP request context
func SetRequestContext(ctx context.Context, reqCtx *RequestContext) context.Context {
	return context.WithValue(ctx, RequestContextKey, reqCtx)
}

// Router interface for middleware integration
type Router interface {
	Use(middlewares ...func(http.Handler) http.Handler)
	Route(pattern string, fn func(chi.Router)) chi.Router
	Get(pattern string, handlerFn http.HandlerFunc)
	Post(pattern string, handlerFn http.HandlerFunc)
	Put(pattern string, handlerFn http.HandlerFunc)
	Delete(pattern string, handlerFn http.HandlerFunc)
}

