# AgentAPI Unified Middleware System

This directory contains the consolidated API middleware integration system for AgentAPI, implementing the requirements from Linear issue ZAM-783.

## Overview

The unified middleware system consolidates multiple middleware components into a single, cohesive API middleware layer that provides:

- **Authentication & Security** - JWT and API key authentication with configurable security policies
- **Request Validation** - Comprehensive input validation and sanitization
- **Response Formatting** - Standardized API response formats with compression and caching
- **Real-time Synchronization** - WebSocket and SSE support for live updates
- **Claude Code Integration** - Specialized middleware for Claude Code communication protocols
- **Error Handling** - Centralized error processing with logging and sanitization

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Middleware Manager                        │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │    Auth     │  │ Validation  │  │  Response   │         │
│  │ Middleware  │  │ Middleware  │  │ Middleware  │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
│                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │    Sync     │  │   Claude    │  │   Errors    │         │
│  │ Middleware  │  │ Middleware  │  │ Middleware  │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
└─────────────────────────────────────────────────────────────┘
```

## Components

### 1. Authentication Middleware (`auth/`)

Provides secure authentication and authorization:

- **JWT Token Authentication** - Bearer token validation with configurable expiration
- **API Key Authentication** - Simple API key-based authentication
- **User Context Management** - Automatic user context injection into requests
- **Token Generation** - JWT and API key generation utilities

**Configuration:**
```go
AuthConfig{
    Enabled:    true,
    Type:       "bearer", // or "api_key"
    Secret:     "your-secret-key",
    Expiration: 24 * time.Hour,
    Issuer:     "agentapi",
}
```

### 2. Validation Middleware (`validation/`)

Ensures request integrity and security:

- **Request Size Validation** - Configurable maximum request size limits
- **Content Type Validation** - Ensures proper content types for different endpoints
- **JSON Schema Validation** - Validates JSON request bodies
- **Header Validation** - Validates required and optional headers
- **Strict Mode** - Enhanced validation for production environments

**Configuration:**
```go
ValidationConfig{
    Enabled:         true,
    StrictMode:      false,
    MaxRequestSize:  10 * 1024 * 1024, // 10MB
    ValidateHeaders: true,
}
```

### 3. Response Middleware (`response/`)

Standardizes API responses:

- **Standard Response Format** - Consistent JSON response structure
- **Compression** - Gzip compression with configurable levels
- **Caching Headers** - Automatic cache control headers
- **Request ID Tracking** - Unique request ID generation and tracking
- **Processing Time Headers** - Response time measurement

**Configuration:**
```go
ResponseConfig{
    Enabled:           true,
    StandardFormat:    true,
    IncludeTimestamp:  true,
    IncludeRequestID:  true,
    CompressionLevel:  6,
    CacheControl:      "no-cache",
}
```

### 4. Sync Middleware (`sync/`)

Enables real-time communication:

- **WebSocket Support** - Full-duplex real-time communication
- **Server-Sent Events (SSE)** - One-way real-time updates
- **Client Management** - Connection tracking and lifecycle management
- **Message Broadcasting** - Targeted and broadcast message delivery
- **Heartbeat Mechanism** - Connection health monitoring

**Configuration:**
```go
SyncConfig{
    Enabled:           true,
    WebSocketEnabled:  true,
    SSEEnabled:        true,
    HeartbeatInterval: 30 * time.Second,
    BufferSize:        1000,
}
```

### 5. Claude Middleware (`claude/`)

Specialized Claude Code integration:

- **Session Management** - Claude Code session lifecycle management
- **Request Forwarding** - Intelligent request routing to Claude Code
- **Protocol Translation** - AgentAPI to Claude Code protocol conversion
- **Retry Logic** - Configurable retry mechanisms for reliability
- **Rate Limiting** - Built-in rate limiting for Claude Code requests

**Configuration:**
```go
ClaudeConfig{
    Enabled:         true,
    APIEndpoint:     "http://localhost:8080/api/claude",
    Version:         "1.0",
    MaxRetries:      3,
    TimeoutSeconds:  30,
    RateLimitRPS:    10,
    EnableStreaming: true,
}
```

### 6. Error Middleware (`errors/`)

Comprehensive error handling:

- **Panic Recovery** - Graceful panic recovery with stack traces
- **Error Logging** - Structured error logging with context
- **Secret Sanitization** - Automatic removal of sensitive information
- **Standard Error Responses** - Consistent error response formatting
- **Stack Trace Management** - Configurable stack trace inclusion

**Configuration:**
```go
ErrorConfig{
    Enabled:         true,
    DetailedErrors:  false,
    LogErrors:       true,
    IncludeStack:    false,
    SanitizeSecrets: true,
}
```

## Usage

### Basic Setup

```go
package main

import (
    "context"
    "log/slog"
    
    "github.com/coder/agentapi/lib/middleware"
)

func main() {
    // Create default configuration
    config := middleware.CreateDefaultConfig()
    
    // Create logger
    logger := slog.Default()
    
    // Create enhanced server with unified middleware
    server, err := middleware.CreateEnhancedServer(
        context.Background(),
        3284, // port
        config,
        logger,
    )
    if err != nil {
        panic(err)
    }
    
    // Start server
    if err := server.Start(); err != nil {
        panic(err)
    }
}
```

### Integration with Existing Server

```go
// Integrate with existing AgentAPI server
manager, err := middleware.IntegrateWithServer(existingServer, config, logger)
if err != nil {
    return err
}

// Apply middleware to router
manager.ApplyToRouter(router)

// Start middleware background processes
if err := manager.Start(ctx); err != nil {
    return err
}
```

### Custom Configuration

```go
config := middleware.MiddlewareConfig{
    Auth: middleware.AuthConfig{
        Enabled:    true,
        Type:       "bearer",
        Secret:     os.Getenv("JWT_SECRET"),
        Expiration: 24 * time.Hour,
    },
    Validation: middleware.ValidationConfig{
        Enabled:         true,
        StrictMode:      true,
        MaxRequestSize:  5 * 1024 * 1024, // 5MB
        ValidateHeaders: true,
    },
    // ... other configurations
}
```

## API Endpoints

The unified middleware system exposes several management endpoints:

- `GET /health` - Health check with middleware status
- `GET /middleware/status` - Detailed middleware component status
- `GET /middleware/config` - Current middleware configuration
- `PUT /middleware/config` - Update middleware configuration
- `POST /auth/login` - Authentication endpoint
- `POST /auth/logout` - Logout endpoint
- `POST /auth/refresh` - Token refresh endpoint
- `GET /ws` - WebSocket endpoint for real-time communication
- `GET /events` - Server-Sent Events endpoint

## Real-time Features

### WebSocket Communication

```javascript
const ws = new WebSocket('ws://localhost:3284/ws');

ws.onopen = function() {
    // Subscribe to agent updates
    ws.send(JSON.stringify({
        type: 'subscribe',
        data: { agent_id: 'agent-123' }
    }));
};

ws.onmessage = function(event) {
    const message = JSON.parse(event.data);
    console.log('Received:', message);
};
```

### Server-Sent Events

```javascript
const eventSource = new EventSource('http://localhost:3284/events');

eventSource.onmessage = function(event) {
    const data = JSON.parse(event.data);
    console.log('Event:', data);
};
```

## Security Features

- **JWT Authentication** with configurable expiration
- **API Key Management** with user mapping
- **Request Sanitization** to prevent injection attacks
- **Secret Redaction** in logs and error messages
- **CORS Protection** with configurable origins
- **Rate Limiting** for Claude Code integration
- **Input Validation** with strict mode support

## Monitoring and Observability

- **Structured Logging** with request context
- **Request ID Tracking** across all components
- **Processing Time Measurement** for performance monitoring
- **Connection Metrics** for real-time features
- **Error Rate Tracking** with detailed error information
- **Health Check Endpoints** for service monitoring

## Configuration Management

The middleware system supports:

- **Environment-based Configuration** via environment variables
- **Runtime Configuration Updates** via API endpoints
- **Configuration Validation** to prevent invalid settings
- **Default Configuration** for quick setup
- **Per-component Configuration** for fine-grained control

## Error Handling

Comprehensive error handling includes:

- **Panic Recovery** with graceful degradation
- **Structured Error Responses** with consistent formatting
- **Error Context Preservation** across middleware layers
- **Secret Sanitization** in error messages
- **Stack Trace Management** for debugging
- **Error Rate Limiting** to prevent spam

## Performance Considerations

- **Middleware Ordering** optimized for performance
- **Connection Pooling** for external services
- **Response Compression** to reduce bandwidth
- **Efficient JSON Processing** with streaming where possible
- **Memory Management** with configurable buffer sizes
- **Background Cleanup** for resource management

## Testing

The middleware system includes:

- **Unit Tests** for individual components
- **Integration Tests** for middleware chains
- **Performance Tests** for load validation
- **Security Tests** for vulnerability assessment
- **Mock Implementations** for testing isolation

## Contributing

When adding new middleware components:

1. Implement the `Middleware` interface
2. Add configuration to `MiddlewareConfig`
3. Update the `Manager` to include the new component
4. Add appropriate tests
5. Update documentation

## License

This middleware system is part of the AgentAPI project and follows the same licensing terms.

