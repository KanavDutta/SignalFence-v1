package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics tracks rate limiting statistics
type Metrics struct {
	totalRequests   atomic.Int64
	allowedRequests atomic.Int64
	blockedRequests atomic.Int64
	
	// Per-client stats
	mu           sync.RWMutex
	clientStats  map[string]*ClientStats
	startTime    time.Time
}

// ClientStats tracks statistics for a specific client
type ClientStats struct {
	ClientID        string
	TotalRequests   int64
	AllowedRequests int64
	BlockedRequests int64
	LastRequestAt   time.Time
	FirstRequestAt  time.Time
}

// NewMetrics creates a new metrics tracker
func NewMetrics() *Metrics {
	return &Metrics{
		clientStats: make(map[string]*ClientStats),
		startTime:   time.Now(),
	}
}

// RecordRequest records a rate limit check
func (m *Metrics) RecordRequest(clientID string, allowed bool) {
	m.totalRequests.Add(1)
	
	if allowed {
		m.allowedRequests.Add(1)
	} else {
		m.blockedRequests.Add(1)
	}
	
	// Update per-client stats
	m.mu.Lock()
	defer m.mu.Unlock()
	
	stats, exists := m.clientStats[clientID]
	if !exists {
		stats = &ClientStats{
			ClientID:      clientID,
			FirstRequestAt: time.Now(),
		}
		m.clientStats[clientID] = stats
	}
	
	stats.TotalRequests++
	if allowed {
		stats.AllowedRequests++
	} else {
		stats.BlockedRequests++
	}
	stats.LastRequestAt = time.Now()
}

// GetSnapshot returns a snapshot of current metrics
func (m *Metrics) GetSnapshot() *Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Copy client stats
	topClients := make([]*ClientStats, 0, len(m.clientStats))
	for _, stats := range m.clientStats {
		topClients = append(topClients, &ClientStats{
			ClientID:        stats.ClientID,
			TotalRequests:   stats.TotalRequests,
			AllowedRequests: stats.AllowedRequests,
			BlockedRequests: stats.BlockedRequests,
			LastRequestAt:   stats.LastRequestAt,
			FirstRequestAt:  stats.FirstRequestAt,
		})
	}
	
	// Sort by total requests (top 10)
	sortByTotalRequests(topClients)
	if len(topClients) > 10 {
		topClients = topClients[:10]
	}
	
	uptime := time.Since(m.startTime)
	
	return &Snapshot{
		TotalRequests:   m.totalRequests.Load(),
		AllowedRequests: m.allowedRequests.Load(),
		BlockedRequests: m.blockedRequests.Load(),
		UniqueClients:   int64(len(m.clientStats)),
		TopClients:      topClients,
		UptimeSeconds:   int64(uptime.Seconds()),
		StartTime:       m.startTime,
	}
}

// Snapshot represents a point-in-time view of metrics
type Snapshot struct {
	TotalRequests   int64          `json:"total_requests"`
	AllowedRequests int64          `json:"allowed_requests"`
	BlockedRequests int64          `json:"blocked_requests"`
	UniqueClients   int64          `json:"unique_clients"`
	TopClients      []*ClientStats `json:"top_clients"`
	UptimeSeconds   int64          `json:"uptime_seconds"`
	StartTime       time.Time      `json:"start_time"`
}

// Helper to sort clients by total requests
func sortByTotalRequests(clients []*ClientStats) {
	for i := 0; i < len(clients)-1; i++ {
		for j := i + 1; j < len(clients); j++ {
			if clients[j].TotalRequests > clients[i].TotalRequests {
				clients[i], clients[j] = clients[j], clients[i]
			}
		}
	}
}
