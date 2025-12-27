package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/yourusername/signalfence/core"
	"github.com/yourusername/signalfence/store"
)

// Handler handles rate limit check requests
type Handler struct {
	bucket        *core.TokenBucket
	store         store.Store
	defaultPolicy core.Config
	metrics       MetricsRecorder
}

// MetricsRecorder defines the interface for recording metrics
type MetricsRecorder interface {
	RecordRequest(clientID string, allowed bool)
}

// NewHandler creates a new API handler
func NewHandler(store store.Store, defaultPolicy core.Config, metrics MetricsRecorder) *Handler {
	return &Handler{
		bucket:        core.NewTokenBucket(defaultPolicy),
		store:         store,
		defaultPolicy: defaultPolicy,
		metrics:       metrics,
	}
}

// CheckRequest represents the incoming rate limit check request
type CheckRequest struct {
	ClientID string  `json:"client_id"`           // Required: unique identifier (user ID, API key, IP)
	Capacity *float64 `json:"capacity,omitempty"` // Optional: override default capacity
	RefillPerSec *float64 `json:"refill_per_sec,omitempty"` // Optional: override default refill rate
}

// CheckResponse represents the rate limit check response
type CheckResponse struct {
	Allowed      bool    `json:"allowed"`                 // Whether request is allowed
	Remaining    float64 `json:"remaining"`               // Tokens remaining
	Limit        float64 `json:"limit"`                   // Total capacity
	RetryAfterMs int64   `json:"retry_after_ms,omitempty"` // Milliseconds until retry (if blocked)
	ResetAt      int64   `json:"reset_at"`                // Unix timestamp when bucket is full
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// CheckRateLimit handles POST /check requests
func (h *Handler) CheckRateLimit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST requests are allowed")
		return
	}

	// Parse request
	var req CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	// Validate client_id
	if req.ClientID == "" {
		h.sendError(w, http.StatusBadRequest, "missing_client_id", "client_id is required")
		return
	}

	// Use custom policy if provided, otherwise use default
	policy := h.defaultPolicy
	if req.Capacity != nil {
		policy.Capacity = *req.Capacity
	}
	if req.RefillPerSec != nil {
		policy.RefillPerSec = *req.RefillPerSec
	}

	// Create bucket with policy (might be custom)
	bucket := core.NewTokenBucket(policy)

	// Get current state
	state := h.store.Get(req.ClientID)

	// Check rate limit
	newState, result := bucket.Check(state, time.Now())

	// Update state
	h.store.Set(req.ClientID, newState)

	// Record metrics
	if h.metrics != nil {
		h.metrics.RecordRequest(req.ClientID, result.Allowed)
	}

	// Calculate reset time (when bucket will be full)
	tokensNeeded := policy.Capacity - newState.Tokens
	secondsToFull := tokensNeeded / policy.RefillPerSec
	resetAt := time.Now().Add(time.Duration(secondsToFull * float64(time.Second))).Unix()

	// Build response
	response := CheckResponse{
		Allowed:      result.Allowed,
		Remaining:    result.Remaining,
		Limit:        result.Limit,
		RetryAfterMs: result.RetryAfterMs,
		ResetAt:      resetAt,
	}

	// Set status code
	statusCode := http.StatusOK
	if !result.Allowed {
		statusCode = http.StatusTooManyRequests
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) sendError(w http.ResponseWriter, statusCode int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   errorCode,
		Message: message,
	})
}
