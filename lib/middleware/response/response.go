package response

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/coder/agentapi/lib/middleware"
)

// ResponseMiddleware handles response formatting for AgentAPI
type ResponseMiddleware struct {
	config  middleware.ResponseConfig
	enabled bool
}

// ResponseWriter wraps http.ResponseWriter to capture response data
type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
	startTime  time.Time
}

// NewResponseWriter creates a new ResponseWriter
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:          &bytes.Buffer{},
		startTime:     time.Now(),
	}
}

// WriteHeader captures the status code
func (rw *ResponseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// Write captures the response body
func (rw *ResponseWriter) Write(data []byte) (int, error) {
	rw.body.Write(data)
	return rw.ResponseWriter.Write(data)
}

// NewResponseMiddleware creates a new response middleware instance
func NewResponseMiddleware(config middleware.ResponseConfig) *ResponseMiddleware {
	return &ResponseMiddleware{
		config:  config,
		enabled: config.Enabled,
	}
}

// Name returns the middleware name
func (r *ResponseMiddleware) Name() string {
	return "response"
}

// IsEnabled returns whether the middleware is enabled
func (r *ResponseMiddleware) IsEnabled() bool {
	return r.enabled
}

// Configure configures the middleware with the provided config
func (r *ResponseMiddleware) Configure(config interface{}) error {
	if responseConfig, ok := config.(middleware.ResponseConfig); ok {
		r.config = responseConfig
		r.enabled = responseConfig.Enabled
		return nil
	}
	return fmt.Errorf("invalid response config type")
}

// Handler returns the HTTP middleware handler
func (r *ResponseMiddleware) Handler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if !r.enabled {
				next.ServeHTTP(w, req)
				return
			}

			// Set standard headers
			r.setStandardHeaders(w, req)

			// Wrap the response writer
			rw := NewResponseWriter(w)

			// Process the request
			next.ServeHTTP(rw, req)

			// Post-process the response if needed
			r.postProcessResponse(rw, req)
		})
	}
}

// setStandardHeaders sets standard response headers
func (r *ResponseMiddleware) setStandardHeaders(w http.ResponseWriter, req *http.Request) {
	// Set cache control
	if r.config.CacheControl != "" {
		w.Header().Set("Cache-Control", r.config.CacheControl)
	}

	// Set compression headers if supported
	if r.supportsCompression(req) && r.config.CompressionLevel > 0 {
		w.Header().Set("Content-Encoding", "gzip")
	}

	// Set standard API headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	
	// Set API version header
	w.Header().Set("X-API-Version", "1.0")
	
	// Set request ID header if available
	if reqCtx := middleware.GetRequestContext(req.Context()); reqCtx != nil {
		w.Header().Set("X-Request-ID", reqCtx.RequestID)
	}
}

// postProcessResponse handles post-processing of responses
func (r *ResponseMiddleware) postProcessResponse(rw *ResponseWriter, req *http.Request) {
	// Add processing time header
	processingTime := time.Since(rw.startTime)
	rw.Header().Set("X-Processing-Time", processingTime.String())

	// Handle compression if needed
	if r.supportsCompression(req) && r.config.CompressionLevel > 0 {
		r.compressResponse(rw)
	}
}

// supportsCompression checks if the client supports compression
func (r *ResponseMiddleware) supportsCompression(req *http.Request) bool {
	acceptEncoding := req.Header.Get("Accept-Encoding")
	return strings.Contains(acceptEncoding, "gzip")
}

// compressResponse compresses the response body
func (r *ResponseMiddleware) compressResponse(rw *ResponseWriter) {
	if rw.body.Len() == 0 {
		return
	}

	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, r.config.CompressionLevel)
	if err != nil {
		return
	}

	if _, err := gz.Write(rw.body.Bytes()); err != nil {
		gz.Close()
		return
	}

	if err := gz.Close(); err != nil {
		return
	}

	// Update content length
	rw.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
}

// FormatStandardResponse formats a response according to the standard format
func (r *ResponseMiddleware) FormatStandardResponse(ctx context.Context, data interface{}, err error) middleware.StandardResponse {
	response := middleware.StandardResponse{
		Success:   err == nil,
		Data:      data,
		Timestamp: time.Now(),
	}

	// Add request ID if available
	if reqCtx := middleware.GetRequestContext(ctx); reqCtx != nil {
		response.RequestID = reqCtx.RequestID
	} else {
		response.RequestID = r.generateRequestID()
	}

	// Add error information if present
	if err != nil {
		response.Error = &middleware.ErrorInfo{
			Code:    "INTERNAL_ERROR",
			Message: err.Error(),
		}
	}

	// Add metadata if configured
	if r.config.IncludeTimestamp || r.config.IncludeRequestID {
		response.Meta = &middleware.MetaInfo{
			Version: "1.0",
		}
	}

	return response
}

// WriteJSONResponse writes a JSON response with standard formatting
func (r *ResponseMiddleware) WriteJSONResponse(w http.ResponseWriter, req *http.Request, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	var response interface{}
	
	if r.config.StandardFormat {
		// Use standard response format
		response = r.FormatStandardResponse(req.Context(), data, nil)
	} else {
		// Use raw data
		response = data
	}

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(response); err != nil {
		// Fallback error response
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"success":false,"error":{"code":"ENCODING_ERROR","message":"Failed to encode response"}}`))
	}
}

// WriteErrorResponse writes an error response with standard formatting
func (r *ResponseMiddleware) WriteErrorResponse(w http.ResponseWriter, req *http.Request, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := middleware.StandardResponse{
		Success: false,
		Error: &middleware.ErrorInfo{
			Code:    code,
			Message: message,
		},
		Timestamp: time.Now(),
	}

	// Add request ID if available
	if reqCtx := middleware.GetRequestContext(req.Context()); reqCtx != nil {
		response.RequestID = reqCtx.RequestID
	} else {
		response.RequestID = r.generateRequestID()
	}

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(response); err != nil {
		// Fallback error response
		w.Write([]byte(`{"success":false,"error":{"code":"ENCODING_ERROR","message":"Failed to encode error response"}}`))
	}
}

// WriteSuccessResponse writes a success response with standard formatting
func (r *ResponseMiddleware) WriteSuccessResponse(w http.ResponseWriter, req *http.Request, data interface{}) {
	r.WriteJSONResponse(w, req, http.StatusOK, data)
}

// WriteCreatedResponse writes a created response with standard formatting
func (r *ResponseMiddleware) WriteCreatedResponse(w http.ResponseWriter, req *http.Request, data interface{}) {
	r.WriteJSONResponse(w, req, http.StatusCreated, data)
}

// WriteNoContentResponse writes a no content response
func (r *ResponseMiddleware) WriteNoContentResponse(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

// generateRequestID generates a unique request ID
func (r *ResponseMiddleware) generateRequestID() string {
	return fmt.Sprintf("resp_%d", time.Now().UnixNano())
}

// AddRateLimitHeaders adds rate limiting headers to the response
func (r *ResponseMiddleware) AddRateLimitHeaders(w http.ResponseWriter, limit, remaining, reset int) {
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	w.Header().Set("X-RateLimit-Reset", strconv.Itoa(reset))
}

// AddCORSHeaders adds CORS headers to the response
func (r *ResponseMiddleware) AddCORSHeaders(w http.ResponseWriter, req *http.Request) {
	origin := req.Header.Get("Origin")
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	} else {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}
	
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token, X-API-Key")
	w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID, X-Processing-Time, X-RateLimit-Limit, X-RateLimit-Remaining")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Max-Age", "86400")
}

// HandlePreflight handles CORS preflight requests
func (r *ResponseMiddleware) HandlePreflight(w http.ResponseWriter, req *http.Request) {
	r.AddCORSHeaders(w, req)
	w.WriteHeader(http.StatusNoContent)
}

