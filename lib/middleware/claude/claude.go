package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/agentapi/lib/middleware"
)

// ClaudeMiddleware handles Claude Code communication protocols
type ClaudeMiddleware struct {
	config   middleware.ClaudeConfig
	enabled  bool
	client   *http.Client
	sessions map[string]*ClaudeSession
	mu       sync.RWMutex
}

// ClaudeSession represents an active Claude Code session
type ClaudeSession struct {
	ID             string            `json:"id"`
	ConversationID string            `json:"conversation_id"`
	Model          string            `json:"model"`
	Tools          []string          `json:"tools"`
	Context        map[string]string `json:"context"`
	CreatedAt      time.Time         `json:"created_at"`
	LastActivity   time.Time         `json:"last_activity"`
	Status         string            `json:"status"`
}

// ClaudeRequest represents a request to Claude Code
type ClaudeRequest struct {
	SessionID      string                 `json:"session_id,omitempty"`
	ConversationID string                 `json:"conversation_id,omitempty"`
	Model          string                 `json:"model,omitempty"`
	Message        string                 `json:"message"`
	Tools          []string               `json:"tools,omitempty"`
	Context        map[string]interface{} `json:"context,omitempty"`
	Stream         bool                   `json:"stream,omitempty"`
}

// ClaudeResponse represents a response from Claude Code
type ClaudeResponse struct {
	SessionID      string                 `json:"session_id"`
	ConversationID string                 `json:"conversation_id"`
	Message        string                 `json:"message"`
	Status         string                 `json:"status"`
	Tools          []string               `json:"tools,omitempty"`
	Context        map[string]interface{} `json:"context,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
}

// NewClaudeMiddleware creates a new Claude middleware instance
func NewClaudeMiddleware(config middleware.ClaudeConfig) *ClaudeMiddleware {
	return &ClaudeMiddleware{
		config:   config,
		enabled:  config.Enabled,
		sessions: make(map[string]*ClaudeSession),
		client: &http.Client{
			Timeout: time.Duration(config.TimeoutSeconds) * time.Second,
		},
	}
}

// Name returns the middleware name
func (c *ClaudeMiddleware) Name() string {
	return "claude"
}

// IsEnabled returns whether the middleware is enabled
func (c *ClaudeMiddleware) IsEnabled() bool {
	return c.enabled
}

// Configure configures the middleware with the provided config
func (c *ClaudeMiddleware) Configure(config interface{}) error {
	if claudeConfig, ok := config.(middleware.ClaudeConfig); ok {
		c.config = claudeConfig
		c.enabled = claudeConfig.Enabled
		c.client.Timeout = time.Duration(claudeConfig.TimeoutSeconds) * time.Second
		return nil
	}
	return fmt.Errorf("invalid claude config type")
}

// Handler returns the HTTP middleware handler
func (c *ClaudeMiddleware) Handler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !c.enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Check if this is a Claude-related request
			if c.isClaudeRequest(r) {
				c.handleClaudeRequest(w, r)
				return
			}

			// Add Claude context to regular requests
			ctx := c.addClaudeContext(r.Context(), r)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// isClaudeRequest checks if the request is Claude-related
func (c *ClaudeMiddleware) isClaudeRequest(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, "/claude/") ||
		   strings.HasPrefix(r.URL.Path, "/api/claude/") ||
		   r.Header.Get("X-Claude-Request") == "true"
}

// handleClaudeRequest handles Claude-specific requests
func (c *ClaudeMiddleware) handleClaudeRequest(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		c.handleClaudeMessage(w, r)
	case "GET":
		if strings.HasSuffix(r.URL.Path, "/session") {
			c.handleGetSession(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/status") {
			c.handleGetStatus(w, r)
		} else {
			c.writeError(w, http.StatusNotFound, "ENDPOINT_NOT_FOUND", "Claude endpoint not found")
		}
	case "DELETE":
		if strings.Contains(r.URL.Path, "/session/") {
			c.handleDeleteSession(w, r)
		} else {
			c.writeError(w, http.StatusNotFound, "ENDPOINT_NOT_FOUND", "Claude endpoint not found")
		}
	default:
		c.writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed for Claude endpoint")
	}
}

// handleClaudeMessage handles Claude message requests
func (c *ClaudeMiddleware) handleClaudeMessage(w http.ResponseWriter, r *http.Request) {
	var req ClaudeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid Claude request format")
		return
	}

	// Get or create session
	session := c.getOrCreateSession(req.SessionID, req.ConversationID, req.Model)
	
	// Update session context
	if req.Context != nil {
		for key, value := range req.Context {
			if strValue, ok := value.(string); ok {
				session.Context[key] = strValue
			}
		}
	}

	// Forward request to Claude Code
	response, err := c.forwardToClaudeCode(session, req)
	if err != nil {
		c.writeError(w, http.StatusInternalServerError, "CLAUDE_ERROR", err.Error())
		return
	}

	// Update session activity
	session.LastActivity = time.Now()
	session.Status = response.Status

	// Write response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetSession handles session retrieval requests
func (c *ClaudeMiddleware) handleGetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		c.writeError(w, http.StatusBadRequest, "MISSING_SESSION_ID", "Session ID is required")
		return
	}

	c.mu.RLock()
	session, exists := c.sessions[sessionID]
	c.mu.RUnlock()

	if !exists {
		c.writeError(w, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// handleGetStatus handles status requests
func (c *ClaudeMiddleware) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	c.mu.RLock()
	sessionCount := len(c.sessions)
	c.mu.RUnlock()

	status := map[string]interface{}{
		"enabled":       c.enabled,
		"endpoint":      c.config.APIEndpoint,
		"version":       c.config.Version,
		"session_count": sessionCount,
		"timestamp":     time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleDeleteSession handles session deletion requests
func (c *ClaudeMiddleware) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := strings.TrimPrefix(r.URL.Path, "/claude/session/")
	sessionID = strings.TrimPrefix(sessionID, "/api/claude/session/")
	
	if sessionID == "" {
		c.writeError(w, http.StatusBadRequest, "MISSING_SESSION_ID", "Session ID is required")
		return
	}

	c.mu.Lock()
	delete(c.sessions, sessionID)
	c.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

// getOrCreateSession gets an existing session or creates a new one
func (c *ClaudeMiddleware) getOrCreateSession(sessionID, conversationID, model string) *ClaudeSession {
	c.mu.Lock()
	defer c.mu.Unlock()

	if sessionID != "" {
		if session, exists := c.sessions[sessionID]; exists {
			return session
		}
	}

	// Create new session
	if sessionID == "" {
		sessionID = c.generateSessionID()
	}
	if conversationID == "" {
		conversationID = c.generateConversationID()
	}
	if model == "" {
		model = "claude-3-sonnet"
	}

	session := &ClaudeSession{
		ID:             sessionID,
		ConversationID: conversationID,
		Model:          model,
		Tools:          []string{},
		Context:        make(map[string]string),
		CreatedAt:      time.Now(),
		LastActivity:   time.Now(),
		Status:         "active",
	}

	c.sessions[sessionID] = session
	return session
}

// forwardToClaudeCode forwards the request to Claude Code
func (c *ClaudeMiddleware) forwardToClaudeCode(session *ClaudeSession, req ClaudeRequest) (*ClaudeResponse, error) {
	// Prepare request payload
	payload := map[string]interface{}{
		"session_id":      session.ID,
		"conversation_id": session.ConversationID,
		"model":           session.Model,
		"message":         req.Message,
		"tools":           session.Tools,
		"context":         session.Context,
		"stream":          req.Stream,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", c.config.APIEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "AgentAPI/1.0")

	// Send request with retries
	var resp *http.Response
	for i := 0; i <= c.config.MaxRetries; i++ {
		resp, err = c.client.Do(httpReq)
		if err == nil && resp.StatusCode < 500 {
			break
		}
		if i < c.config.MaxRetries {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Claude Code returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var claudeResp ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	claudeResp.Timestamp = time.Now()
	return &claudeResp, nil
}

// addClaudeContext adds Claude context to the request
func (c *ClaudeMiddleware) addClaudeContext(ctx context.Context, r *http.Request) context.Context {
	reqCtx := middleware.GetRequestContext(ctx)
	if reqCtx == nil {
		reqCtx = &middleware.RequestContext{
			RequestID: c.generateRequestID(),
			Timestamp: time.Now(),
			Metadata:  make(map[string]interface{}),
		}
	}

	// Add Claude state if available
	sessionID := r.Header.Get("X-Claude-Session-ID")
	if sessionID != "" {
		c.mu.RLock()
		if session, exists := c.sessions[sessionID]; exists {
			reqCtx.ClaudeState = &middleware.ClaudeState{
				SessionID:      session.ID,
				ConversationID: session.ConversationID,
				Model:          session.Model,
				Tools:          session.Tools,
				Context:        session.Context,
			}
		}
		c.mu.RUnlock()
	}

	return middleware.SetRequestContext(ctx, reqCtx)
}

// generateSessionID generates a unique session ID
func (c *ClaudeMiddleware) generateSessionID() string {
	return fmt.Sprintf("claude_session_%d", time.Now().UnixNano())
}

// generateConversationID generates a unique conversation ID
func (c *ClaudeMiddleware) generateConversationID() string {
	return fmt.Sprintf("claude_conv_%d", time.Now().UnixNano())
}

// generateRequestID generates a unique request ID
func (c *ClaudeMiddleware) generateRequestID() string {
	return fmt.Sprintf("claude_req_%d", time.Now().UnixNano())
}

// writeError writes an error response
func (c *ClaudeMiddleware) writeError(w http.ResponseWriter, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := middleware.StandardResponse{
		Success: false,
		Error: &middleware.ErrorInfo{
			Code:    code,
			Message: message,
		},
		RequestID: c.generateRequestID(),
		Timestamp: time.Now(),
	}
	
	json.NewEncoder(w).Encode(response)
}

// CleanupSessions removes inactive sessions
func (c *ClaudeMiddleware) CleanupSessions(maxAge time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for sessionID, session := range c.sessions {
		if session.LastActivity.Before(cutoff) {
			delete(c.sessions, sessionID)
		}
	}
}

// GetSessionCount returns the number of active sessions
func (c *ClaudeMiddleware) GetSessionCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.sessions)
}

// GetSession returns a session by ID
func (c *ClaudeMiddleware) GetSession(sessionID string) (*ClaudeSession, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	session, exists := c.sessions[sessionID]
	return session, exists
}

