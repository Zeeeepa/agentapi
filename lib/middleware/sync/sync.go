package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/agentapi/lib/middleware"
	"github.com/gorilla/websocket"
)

// SyncMiddleware handles real-time synchronization for AgentAPI
type SyncMiddleware struct {
	config    middleware.SyncConfig
	enabled   bool
	logger    *slog.Logger
	upgrader  websocket.Upgrader
	clients   map[string]*Client
	mu        sync.RWMutex
	broadcast chan *SyncMessage
	register  chan *Client
	unregister chan *Client
}

// Client represents a connected WebSocket client
type Client struct {
	ID         string          `json:"id"`
	UserID     string          `json:"user_id,omitempty"`
	AgentID    string          `json:"agent_id,omitempty"`
	Connection *websocket.Conn `json:"-"`
	Send       chan *SyncMessage `json:"-"`
	LastPing   time.Time       `json:"last_ping"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// SyncMessage represents a real-time synchronization message
type SyncMessage struct {
	Type      string                 `json:"type"`
	Data      interface{}            `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	ClientID  string                 `json:"client_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	AgentID   string                 `json:"agent_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SyncEvent represents different types of sync events
type SyncEvent struct {
	AgentStatus    *AgentStatusEvent    `json:"agent_status,omitempty"`
	MessageUpdate  *MessageUpdateEvent  `json:"message_update,omitempty"`
	SessionUpdate  *SessionUpdateEvent  `json:"session_update,omitempty"`
	ErrorEvent     *ErrorEvent          `json:"error_event,omitempty"`
	HeartbeatEvent *HeartbeatEvent      `json:"heartbeat_event,omitempty"`
}

// AgentStatusEvent represents agent status changes
type AgentStatusEvent struct {
	AgentID   string    `json:"agent_id"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// MessageUpdateEvent represents message updates
type MessageUpdateEvent struct {
	MessageID string    `json:"message_id"`
	Content   string    `json:"content"`
	Status    string    `json:"status"`
	AgentID   string    `json:"agent_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// SessionUpdateEvent represents session updates
type SessionUpdateEvent struct {
	SessionID string                 `json:"session_id"`
	Status    string                 `json:"status"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ErrorEvent represents error events
type ErrorEvent struct {
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// HeartbeatEvent represents heartbeat events
type HeartbeatEvent struct {
	ClientID  string    `json:"client_id"`
	Timestamp time.Time `json:"timestamp"`
}

// NewSyncMiddleware creates a new sync middleware instance
func NewSyncMiddleware(config middleware.SyncConfig, logger *slog.Logger) *SyncMiddleware {
	if logger == nil {
		logger = slog.Default()
	}

	return &SyncMiddleware{
		config:  config,
		enabled: config.Enabled,
		logger:  logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		clients:    make(map[string]*Client),
		broadcast:  make(chan *SyncMessage, config.BufferSize),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Name returns the middleware name
func (s *SyncMiddleware) Name() string {
	return "sync"
}

// IsEnabled returns whether the middleware is enabled
func (s *SyncMiddleware) IsEnabled() bool {
	return s.enabled
}

// Configure configures the middleware with the provided config
func (s *SyncMiddleware) Configure(config interface{}) error {
	if syncConfig, ok := config.(middleware.SyncConfig); ok {
		s.config = syncConfig
		s.enabled = syncConfig.Enabled
		return nil
	}
	return fmt.Errorf("invalid sync config type")
}

// Handler returns the HTTP middleware handler
func (s *SyncMiddleware) Handler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !s.enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Handle WebSocket upgrade requests
			if s.isWebSocketRequest(r) {
				s.handleWebSocket(w, r)
				return
			}

			// Handle SSE requests
			if s.isSSERequest(r) {
				s.handleSSE(w, r)
				return
			}

			// Add sync context to regular requests
			ctx := s.addSyncContext(r.Context(), r)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Start starts the sync middleware hub
func (s *SyncMiddleware) Start(ctx context.Context) {
	go s.runHub(ctx)
	go s.runHeartbeat(ctx)
}

// runHub runs the main synchronization hub
func (s *SyncMiddleware) runHub(ctx context.Context) {
	for {
		select {
		case client := <-s.register:
			s.registerClient(client)
		case client := <-s.unregister:
			s.unregisterClient(client)
		case message := <-s.broadcast:
			s.broadcastMessage(message)
		case <-ctx.Done():
			s.logger.Info("Sync hub shutting down")
			return
		}
	}
}

// runHeartbeat runs the heartbeat mechanism
func (s *SyncMiddleware) runHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(s.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.sendHeartbeat()
		case <-ctx.Done():
			return
		}
	}
}

// isWebSocketRequest checks if the request is a WebSocket upgrade request
func (s *SyncMiddleware) isWebSocketRequest(r *http.Request) bool {
	return s.config.WebSocketEnabled &&
		   strings.ToLower(r.Header.Get("Connection")) == "upgrade" &&
		   strings.ToLower(r.Header.Get("Upgrade")) == "websocket"
}

// isSSERequest checks if the request is an SSE request
func (s *SyncMiddleware) isSSERequest(r *http.Request) bool {
	return s.config.SSEEnabled &&
		   strings.Contains(r.Header.Get("Accept"), "text/event-stream")
}

// handleWebSocket handles WebSocket connections
func (s *SyncMiddleware) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("WebSocket upgrade failed", "error", err)
		return
	}

	// Create client
	client := &Client{
		ID:         s.generateClientID(),
		Connection: conn,
		Send:       make(chan *SyncMessage, s.config.BufferSize),
		LastPing:   time.Now(),
		Metadata:   make(map[string]interface{}),
	}

	// Extract user and agent IDs from request context
	if reqCtx := middleware.GetRequestContext(r.Context()); reqCtx != nil {
		client.UserID = reqCtx.UserID
		client.AgentID = reqCtx.AgentID
	}

	// Register client
	s.register <- client

	// Start client goroutines
	go s.handleClientWrite(client)
	go s.handleClientRead(client)
}

// handleSSE handles Server-Sent Events connections
func (s *SyncMiddleware) handleSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a channel for this SSE connection
	clientID := s.generateClientID()
	eventChan := make(chan *SyncMessage, s.config.BufferSize)

	// Register SSE client (simplified)
	s.mu.Lock()
	s.clients[clientID] = &Client{
		ID:       clientID,
		Send:     eventChan,
		LastPing: time.Now(),
		Metadata: map[string]interface{}{"type": "sse"},
	}
	s.mu.Unlock()

	// Clean up on disconnect
	defer func() {
		s.mu.Lock()
		delete(s.clients, clientID)
		close(eventChan)
		s.mu.Unlock()
	}()

	// Send events
	for {
		select {
		case message := <-eventChan:
			data, err := json.Marshal(message)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

// handleClientWrite handles writing messages to WebSocket clients
func (s *SyncMiddleware) handleClientWrite(client *Client) {
	defer client.Connection.Close()

	for {
		select {
		case message, ok := <-client.Send:
			if !ok {
				client.Connection.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			client.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Connection.WriteJSON(message); err != nil {
				s.logger.Error("WebSocket write failed", "client_id", client.ID, "error", err)
				return
			}
		}
	}
}

// handleClientRead handles reading messages from WebSocket clients
func (s *SyncMiddleware) handleClientRead(client *Client) {
	defer func() {
		s.unregister <- client
		client.Connection.Close()
	}()

	client.Connection.SetReadLimit(512)
	client.Connection.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Connection.SetPongHandler(func(string) error {
		client.Connection.SetReadDeadline(time.Now().Add(60 * time.Second))
		client.LastPing = time.Now()
		return nil
	})

	for {
		var message SyncMessage
		if err := client.Connection.ReadJSON(&message); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Error("WebSocket read error", "client_id", client.ID, "error", err)
			}
			break
		}

		// Handle client message
		s.handleClientMessage(client, &message)
	}
}

// handleClientMessage handles messages received from clients
func (s *SyncMiddleware) handleClientMessage(client *Client, message *SyncMessage) {
	message.ClientID = client.ID
	message.UserID = client.UserID
	message.Timestamp = time.Now()

	switch message.Type {
	case "ping":
		// Respond with pong
		pong := &SyncMessage{
			Type:      "pong",
			Timestamp: time.Now(),
			ClientID:  client.ID,
		}
		select {
		case client.Send <- pong:
		default:
			close(client.Send)
		}
	case "subscribe":
		// Handle subscription requests
		s.handleSubscription(client, message)
	default:
		// Broadcast other messages
		s.broadcast <- message
	}
}

// handleSubscription handles client subscription requests
func (s *SyncMiddleware) handleSubscription(client *Client, message *SyncMessage) {
	if data, ok := message.Data.(map[string]interface{}); ok {
		if agentID, exists := data["agent_id"]; exists {
			if agentIDStr, ok := agentID.(string); ok {
				client.AgentID = agentIDStr
				client.Metadata["subscribed_agent"] = agentIDStr
			}
		}
	}
}

// registerClient registers a new client
func (s *SyncMiddleware) registerClient(client *Client) {
	s.mu.Lock()
	s.clients[client.ID] = client
	s.mu.Unlock()

	s.logger.Info("Client registered", "client_id", client.ID, "user_id", client.UserID)

	// Send welcome message
	welcome := &SyncMessage{
		Type:      "welcome",
		Data:      map[string]interface{}{"client_id": client.ID},
		Timestamp: time.Now(),
	}
	
	select {
	case client.Send <- welcome:
	default:
		close(client.Send)
		delete(s.clients, client.ID)
	}
}

// unregisterClient unregisters a client
func (s *SyncMiddleware) unregisterClient(client *Client) {
	s.mu.Lock()
	if _, ok := s.clients[client.ID]; ok {
		delete(s.clients, client.ID)
		close(client.Send)
	}
	s.mu.Unlock()

	s.logger.Info("Client unregistered", "client_id", client.ID)
}

// broadcastMessage broadcasts a message to all relevant clients
func (s *SyncMiddleware) broadcastMessage(message *SyncMessage) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, client := range s.clients {
		// Filter messages based on client subscriptions
		if s.shouldReceiveMessage(client, message) {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(s.clients, client.ID)
			}
		}
	}
}

// shouldReceiveMessage determines if a client should receive a message
func (s *SyncMiddleware) shouldReceiveMessage(client *Client, message *SyncMessage) bool {
	// Send to all clients if no specific targeting
	if message.UserID == "" && message.AgentID == "" {
		return true
	}

	// Send to specific user
	if message.UserID != "" && client.UserID == message.UserID {
		return true
	}

	// Send to clients subscribed to specific agent
	if message.AgentID != "" && client.AgentID == message.AgentID {
		return true
	}

	return false
}

// sendHeartbeat sends heartbeat messages to all clients
func (s *SyncMiddleware) sendHeartbeat() {
	heartbeat := &SyncMessage{
		Type:      "heartbeat",
		Data:      &HeartbeatEvent{Timestamp: time.Now()},
		Timestamp: time.Now(),
	}

	s.broadcast <- heartbeat
}

// addSyncContext adds sync context to the request
func (s *SyncMiddleware) addSyncContext(ctx context.Context, r *http.Request) context.Context {
	reqCtx := middleware.GetRequestContext(ctx)
	if reqCtx == nil {
		reqCtx = &middleware.RequestContext{
			RequestID: s.generateRequestID(),
			Timestamp: time.Now(),
			Metadata:  make(map[string]interface{}),
		}
	}

	reqCtx.Metadata["sync"] = map[string]interface{}{
		"websocket_enabled": s.config.WebSocketEnabled,
		"sse_enabled":       s.config.SSEEnabled,
		"client_count":      len(s.clients),
	}

	return middleware.SetRequestContext(ctx, reqCtx)
}

// BroadcastAgentStatus broadcasts agent status changes
func (s *SyncMiddleware) BroadcastAgentStatus(agentID, status, message string) {
	event := &AgentStatusEvent{
		AgentID:   agentID,
		Status:    status,
		Message:   message,
		Timestamp: time.Now(),
	}

	syncMessage := &SyncMessage{
		Type:      "agent_status",
		Data:      event,
		AgentID:   agentID,
		Timestamp: time.Now(),
	}

	s.broadcast <- syncMessage
}

// BroadcastMessageUpdate broadcasts message updates
func (s *SyncMiddleware) BroadcastMessageUpdate(messageID, content, status, agentID string) {
	event := &MessageUpdateEvent{
		MessageID: messageID,
		Content:   content,
		Status:    status,
		AgentID:   agentID,
		Timestamp: time.Now(),
	}

	syncMessage := &SyncMessage{
		Type:      "message_update",
		Data:      event,
		AgentID:   agentID,
		Timestamp: time.Now(),
	}

	s.broadcast <- syncMessage
}

// BroadcastError broadcasts error events
func (s *SyncMiddleware) BroadcastError(code, message, details string) {
	event := &ErrorEvent{
		Code:      code,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
	}

	syncMessage := &SyncMessage{
		Type:      "error",
		Data:      event,
		Timestamp: time.Now(),
	}

	s.broadcast <- syncMessage
}

// generateClientID generates a unique client ID
func (s *SyncMiddleware) generateClientID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}

// generateRequestID generates a unique request ID
func (s *SyncMiddleware) generateRequestID() string {
	return fmt.Sprintf("sync_req_%d", time.Now().UnixNano())
}

// GetClientCount returns the number of connected clients
func (s *SyncMiddleware) GetClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

// GetClients returns a list of connected clients
func (s *SyncMiddleware) GetClients() []*Client {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clients := make([]*Client, 0, len(s.clients))
	for _, client := range s.clients {
		clients = append(clients, client)
	}
	return clients
}

