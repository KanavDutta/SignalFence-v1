package api

import (
	"encoding/json"
	"net/http"

	"github.com/yourusername/signalfence/metrics"
)

// MetricsProvider defines the interface for getting metrics
type MetricsProvider interface {
	GetSnapshot() *metrics.Snapshot
}

// MetricsHandler handles GET /metrics requests
type MetricsHandler struct {
	provider MetricsProvider
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(provider MetricsProvider) *MetricsHandler {
	return &MetricsHandler{provider: provider}
}

// ServeHTTP handles the metrics endpoint
func (h *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	snapshot := h.provider.GetSnapshot()
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*") // Allow dashboard to fetch
	json.NewEncoder(w).Encode(snapshot)
}
