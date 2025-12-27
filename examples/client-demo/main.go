package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// This demonstrates how a client application would use SignalFence

type CheckRequest struct {
	ClientID string `json:"client_id"`
}

type CheckResponse struct {
	Allowed      bool    `json:"allowed"`
	Remaining    float64 `json:"remaining"`
	Limit        float64 `json:"limit"`
	RetryAfterMs int64   `json:"retry_after_ms,omitempty"`
	ResetAt      int64   `json:"reset_at"`
}

func main() {
	signalFenceURL := "http://localhost:8080/check"
	clientID := "user-123" // Could be user ID, API key, IP, etc.

	fmt.Println("🧪 SignalFence Client Demo")
	fmt.Println("Testing rate limiting with client:", clientID)
	fmt.Println()

	// Make requests until we hit the rate limit
	for i := 1; i <= 25; i++ {
		allowed, remaining, retryAfter := checkRateLimit(signalFenceURL, clientID)
		
		if allowed {
			fmt.Printf("✅ Request %2d: ALLOWED  (%.0f tokens remaining)\n", i, remaining)
			// Simulate doing actual work
			handleRequest(i)
		} else {
			fmt.Printf("❌ Request %2d: BLOCKED  (retry after %dms)\n", i, retryAfter)
			// In real app, you'd return 429 to the client
			time.Sleep(time.Duration(retryAfter) * time.Millisecond)
		}

		time.Sleep(50 * time.Millisecond) // Small delay between requests
	}
}

func checkRateLimit(url, clientID string) (allowed bool, remaining float64, retryAfterMs int64) {
	reqBody := CheckRequest{ClientID: clientID}
	jsonData, _ := json.Marshal(reqBody)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error calling SignalFence:", err)
		return false, 0, 1000
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var checkResp CheckResponse
	json.Unmarshal(body, &checkResp)

	return checkResp.Allowed, checkResp.Remaining, checkResp.RetryAfterMs
}

func handleRequest(requestNum int) {
	// Simulate processing the request
	// In a real app, this would be your actual business logic
	time.Sleep(10 * time.Millisecond)
}
