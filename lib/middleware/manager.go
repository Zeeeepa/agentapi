package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/agentapi/lib/middleware/auth"
	"github.com/coder/agentapi/lib/middleware/claude"
	"github.com/coder/agentapi/lib/middleware/errors"
	"github.com/coder/agentapi/lib/middleware/response"
	"github.com/coder/agentapi/lib/middleware/sync"
	"github.com/coder/agentapi/lib/middleware/validation"
	"github.com/go-chi/chi/v5"
)

// Manager manages all middleware components for AgentAPI
type Manager struct {
	config     MiddlewareConfig
	logger     *slog.Logger
	auth       *auth.AuthMiddleware
	validation *validation.ValidationMiddleware
	response   *response.ResponseMiddleware
	sync       *sync.SyncMiddleware
	claude     *claude.ClaudeMiddleware
	errors     *errors.ErrorMiddleware
	chain      []func(http.Handler) http.Handler
}

// NewManager creates a new middleware manager
func NewManager(config MiddlewareConfig, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}

	manager := &Manager{
		config: config,
		logger: logger,
	}

	// Initialize middleware components
	manager.initializeMiddleware()
	
	// Build middleware chain
	manager.buildChain()

	return manager
}

// initializeMiddleware initializes all middleware components
func (m *Manager) initializeMiddleware() {
	// Initialize authentication middleware
	m.auth = auth.NewAuthMiddleware(m.config.Auth)
	
	// Initialize validation middleware
	m.validation = validation.NewValidationMiddleware(m.config.Validation)
	
	// Initialize response middleware
	m.response = response.NewResponseMiddleware(m.config.Response)
	
	// Initialize sync middleware
	m.sync = sync.NewSyncMiddleware(m.config.Sync, m.logger)
	
	// Initialize Claude middleware
	m.claude = claude.NewClaudeMiddleware(m.config.Claude)
	
	// Initialize error middleware
	m.errors = errors.NewErrorMiddleware(m.config.Errors, m.logger)
}

// buildChain builds the middleware chain in the correct order
func (m *Manager) buildChain() {
	m.chain = []func(http.Handler) http.Handler{}

	// Error handling should be first to catch all errors
	if m.errors.IsEnabled() {
		m.chain = append(m.chain, m.errors.Handler())
	}

	// Response formatting should be early to set headers
	if m.response.IsEnabled() {
		m.chain = append(m.chain, m.response.Handler())
	}

	// Authentication should be early but after error handling
	if m.auth.IsEnabled() {
		m.chain = append(m.chain, m.auth.Handler())
	}

	// Validation should come after authentication
	if m.validation.IsEnabled() {
		m.chain = append(m.chain, m.validation.Handler())
	}

	// Sync middleware for real-time features
	if m.sync.IsEnabled() {
		m.chain = append(m.chain, m.sync.Handler())
	}

	// Claude middleware for Claude Code integration
	if m.claude.IsEnabled() {
		m.chain = append(m.chain, m.claude.Handler())
	}
}

// ApplyToRouter applies all middleware to a Chi router
func (m *Manager) ApplyToRouter(router chi.Router) {
	for _, middleware := range m.chain {
		router.Use(middleware)
	}
}

// GetChain returns the middleware chain
func (m *Manager) GetChain() []func(http.Handler) http.Handler {
	return m.chain
}

// Start starts all middleware components that require background processing
func (m *Manager) Start(ctx context.Context) error {
	m.logger.Info("Starting middleware manager")

	// Start sync middleware if enabled
	if m.sync.IsEnabled() {
		m.sync.Start(ctx)
		m.logger.Info("Sync middleware started")
	}

	// Start cleanup routines
	go m.runCleanup(ctx)

	m.logger.Info("Middleware manager started successfully")
	return nil
}

// runCleanup runs periodic cleanup tasks
func (m *Manager) runCleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.performCleanup()
		case <-ctx.Done():
			m.logger.Info("Cleanup routine shutting down")
			return
		}
	}
}

// performCleanup performs periodic cleanup tasks
func (m *Manager) performCleanup() {
	// Clean up Claude sessions
	if m.claude.IsEnabled() {
		m.claude.CleanupSessions(30 * time.Minute)
	}

	// Log middleware statistics
	m.logStatistics()
}

// logStatistics logs middleware statistics
func (m *Manager) logStatistics() {
	stats := map[string]interface{}{
		"timestamp": time.Now(),
	}

	if m.sync.IsEnabled() {
		stats["sync_clients"] = m.sync.GetClientCount()
	}

	if m.claude.IsEnabled() {
		stats["claude_sessions"] = m.claude.GetSessionCount()
	}

	m.logger.Info("Middleware statistics", slog.Any("stats", stats))
}

// GetAuthMiddleware returns the authentication middleware
func (m *Manager) GetAuthMiddleware() *auth.AuthMiddleware {
	return m.auth
}

// GetValidationMiddleware returns the validation middleware
func (m *Manager) GetValidationMiddleware() *validation.ValidationMiddleware {
	return m.validation
}

// GetResponseMiddleware returns the response middleware
func (m *Manager) GetResponseMiddleware() *response.ResponseMiddleware {
	return m.response
}

// GetSyncMiddleware returns the sync middleware
func (m *Manager) GetSyncMiddleware() *sync.SyncMiddleware {
	return m.sync
}

// GetClaudeMiddleware returns the Claude middleware
func (m *Manager) GetClaudeMiddleware() *claude.ClaudeMiddleware {
	return m.claude
}

// GetErrorMiddleware returns the error middleware
func (m *Manager) GetErrorMiddleware() *errors.ErrorMiddleware {
	return m.errors
}

// UpdateConfig updates the middleware configuration
func (m *Manager) UpdateConfig(config MiddlewareConfig) error {
	m.config = config

	// Reconfigure all middleware components
	if err := m.auth.Configure(config.Auth); err != nil {
		return fmt.Errorf("failed to configure auth middleware: %w", err)
	}

	if err := m.validation.Configure(config.Validation); err != nil {
		return fmt.Errorf("failed to configure validation middleware: %w", err)
	}

	if err := m.response.Configure(config.Response); err != nil {
		return fmt.Errorf("failed to configure response middleware: %w", err)
	}

	if err := m.sync.Configure(config.Sync); err != nil {
		return fmt.Errorf("failed to configure sync middleware: %w", err)
	}

	if err := m.claude.Configure(config.Claude); err != nil {
		return fmt.Errorf("failed to configure claude middleware: %w", err)
	}

	if err := m.errors.Configure(config.Errors); err != nil {
		return fmt.Errorf("failed to configure errors middleware: %w", err)
	}

	// Rebuild the middleware chain
	m.buildChain()

	m.logger.Info("Middleware configuration updated")
	return nil
}

// GetConfig returns the current middleware configuration
func (m *Manager) GetConfig() MiddlewareConfig {
	return m.config
}

// GetStatus returns the status of all middleware components
func (m *Manager) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"timestamp": time.Now(),
		"auth": map[string]interface{}{
			"enabled": m.auth.IsEnabled(),
			"name":    m.auth.Name(),
		},
		"validation": map[string]interface{}{
			"enabled": m.validation.IsEnabled(),
			"name":    m.validation.Name(),
		},
		"response": map[string]interface{}{
			"enabled": m.response.IsEnabled(),
			"name":    m.response.Name(),
		},
		"sync": map[string]interface{}{
			"enabled":      m.sync.IsEnabled(),
			"name":         m.sync.Name(),
			"client_count": m.sync.GetClientCount(),
		},
		"claude": map[string]interface{}{
			"enabled":       m.claude.IsEnabled(),
			"name":          m.claude.Name(),
			"session_count": m.claude.GetSessionCount(),
		},
		"errors": map[string]interface{}{
			"enabled": m.errors.IsEnabled(),
			"name":    m.errors.Name(),
		},
	}

	return status
}

// CreateDefaultConfig creates a default middleware configuration
func CreateDefaultConfig() MiddlewareConfig {
	return MiddlewareConfig{
		Auth: AuthConfig{
			Enabled:    true,
			Type:       "bearer",
			Secret:     "default-secret-change-me",
			Expiration: 24 * time.Hour,
			Issuer:     "agentapi",
		},
		Validation: ValidationConfig{
			Enabled:         true,
			StrictMode:      false,
			MaxRequestSize:  10 * 1024 * 1024, // 10MB
			ValidateHeaders: true,
		},
		Response: ResponseConfig{
			Enabled:           true,
			StandardFormat:    true,
			IncludeTimestamp:  true,
			IncludeRequestID:  true,
			CompressionLevel:  6,
			CacheControl:      "no-cache",
		},
		Sync: SyncConfig{
			Enabled:           true,
			WebSocketEnabled:  true,
			SSEEnabled:        true,
			HeartbeatInterval: 30 * time.Second,
			BufferSize:        1000,
		},
		Claude: ClaudeConfig{
			Enabled:         true,
			APIEndpoint:     "http://localhost:8080/api/claude",
			Version:         "1.0",
			MaxRetries:      3,
			TimeoutSeconds:  30,
			RateLimitRPS:    10,
			EnableStreaming: true,
		},
		Errors: ErrorConfig{
			Enabled:         true,
			DetailedErrors:  false,
			LogErrors:       true,
			IncludeStack:    false,
			SanitizeSecrets: true,
		},
	}
}

// ValidateConfig validates the middleware configuration
func ValidateConfig(config MiddlewareConfig) error {
	// Validate auth config
	if config.Auth.Enabled {
		if config.Auth.Secret == "" {
			return fmt.Errorf("auth secret cannot be empty when auth is enabled")
		}
		if config.Auth.Type != "bearer" && config.Auth.Type != "api_key" {
			return fmt.Errorf("unsupported auth type: %s", config.Auth.Type)
		}
	}

	// Validate validation config
	if config.Validation.Enabled {
		if config.Validation.MaxRequestSize <= 0 {
			return fmt.Errorf("max request size must be positive")
		}
	}

	// Validate sync config
	if config.Sync.Enabled {
		if config.Sync.BufferSize <= 0 {
			return fmt.Errorf("sync buffer size must be positive")
		}
		if config.Sync.HeartbeatInterval <= 0 {
			return fmt.Errorf("heartbeat interval must be positive")
		}
	}

	// Validate Claude config
	if config.Claude.Enabled {
		if config.Claude.APIEndpoint == "" {
			return fmt.Errorf("Claude API endpoint cannot be empty when Claude is enabled")
		}
		if config.Claude.TimeoutSeconds <= 0 {
			return fmt.Errorf("Claude timeout must be positive")
		}
	}

	return nil
}

