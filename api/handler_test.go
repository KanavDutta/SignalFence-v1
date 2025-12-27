package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yourusername/signalfence/core"
	"github.com/yourusername/signalfence/store"
)

func TestCheckRateLimit_AllowsRequests(t *testing.T) {
	storage := store.NewMemoryStore()
	policy := core.Config{
		Capacity:     10,
		RefillPerSec: 5,
	}
	handler := NewHandler(storage, policy, nil)

	// Make a request
	reqBody := CheckRequest{ClientID: "test-user"}
	body, _ := json.Marshal(reqBody)
	
	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler.CheckRateLimit(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp CheckResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if !resp.Allowed {
		t.Error("Request should be allowed")
	}

	if resp.Limit != 10 {
		t.Errorf("Limit = %.0f, want 10", resp.Limit)
	}
}

func TestCheckRateLimit_BlocksWhenExceeded(t *testing.T) {
	storage := store.NewMemoryStore()
	policy := core.Config{
		Capacity:     5,
		RefillPerSec: 2,
	}
	handler := NewHandler(storage, policy, nil)

	clientID := "test-user"

	// Drain the bucket
	for i := 0; i < 5; i++ {
		reqBody := CheckRequest{ClientID: clientID}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		handler.CheckRateLimit(w, req)
	}

	// Next request should be blocked
	reqBody := CheckRequest{ClientID: clientID}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler.CheckRateLimit(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}

	var resp CheckResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Allowed {
		t.Error("Request should be blocked")
	}

	if resp.RetryAfterMs <= 0 {
		t.Error("RetryAfterMs should be positive when blocked")
	}
}

func TestCheckRateLimit_RequiresClientID(t *testing.T) {
	storage := store.NewMemoryStore()
	policy := core.Config{Capacity: 10, RefillPerSec: 5}
	handler := NewHandler(storage, policy, nil)

	// Request without client_id
	reqBody := CheckRequest{}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler.CheckRateLimit(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCheckRateLimit_CustomPolicy(t *testing.T) {
	storage := store.NewMemoryStore()
	defaultPolicy := core.Config{Capacity: 10, RefillPerSec: 5}
	handler := NewHandler(storage, defaultPolicy, nil)

	// Request with custom policy
	customCapacity := 20.0
	reqBody := CheckRequest{
		ClientID: "premium-user",
		Capacity: &customCapacity,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler.CheckRateLimit(w, req)

	var resp CheckResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Limit != 20 {
		t.Errorf("Limit = %.0f, want 20 (custom policy)", resp.Limit)
	}
}
