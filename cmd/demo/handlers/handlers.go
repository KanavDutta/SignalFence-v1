package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

// Response is a generic JSON response structure
type Response struct {
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// Health returns a health check endpoint
func Health(w http.ResponseWriter, r *http.Request) {
	resp := Response{
		Message:   "SignalFence demo server is healthy",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Search handles search requests (lenient rate limit)
func Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		query = "all"
	}

	resp := Response{
		Message: "Search endpoint - lenient rate limit (100 req/min)",
		Data: map[string]interface{}{
			"query":   query,
			"results": []string{"result1", "result2", "result3"},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Create handles resource creation (moderate rate limit)
func Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := Response{
		Message: "Create endpoint - moderate rate limit (20 req/min)",
		Data: map[string]interface{}{
			"id":      "12345",
			"created": true,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// Login handles authentication (strict rate limit to prevent brute force)
func Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := Response{
		Message: "Login endpoint - strict rate limit (5 req/min)",
		Data: map[string]interface{}{
			"token": "mock-jwt-token",
			"user":  "demo-user",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Update handles resource updates (moderate rate limit)
func Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != "PATCH" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := Response{
		Message: "Update endpoint - moderate rate limit (30 req/min)",
		Data: map[string]interface{}{
			"id":      "12345",
			"updated": true,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
