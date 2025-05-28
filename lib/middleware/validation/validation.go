package validation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/coder/agentapi/lib/middleware"
)

// ValidationMiddleware handles request validation for AgentAPI
type ValidationMiddleware struct {
	config  middleware.ValidationConfig
	enabled bool
}

// NewValidationMiddleware creates a new validation middleware instance
func NewValidationMiddleware(config middleware.ValidationConfig) *ValidationMiddleware {
	return &ValidationMiddleware{
		config:  config,
		enabled: config.Enabled,
	}
}

// Name returns the middleware name
func (v *ValidationMiddleware) Name() string {
	return "validation"
}

// IsEnabled returns whether the middleware is enabled
func (v *ValidationMiddleware) IsEnabled() bool {
	return v.enabled
}

// Configure configures the middleware with the provided config
func (v *ValidationMiddleware) Configure(config interface{}) error {
	if validationConfig, ok := config.(middleware.ValidationConfig); ok {
		v.config = validationConfig
		v.enabled = validationConfig.Enabled
		return nil
	}
	return fmt.Errorf("invalid validation config type")
}

// Handler returns the HTTP middleware handler
func (v *ValidationMiddleware) Handler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !v.enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Validate request size
			if err := v.validateRequestSize(r); err != nil {
				v.writeBadRequest(w, err.Error())
				return
			}

			// Validate headers
			if v.config.ValidateHeaders {
				if err := v.validateHeaders(r); err != nil {
					v.writeBadRequest(w, err.Error())
					return
				}
			}

			// Validate content type for POST/PUT requests
			if r.Method == "POST" || r.Method == "PUT" {
				if err := v.validateContentType(r); err != nil {
					v.writeBadRequest(w, err.Error())
					return
				}
			}

			// Validate JSON body for applicable requests
			if v.hasJSONBody(r) {
				body, err := v.validateAndBufferJSON(r)
				if err != nil {
					v.writeBadRequest(w, err.Error())
					return
				}
				
				// Replace the body with buffered content
				r.Body = io.NopCloser(bytes.NewReader(body))
			}

			// Add validation context
			ctx := v.addValidationContext(r.Context())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// validateRequestSize validates the request size
func (v *ValidationMiddleware) validateRequestSize(r *http.Request) error {
	if v.config.MaxRequestSize > 0 && r.ContentLength > int64(v.config.MaxRequestSize) {
		return fmt.Errorf("request size %d exceeds maximum allowed size %d", 
			r.ContentLength, v.config.MaxRequestSize)
	}
	return nil
}

// validateHeaders validates required headers
func (v *ValidationMiddleware) validateHeaders(r *http.Request) error {
	// Validate User-Agent header
	if userAgent := r.Header.Get("User-Agent"); userAgent == "" {
		if v.config.StrictMode {
			return fmt.Errorf("missing User-Agent header")
		}
	}

	// Validate Accept header for API requests
	if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/v1/") {
		accept := r.Header.Get("Accept")
		if accept != "" && !strings.Contains(accept, "application/json") && !strings.Contains(accept, "*/*") {
			return fmt.Errorf("unsupported Accept header: %s", accept)
		}
	}

	return nil
}

// validateContentType validates the content type for requests with bodies
func (v *ValidationMiddleware) validateContentType(r *http.Request) error {
	if r.ContentLength == 0 {
		return nil // No body, no content type required
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return fmt.Errorf("missing Content-Type header")
	}

	// Check for supported content types
	supportedTypes := []string{
		"application/json",
		"application/x-www-form-urlencoded",
		"multipart/form-data",
		"text/plain",
	}

	for _, supportedType := range supportedTypes {
		if strings.HasPrefix(contentType, supportedType) {
			return nil
		}
	}

	return fmt.Errorf("unsupported Content-Type: %s", contentType)
}

// hasJSONBody checks if the request has a JSON body
func (v *ValidationMiddleware) hasJSONBody(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	return strings.HasPrefix(contentType, "application/json") && r.ContentLength > 0
}

// validateAndBufferJSON validates and buffers JSON request body
func (v *ValidationMiddleware) validateAndBufferJSON(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	r.Body.Close()

	if len(body) == 0 {
		return body, nil
	}

	// Validate JSON syntax
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Additional JSON validation rules
	if v.config.StrictMode {
		if err := v.validateJSONStructure(jsonData); err != nil {
			return nil, err
		}
	}

	return body, nil
}

// validateJSONStructure performs additional JSON structure validation
func (v *ValidationMiddleware) validateJSONStructure(data interface{}) error {
	switch d := data.(type) {
	case map[string]interface{}:
		// Validate object structure
		for key, value := range d {
			if key == "" {
				return fmt.Errorf("empty object key not allowed")
			}
			if err := v.validateJSONStructure(value); err != nil {
				return err
			}
		}
	case []interface{}:
		// Validate array structure
		for _, item := range d {
			if err := v.validateJSONStructure(item); err != nil {
				return err
			}
		}
	case string:
		// Validate string length (prevent extremely long strings)
		if len(d) > 10000 {
			return fmt.Errorf("string value too long: %d characters", len(d))
		}
	}
	return nil
}

// addValidationContext adds validation metadata to the request context
func (v *ValidationMiddleware) addValidationContext(ctx context.Context) context.Context {
	reqCtx := middleware.GetRequestContext(ctx)
	if reqCtx == nil {
		reqCtx = &middleware.RequestContext{
			RequestID: v.generateRequestID(),
			Timestamp: time.Now(),
			Metadata:  make(map[string]interface{}),
		}
	}
	
	reqCtx.Metadata["validation"] = map[string]interface{}{
		"validated_at": time.Now(),
		"strict_mode":  v.config.StrictMode,
	}
	
	return middleware.SetRequestContext(ctx, reqCtx)
}

// generateRequestID generates a unique request ID
func (v *ValidationMiddleware) generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// writeBadRequest writes a bad request response
func (v *ValidationMiddleware) writeBadRequest(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	
	response := middleware.StandardResponse{
		Success:   false,
		Error: &middleware.ErrorInfo{
			Code:    "VALIDATION_ERROR",
			Message: message,
		},
		RequestID: v.generateRequestID(),
		Timestamp: time.Now(),
	}
	
	// Simple JSON encoding
	w.Write([]byte(fmt.Sprintf(`{"success":false,"error":{"code":"VALIDATION_ERROR","message":"%s"},"request_id":"%s","timestamp":"%s"}`,
		message, response.RequestID, response.Timestamp.Format(time.RFC3339))))
}

// ValidateMessageRequest validates a message request specifically
func (v *ValidationMiddleware) ValidateMessageRequest(data map[string]interface{}) error {
	// Validate required fields
	content, hasContent := data["content"]
	if !hasContent {
		return fmt.Errorf("missing required field: content")
	}
	
	contentStr, ok := content.(string)
	if !ok {
		return fmt.Errorf("content must be a string")
	}
	
	if len(contentStr) == 0 {
		return fmt.Errorf("content cannot be empty")
	}
	
	if len(contentStr) > 10000 {
		return fmt.Errorf("content too long: maximum 10000 characters")
	}
	
	// Validate message type if present
	if msgType, hasType := data["type"]; hasType {
		typeStr, ok := msgType.(string)
		if !ok {
			return fmt.Errorf("type must be a string")
		}
		
		validTypes := []string{"user", "raw"}
		isValid := false
		for _, validType := range validTypes {
			if typeStr == validType {
				isValid = true
				break
			}
		}
		
		if !isValid {
			return fmt.Errorf("invalid message type: %s", typeStr)
		}
	}
	
	return nil
}

// ValidateAgentRequest validates agent-related requests
func (v *ValidationMiddleware) ValidateAgentRequest(data map[string]interface{}) error {
	// Validate agent ID if present
	if agentID, hasAgentID := data["agent_id"]; hasAgentID {
		agentIDStr, ok := agentID.(string)
		if !ok {
			return fmt.Errorf("agent_id must be a string")
		}
		
		if len(agentIDStr) == 0 {
			return fmt.Errorf("agent_id cannot be empty")
		}
		
		// Basic agent ID format validation
		if len(agentIDStr) > 100 {
			return fmt.Errorf("agent_id too long: maximum 100 characters")
		}
	}
	
	return nil
}

