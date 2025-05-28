package errors

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/coder/agentapi/lib/middleware"
)

// ErrorMiddleware handles error processing and logging for AgentAPI
type ErrorMiddleware struct {
	config  middleware.ErrorConfig
	enabled bool
	logger  *slog.Logger
}

// ErrorContext holds error-specific context information
type ErrorContext struct {
	RequestID    string                 `json:"request_id"`
	UserID       string                 `json:"user_id,omitempty"`
	Path         string                 `json:"path"`
	Method       string                 `json:"method"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	RemoteAddr   string                 `json:"remote_addr,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	Stack        []string               `json:"stack,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// PanicRecovery represents a recovered panic
type PanicRecovery struct {
	Value interface{} `json:"value"`
	Stack []string    `json:"stack"`
}

// NewErrorMiddleware creates a new error middleware instance
func NewErrorMiddleware(config middleware.ErrorConfig, logger *slog.Logger) *ErrorMiddleware {
	if logger == nil {
		logger = slog.Default()
	}
	
	return &ErrorMiddleware{
		config:  config,
		enabled: config.Enabled,
		logger:  logger,
	}
}

// Name returns the middleware name
func (e *ErrorMiddleware) Name() string {
	return "errors"
}

// IsEnabled returns whether the middleware is enabled
func (e *ErrorMiddleware) IsEnabled() bool {
	return e.enabled
}

// Configure configures the middleware with the provided config
func (e *ErrorMiddleware) Configure(config interface{}) error {
	if errorConfig, ok := config.(middleware.ErrorConfig); ok {
		e.config = errorConfig
		e.enabled = errorConfig.Enabled
		return nil
	}
	return fmt.Errorf("invalid error config type")
}

// Handler returns the HTTP middleware handler
func (e *ErrorMiddleware) Handler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !e.enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Wrap with panic recovery
			defer e.recoverPanic(w, r)

			// Create error-aware response writer
			ew := &ErrorResponseWriter{
				ResponseWriter: w,
				request:        r,
				errorHandler:   e,
			}

			next.ServeHTTP(ew, r)
		})
	}
}

// ErrorResponseWriter wraps http.ResponseWriter to capture errors
type ErrorResponseWriter struct {
	http.ResponseWriter
	request      *http.Request
	errorHandler *ErrorMiddleware
	statusCode   int
	written      bool
}

// WriteHeader captures the status code and handles errors
func (ew *ErrorResponseWriter) WriteHeader(statusCode int) {
	ew.statusCode = statusCode
	
	// Handle error status codes
	if statusCode >= 400 && !ew.written {
		ew.handleErrorStatus(statusCode)
	}
	
	ew.ResponseWriter.WriteHeader(statusCode)
	ew.written = true
}

// Write captures response writing
func (ew *ErrorResponseWriter) Write(data []byte) (int, error) {
	if ew.statusCode == 0 {
		ew.statusCode = http.StatusOK
	}
	return ew.ResponseWriter.Write(data)
}

// handleErrorStatus handles error status codes
func (ew *ErrorResponseWriter) handleErrorStatus(statusCode int) {
	if !ew.errorHandler.config.LogErrors {
		return
	}

	errorCtx := ew.errorHandler.createErrorContext(ew.request, statusCode)
	
	// Log the error
	ew.errorHandler.logError(errorCtx, fmt.Sprintf("HTTP %d error", statusCode))
}

// recoverPanic recovers from panics and handles them gracefully
func (e *ErrorMiddleware) recoverPanic(w http.ResponseWriter, r *http.Request) {
	if recovered := recover(); recovered != nil {
		// Create panic recovery info
		panicInfo := &PanicRecovery{
			Value: recovered,
			Stack: e.captureStack(),
		}

		// Create error context
		errorCtx := e.createErrorContext(r, http.StatusInternalServerError)
		errorCtx.Metadata = map[string]interface{}{
			"panic": panicInfo,
		}

		// Log the panic
		e.logError(errorCtx, fmt.Sprintf("Panic recovered: %v", recovered))

		// Write error response
		e.writeErrorResponse(w, r, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Internal server error")
	}
}

// createErrorContext creates an error context from the request
func (e *ErrorMiddleware) createErrorContext(r *http.Request, statusCode int) *ErrorContext {
	errorCtx := &ErrorContext{
		Path:       r.URL.Path,
		Method:     r.Method,
		UserAgent:  r.Header.Get("User-Agent"),
		RemoteAddr: r.RemoteAddr,
		Timestamp:  time.Now(),
		Metadata:   make(map[string]interface{}),
	}

	// Add request ID if available
	if reqCtx := middleware.GetRequestContext(r.Context()); reqCtx != nil {
		errorCtx.RequestID = reqCtx.RequestID
		errorCtx.UserID = reqCtx.UserID
	} else {
		errorCtx.RequestID = e.generateRequestID()
	}

	// Add stack trace if configured
	if e.config.IncludeStack {
		errorCtx.Stack = e.captureStack()
	}

	// Add status code to metadata
	errorCtx.Metadata["status_code"] = statusCode

	return errorCtx
}

// logError logs an error with context
func (e *ErrorMiddleware) logError(errorCtx *ErrorContext, message string) {
	if !e.config.LogErrors {
		return
	}

	// Sanitize sensitive information if configured
	if e.config.SanitizeSecrets {
		message = e.sanitizeMessage(message)
	}

	// Create log attributes
	attrs := []slog.Attr{
		slog.String("request_id", errorCtx.RequestID),
		slog.String("path", errorCtx.Path),
		slog.String("method", errorCtx.Method),
		slog.Time("timestamp", errorCtx.Timestamp),
	}

	if errorCtx.UserID != "" {
		attrs = append(attrs, slog.String("user_id", errorCtx.UserID))
	}

	if errorCtx.UserAgent != "" {
		attrs = append(attrs, slog.String("user_agent", errorCtx.UserAgent))
	}

	if errorCtx.RemoteAddr != "" {
		attrs = append(attrs, slog.String("remote_addr", errorCtx.RemoteAddr))
	}

	// Add metadata
	for key, value := range errorCtx.Metadata {
		attrs = append(attrs, slog.Any(key, value))
	}

	// Add stack trace if available
	if len(errorCtx.Stack) > 0 {
		attrs = append(attrs, slog.Any("stack", errorCtx.Stack))
	}

	// Log the error
	e.logger.Error(message, attrs...)
}

// captureStack captures the current stack trace
func (e *ErrorMiddleware) captureStack() []string {
	var stack []string
	
	// Skip the first few frames (this function, recover, etc.)
	for i := 3; i < 20; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		
		// Format stack frame
		frame := fmt.Sprintf("%s:%d %s", file, line, fn.Name())
		stack = append(stack, frame)
	}
	
	return stack
}

// sanitizeMessage removes sensitive information from error messages
func (e *ErrorMiddleware) sanitizeMessage(message string) string {
	// List of sensitive patterns to redact
	sensitivePatterns := []string{
		"password",
		"token",
		"key",
		"secret",
		"auth",
		"credential",
	}

	sanitized := message
	for _, pattern := range sensitivePatterns {
		if strings.Contains(strings.ToLower(sanitized), pattern) {
			// Replace with redacted message
			sanitized = strings.ReplaceAll(sanitized, pattern, "[REDACTED]")
		}
	}

	return sanitized
}

// writeErrorResponse writes a standardized error response
func (e *ErrorMiddleware) writeErrorResponse(w http.ResponseWriter, r *http.Request, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	// Sanitize error message if configured
	if e.config.SanitizeSecrets {
		message = e.sanitizeMessage(message)
	}

	// Create error response
	errorInfo := &middleware.ErrorInfo{
		Code:    code,
		Message: message,
	}

	// Add stack trace if configured and detailed errors are enabled
	if e.config.DetailedErrors && e.config.IncludeStack {
		errorInfo.Stack = strings.Join(e.captureStack(), "\n")
	}

	response := middleware.StandardResponse{
		Success:   false,
		Error:     errorInfo,
		Timestamp: time.Now(),
	}

	// Add request ID if available
	if reqCtx := middleware.GetRequestContext(r.Context()); reqCtx != nil {
		response.RequestID = reqCtx.RequestID
	} else {
		response.RequestID = e.generateRequestID()
	}

	// Encode and write response
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(response); err != nil {
		// Fallback error response
		w.Write([]byte(`{"success":false,"error":{"code":"ENCODING_ERROR","message":"Failed to encode error response"}}`))
	}
}

// HandleError handles an error and writes an appropriate response
func (e *ErrorMiddleware) HandleError(w http.ResponseWriter, r *http.Request, err error, statusCode int) {
	if err == nil {
		return
	}

	// Create error context
	errorCtx := e.createErrorContext(r, statusCode)
	
	// Log the error
	e.logError(errorCtx, err.Error())

	// Determine error code based on status
	var code string
	switch statusCode {
	case http.StatusBadRequest:
		code = "BAD_REQUEST"
	case http.StatusUnauthorized:
		code = "UNAUTHORIZED"
	case http.StatusForbidden:
		code = "FORBIDDEN"
	case http.StatusNotFound:
		code = "NOT_FOUND"
	case http.StatusMethodNotAllowed:
		code = "METHOD_NOT_ALLOWED"
	case http.StatusConflict:
		code = "CONFLICT"
	case http.StatusTooManyRequests:
		code = "RATE_LIMITED"
	case http.StatusInternalServerError:
		code = "INTERNAL_SERVER_ERROR"
	case http.StatusBadGateway:
		code = "BAD_GATEWAY"
	case http.StatusServiceUnavailable:
		code = "SERVICE_UNAVAILABLE"
	case http.StatusGatewayTimeout:
		code = "GATEWAY_TIMEOUT"
	default:
		code = "UNKNOWN_ERROR"
	}

	// Write error response
	e.writeErrorResponse(w, r, statusCode, code, err.Error())
}

// generateRequestID generates a unique request ID
func (e *ErrorMiddleware) generateRequestID() string {
	return fmt.Sprintf("err_%d", time.Now().UnixNano())
}

// LogError logs an error with additional context
func (e *ErrorMiddleware) LogError(ctx context.Context, err error, message string, attrs ...slog.Attr) {
	if !e.config.LogErrors || err == nil {
		return
	}

	// Sanitize message if configured
	if e.config.SanitizeSecrets {
		message = e.sanitizeMessage(message)
	}

	// Add error to attributes
	allAttrs := append(attrs, slog.String("error", err.Error()))

	// Add request context if available
	if reqCtx := middleware.GetRequestContext(ctx); reqCtx != nil {
		allAttrs = append(allAttrs, 
			slog.String("request_id", reqCtx.RequestID),
			slog.Time("timestamp", reqCtx.Timestamp),
		)
		
		if reqCtx.UserID != "" {
			allAttrs = append(allAttrs, slog.String("user_id", reqCtx.UserID))
		}
	}

	e.logger.Error(message, allAttrs...)
}

