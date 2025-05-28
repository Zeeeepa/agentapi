package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/agentapi/lib/httpapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

// IntegrateWithServer integrates the unified middleware with the existing AgentAPI server
func IntegrateWithServer(server *httpapi.Server, config MiddlewareConfig, logger *slog.Logger) (*Manager, error) {
	// Validate configuration
	if err := ValidateConfig(config); err != nil {
		return nil, err
	}

	// Create middleware manager
	manager := NewManager(config, logger)

	// Get the router from the server (this would need to be exposed by the server)
	// For now, we'll show how it would be integrated
	
	return manager, nil
}

// CreateEnhancedServer creates a new AgentAPI server with unified middleware
func CreateEnhancedServer(ctx context.Context, port int, middlewareConfig MiddlewareConfig, logger *slog.Logger) (*EnhancedServer, error) {
	// Validate middleware configuration
	if err := ValidateConfig(middlewareConfig); err != nil {
		return nil, err
	}

	// Create router
	router := chi.NewRouter()

	// Create middleware manager
	manager := NewManager(middlewareConfig, logger)

	// Apply CORS middleware first (before our middleware chain)
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-API-Key", "X-Claude-Session-ID"},
		ExposedHeaders:   []string{"X-Request-ID", "X-Processing-Time", "X-RateLimit-Limit", "X-RateLimit-Remaining"},
		AllowCredentials: true,
		MaxAge:           300,
	})
	router.Use(corsMiddleware.Handler)

	// Apply unified middleware chain
	manager.ApplyToRouter(router)

	// Create enhanced server
	server := &EnhancedServer{
		router:    router,
		manager:   manager,
		port:      port,
		logger:    logger,
		ctx:       ctx,
	}

	// Register API routes
	server.registerRoutes()

	// Start middleware manager
	if err := manager.Start(ctx); err != nil {
		return nil, err
	}

	return server, nil
}

// EnhancedServer represents an AgentAPI server with unified middleware
type EnhancedServer struct {
	router  chi.Router
	manager *Manager
	port    int
	logger  *slog.Logger
	ctx     context.Context
	srv     *http.Server
}

// registerRoutes registers all API routes with the enhanced server
func (s *EnhancedServer) registerRoutes() {
	// Health check endpoint
	s.router.Get("/health", s.handleHealth)
	
	// Middleware status endpoint
	s.router.Get("/middleware/status", s.handleMiddlewareStatus)
	
	// Middleware configuration endpoint
	s.router.Get("/middleware/config", s.handleMiddlewareConfig)
	s.router.Put("/middleware/config", s.handleUpdateMiddlewareConfig)

	// Authentication endpoints
	s.router.Post("/auth/login", s.handleLogin)
	s.router.Post("/auth/logout", s.handleLogout)
	s.router.Post("/auth/refresh", s.handleRefresh)

	// Agent management endpoints (enhanced with middleware)
	s.router.Route("/agents", func(r chi.Router) {
		r.Get("/", s.handleGetAgents)
		r.Post("/", s.handleCreateAgent)
		r.Route("/{agentId}", func(r chi.Router) {
			r.Get("/", s.handleGetAgent)
			r.Put("/", s.handleUpdateAgent)
			r.Delete("/", s.handleDeleteAgent)
			r.Post("/messages", s.handleSendMessage)
			r.Get("/messages", s.handleGetMessages)
			r.Get("/status", s.handleGetAgentStatus)
		})
	})

	// Claude Code integration endpoints
	s.router.Route("/claude", func(r chi.Router) {
		r.Post("/message", s.handleClaudeMessage)
		r.Get("/session", s.handleGetClaudeSession)
		r.Delete("/session/{sessionId}", s.handleDeleteClaudeSession)
		r.Get("/status", s.handleGetClaudeStatus)
	})

	// Real-time endpoints
	s.router.Get("/ws", s.handleWebSocket)
	s.router.Get("/events", s.handleSSE)

	// API documentation
	s.router.Get("/docs", s.handleDocs)
	s.router.Get("/openapi.json", s.handleOpenAPI)
}

// Start starts the enhanced server
func (s *EnhancedServer) Start() error {
	s.srv = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: s.router,
	}

	s.logger.Info("Starting enhanced AgentAPI server", "port", s.port)
	return s.srv.ListenAndServe()
}

// Stop gracefully stops the enhanced server
func (s *EnhancedServer) Stop(ctx context.Context) error {
	if s.srv != nil {
		return s.srv.Shutdown(ctx)
	}
	return nil
}

// Handler implementations

func (s *EnhancedServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   "1.0.0",
		"middleware": s.manager.GetStatus(),
	}
	
	s.manager.GetResponseMiddleware().WriteSuccessResponse(w, r, response)
}

func (s *EnhancedServer) handleMiddlewareStatus(w http.ResponseWriter, r *http.Request) {
	status := s.manager.GetStatus()
	s.manager.GetResponseMiddleware().WriteSuccessResponse(w, r, status)
}

func (s *EnhancedServer) handleMiddlewareConfig(w http.ResponseWriter, r *http.Request) {
	config := s.manager.GetConfig()
	s.manager.GetResponseMiddleware().WriteSuccessResponse(w, r, config)
}

func (s *EnhancedServer) handleUpdateMiddlewareConfig(w http.ResponseWriter, r *http.Request) {
	var config MiddlewareConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		s.manager.GetErrorMiddleware().HandleError(w, r, err, http.StatusBadRequest)
		return
	}

	if err := s.manager.UpdateConfig(config); err != nil {
		s.manager.GetErrorMiddleware().HandleError(w, r, err, http.StatusBadRequest)
		return
	}

	s.manager.GetResponseMiddleware().WriteSuccessResponse(w, r, map[string]string{
		"message": "Configuration updated successfully",
	})
}

func (s *EnhancedServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	var loginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		s.manager.GetErrorMiddleware().HandleError(w, r, err, http.StatusBadRequest)
		return
	}

	// Validate credentials (simplified)
	if loginReq.Username == "" || loginReq.Password == "" {
		s.manager.GetResponseMiddleware().WriteErrorResponse(w, r, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid username or password")
		return
	}

	// Generate JWT token
	token, err := s.manager.GetAuthMiddleware().GenerateJWT(loginReq.Username)
	if err != nil {
		s.manager.GetErrorMiddleware().HandleError(w, r, err, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"token":    token,
		"user_id":  loginReq.Username,
		"expires_in": s.manager.GetConfig().Auth.Expiration.Seconds(),
	}

	s.manager.GetResponseMiddleware().WriteSuccessResponse(w, r, response)
}

func (s *EnhancedServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, you would invalidate the token
	s.manager.GetResponseMiddleware().WriteSuccessResponse(w, r, map[string]string{
		"message": "Logged out successfully",
	})
}

func (s *EnhancedServer) handleRefresh(w http.ResponseWriter, r *http.Request) {
	// Extract current token and generate new one
	userID, err := s.manager.GetAuthMiddleware().AuthenticateRequest(r)
	if err != nil {
		s.manager.GetErrorMiddleware().HandleError(w, r, err, http.StatusUnauthorized)
		return
	}

	token, err := s.manager.GetAuthMiddleware().GenerateJWT(userID)
	if err != nil {
		s.manager.GetErrorMiddleware().HandleError(w, r, err, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"token":    token,
		"user_id":  userID,
		"expires_in": s.manager.GetConfig().Auth.Expiration.Seconds(),
	}

	s.manager.GetResponseMiddleware().WriteSuccessResponse(w, r, response)
}

// Placeholder handlers for other endpoints
func (s *EnhancedServer) handleGetAgents(w http.ResponseWriter, r *http.Request) {
	// Implementation would integrate with existing agent management
	s.manager.GetResponseMiddleware().WriteSuccessResponse(w, r, []interface{}{})
}

func (s *EnhancedServer) handleCreateAgent(w http.ResponseWriter, r *http.Request) {
	s.manager.GetResponseMiddleware().WriteSuccessResponse(w, r, map[string]string{
		"message": "Agent creation not implemented in this consolidation",
	})
}

func (s *EnhancedServer) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentId")
	s.manager.GetResponseMiddleware().WriteSuccessResponse(w, r, map[string]string{
		"agent_id": agentID,
		"message":  "Agent retrieval not implemented in this consolidation",
	})
}

func (s *EnhancedServer) handleUpdateAgent(w http.ResponseWriter, r *http.Request) {
	s.manager.GetResponseMiddleware().WriteSuccessResponse(w, r, map[string]string{
		"message": "Agent update not implemented in this consolidation",
	})
}

func (s *EnhancedServer) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	s.manager.GetResponseMiddleware().WriteNoContentResponse(w, r)
}

func (s *EnhancedServer) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentId")
	
	// Broadcast message update via sync middleware
	s.manager.GetSyncMiddleware().BroadcastMessageUpdate("msg_123", "Message sent", "processing", agentID)
	
	s.manager.GetResponseMiddleware().WriteSuccessResponse(w, r, map[string]string{
		"message": "Message sent successfully",
	})
}

func (s *EnhancedServer) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	s.manager.GetResponseMiddleware().WriteSuccessResponse(w, r, []interface{}{})
}

func (s *EnhancedServer) handleGetAgentStatus(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentId")
	
	// Broadcast agent status via sync middleware
	s.manager.GetSyncMiddleware().BroadcastAgentStatus(agentID, "running", "Processing request")
	
	s.manager.GetResponseMiddleware().WriteSuccessResponse(w, r, map[string]interface{}{
		"agent_id": agentID,
		"status":   "running",
		"message":  "Processing request",
	})
}

func (s *EnhancedServer) handleClaudeMessage(w http.ResponseWriter, r *http.Request) {
	// This would be handled by the Claude middleware
	s.manager.GetResponseMiddleware().WriteSuccessResponse(w, r, map[string]string{
		"message": "Claude message handled by middleware",
	})
}

func (s *EnhancedServer) handleGetClaudeSession(w http.ResponseWriter, r *http.Request) {
	s.manager.GetResponseMiddleware().WriteSuccessResponse(w, r, map[string]string{
		"message": "Claude session retrieval handled by middleware",
	})
}

func (s *EnhancedServer) handleDeleteClaudeSession(w http.ResponseWriter, r *http.Request) {
	s.manager.GetResponseMiddleware().WriteNoContentResponse(w, r)
}

func (s *EnhancedServer) handleGetClaudeStatus(w http.ResponseWriter, r *http.Request) {
	s.manager.GetResponseMiddleware().WriteSuccessResponse(w, r, map[string]string{
		"message": "Claude status handled by middleware",
	})
}

func (s *EnhancedServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// This would be handled by the sync middleware
	s.manager.GetResponseMiddleware().WriteSuccessResponse(w, r, map[string]string{
		"message": "WebSocket handled by sync middleware",
	})
}

func (s *EnhancedServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	// This would be handled by the sync middleware
	s.manager.GetResponseMiddleware().WriteSuccessResponse(w, r, map[string]string{
		"message": "SSE handled by sync middleware",
	})
}

func (s *EnhancedServer) handleDocs(w http.ResponseWriter, r *http.Request) {
	// Serve API documentation
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>AgentAPI Documentation</title>
</head>
<body>
    <h1>AgentAPI with Unified Middleware</h1>
    <p>This is the enhanced AgentAPI with consolidated middleware components.</p>
    <ul>
        <li><a href="/openapi.json">OpenAPI Specification</a></li>
        <li><a href="/health">Health Check</a></li>
        <li><a href="/middleware/status">Middleware Status</a></li>
    </ul>
</body>
</html>
	`))
}

func (s *EnhancedServer) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	// Return OpenAPI specification
	spec := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       "AgentAPI with Unified Middleware",
			"version":     "1.0.0",
			"description": "Enhanced AgentAPI with consolidated middleware components",
		},
		"servers": []map[string]interface{}{
			{"url": fmt.Sprintf("http://localhost:%d", s.port)},
		},
		"paths": map[string]interface{}{
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Server is healthy",
						},
					},
				},
			},
		},
	}
	
	s.manager.GetResponseMiddleware().WriteSuccessResponse(w, r, spec)
}
