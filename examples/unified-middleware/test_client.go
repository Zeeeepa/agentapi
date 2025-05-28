package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// TestClient demonstrates how to interact with the unified middleware system
type TestClient struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewTestClient creates a new test client
func NewTestClient(baseURL string) *TestClient {
	return &TestClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Login authenticates with the server and stores the token
func (tc *TestClient) Login(username, password string) error {
	loginReq := map[string]string{
		"username": username,
		"password": password,
	}

	jsonData, err := json.Marshal(loginReq)
	if err != nil {
		return err
	}

	resp, err := tc.client.Post(tc.baseURL+"/auth/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed: %s", string(body))
	}

	var loginResp struct {
		Success bool `json:"success"`
		Data    struct {
			Token   string  `json:"token"`
			UserID  string  `json:"user_id"`
			ExpiresIn float64 `json:"expires_in"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return err
	}

	if !loginResp.Success {
		return fmt.Errorf("login failed: invalid response")
	}

	tc.token = loginResp.Data.Token
	fmt.Printf("✅ Login successful! Token expires in %.0f seconds\n", loginResp.Data.ExpiresIn)
	return nil
}

// GetHealth checks the server health
func (tc *TestClient) GetHealth() error {
	req, err := http.NewRequest("GET", tc.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	if tc.token != "" {
		req.Header.Set("Authorization", "Bearer "+tc.token)
	}

	resp, err := tc.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var healthResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		return err
	}

	fmt.Printf("✅ Health check: %s\n", healthResp["status"])
	if middleware, ok := healthResp["middleware"].(map[string]interface{}); ok {
		fmt.Printf("   Middleware components:\n")
		for name, status := range middleware {
			if statusMap, ok := status.(map[string]interface{}); ok {
				enabled := statusMap["enabled"]
				fmt.Printf("   - %s: enabled=%v\n", name, enabled)
			}
		}
	}
	return nil
}

// GetMiddlewareStatus gets detailed middleware status
func (tc *TestClient) GetMiddlewareStatus() error {
	req, err := http.NewRequest("GET", tc.baseURL+"/middleware/status", nil)
	if err != nil {
		return err
	}

	if tc.token != "" {
		req.Header.Set("Authorization", "Bearer "+tc.token)
	}

	resp, err := tc.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var statusResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return err
	}

	fmt.Printf("✅ Middleware Status:\n")
	if data, ok := statusResp["data"].(map[string]interface{}); ok {
		for component, details := range data {
			fmt.Printf("   %s: %+v\n", component, details)
		}
	}
	return nil
}

// SendMessage sends a message to an agent
func (tc *TestClient) SendMessage(agentID, message string) error {
	messageReq := map[string]interface{}{
		"content": message,
		"type":    "user",
	}

	jsonData, err := json.Marshal(messageReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", tc.baseURL+"/agents/"+agentID+"/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if tc.token != "" {
		req.Header.Set("Authorization", "Bearer "+tc.token)
	}

	resp, err := tc.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var messageResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&messageResp); err != nil {
		return err
	}

	fmt.Printf("✅ Message sent to agent %s: %s\n", agentID, message)
	return nil
}

// TestWebSocket demonstrates WebSocket functionality
func (tc *TestClient) TestWebSocket() error {
	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + tc.baseURL[4:] + "/ws"

	// Create WebSocket connection
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	headers := http.Header{}
	if tc.token != "" {
		headers.Set("Authorization", "Bearer "+tc.token)
	}

	conn, _, err := dialer.Dial(wsURL, headers)
	if err != nil {
		return fmt.Errorf("WebSocket connection failed: %w", err)
	}
	defer conn.Close()

	fmt.Printf("✅ WebSocket connected\n")

	// Send subscription message
	subscribeMsg := map[string]interface{}{
		"type": "subscribe",
		"data": map[string]string{
			"agent_id": "test-agent-123",
		},
	}

	if err := conn.WriteJSON(subscribeMsg); err != nil {
		return fmt.Errorf("failed to send subscription: %w", err)
	}

	// Read welcome message and a few updates
	for i := 0; i < 3; i++ {
		var message map[string]interface{}
		if err := conn.ReadJSON(&message); err != nil {
			return fmt.Errorf("failed to read WebSocket message: %w", err)
		}

		fmt.Printf("📨 WebSocket message: type=%s\n", message["type"])
		if data, ok := message["data"].(map[string]interface{}); ok {
			fmt.Printf("   Data: %+v\n", data)
		}
	}

	return nil
}

// TestSSE demonstrates Server-Sent Events functionality
func (tc *TestClient) TestSSE() error {
	req, err := http.NewRequest("GET", tc.baseURL+"/events", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	if tc.token != "" {
		req.Header.Set("Authorization", "Bearer "+tc.token)
	}

	resp, err := tc.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SSE connection failed: %s", string(body))
	}

	fmt.Printf("✅ SSE connected\n")

	// Read a few events
	buffer := make([]byte, 1024)
	for i := 0; i < 3; i++ {
		n, err := resp.Body.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read SSE data: %w", err)
		}

		if n > 0 {
			fmt.Printf("📡 SSE data: %s", string(buffer[:n]))
		}

		time.Sleep(1 * time.Second)
	}

	return nil
}

// RunTests runs all test scenarios
func (tc *TestClient) RunTests() {
	fmt.Println("🚀 Starting AgentAPI Unified Middleware Tests")
	fmt.Println("=" * 50)

	// Test 1: Health check without authentication
	fmt.Println("\n1. Testing health check (no auth)...")
	if err := tc.GetHealth(); err != nil {
		fmt.Printf("❌ Health check failed: %v\n", err)
	}

	// Test 2: Login
	fmt.Println("\n2. Testing authentication...")
	if err := tc.Login("demo-user", "demo-password"); err != nil {
		fmt.Printf("❌ Login failed: %v\n", err)
		return
	}

	// Test 3: Health check with authentication
	fmt.Println("\n3. Testing authenticated health check...")
	if err := tc.GetHealth(); err != nil {
		fmt.Printf("❌ Authenticated health check failed: %v\n", err)
	}

	// Test 4: Middleware status
	fmt.Println("\n4. Testing middleware status...")
	if err := tc.GetMiddlewareStatus(); err != nil {
		fmt.Printf("❌ Middleware status failed: %v\n", err)
	}

	// Test 5: Send message (triggers sync middleware)
	fmt.Println("\n5. Testing message sending...")
	if err := tc.SendMessage("test-agent-123", "Hello from test client!"); err != nil {
		fmt.Printf("❌ Message sending failed: %v\n", err)
	}

	// Test 6: WebSocket
	fmt.Println("\n6. Testing WebSocket connection...")
	if err := tc.TestWebSocket(); err != nil {
		fmt.Printf("❌ WebSocket test failed: %v\n", err)
	}

	// Test 7: Server-Sent Events
	fmt.Println("\n7. Testing Server-Sent Events...")
	if err := tc.TestSSE(); err != nil {
		fmt.Printf("❌ SSE test failed: %v\n", err)
	}

	fmt.Println("\n" + "=" * 50)
	fmt.Println("🎉 Test suite completed!")
}

// Example usage
func runTestClient() {
	client := NewTestClient("http://localhost:3284")
	client.RunTests()
}

// Uncomment to run as standalone program
// func main() {
//     runTestClient()
// }

