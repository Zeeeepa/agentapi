package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coder/agentapi/lib/middleware"
)

func main() {
	// Create logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create custom middleware configuration
	config := middleware.MiddlewareConfig{
		Auth: middleware.AuthConfig{
			Enabled:    true,
			Type:       "bearer",
			Secret:     getEnvOrDefault("JWT_SECRET", "demo-secret-change-in-production"),
			Expiration: 24 * time.Hour,
			Issuer:     "agentapi-demo",
		},
		Validation: middleware.ValidationConfig{
			Enabled:         true,
			StrictMode:      false,
			MaxRequestSize:  10 * 1024 * 1024, // 10MB
			ValidateHeaders: true,
		},
		Response: middleware.ResponseConfig{
			Enabled:           true,
			StandardFormat:    true,
			IncludeTimestamp:  true,
			IncludeRequestID:  true,
			CompressionLevel:  6,
			CacheControl:      "no-cache",
		},
		Sync: middleware.SyncConfig{
			Enabled:           true,
			WebSocketEnabled:  true,
			SSEEnabled:        true,
			HeartbeatInterval: 30 * time.Second,
			BufferSize:        1000,
		},
		Claude: middleware.ClaudeConfig{
			Enabled:         true,
			APIEndpoint:     getEnvOrDefault("CLAUDE_API_ENDPOINT", "http://localhost:8080/api/claude"),
			Version:         "1.0",
			MaxRetries:      3,
			TimeoutSeconds:  30,
			RateLimitRPS:    10,
			EnableStreaming: true,
		},
		Errors: middleware.ErrorConfig{
			Enabled:         true,
			DetailedErrors:  false,
			LogErrors:       true,
			IncludeStack:    false,
			SanitizeSecrets: true,
		},
	}

	// Validate configuration
	if err := middleware.ValidateConfig(config); err != nil {
		logger.Error("Invalid middleware configuration", "error", err)
		os.Exit(1)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create enhanced server with unified middleware
	port := 3284
	server, err := middleware.CreateEnhancedServer(ctx, port, config, logger)
	if err != nil {
		logger.Error("Failed to create enhanced server", "error", err)
		os.Exit(1)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		logger.Info("Starting AgentAPI server with unified middleware", "port", port)
		if err := server.Start(); err != nil {
			logger.Error("Server failed to start", "error", err)
			cancel()
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	logger.Info("Shutdown signal received, stopping server...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop server gracefully
	if err := server.Stop(shutdownCtx); err != nil {
		logger.Error("Error during server shutdown", "error", err)
	} else {
		logger.Info("Server stopped gracefully")
	}
}

// getEnvOrDefault returns the value of an environment variable or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

