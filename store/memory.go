package store

import (
	"sync"

	"github.com/yourusername/signalfence/core"
)

// MemoryStore provides thread-safe in-memory storage for bucket states
type MemoryStore struct {
	buckets sync.Map // map[string]*core.BucketState
}

// Ensure MemoryStore implements Store interface
var _ Store = (*MemoryStore)(nil)

// NewMemoryStore creates a new in-memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

// Get retrieves the bucket state for a given key
func (s *MemoryStore) Get(key string) *core.BucketState {
	val, ok := s.buckets.Load(key)
	if !ok {
		return nil
	}
	return val.(*core.BucketState)
}

// Set stores the bucket state for a given key
func (s *MemoryStore) Set(key string, state *core.BucketState) {
	s.buckets.Store(key, state)
}

// Delete removes the bucket state for a given key
func (s *MemoryStore) Delete(key string) {
	s.buckets.Delete(key)
}

// Clear removes all bucket states
func (s *MemoryStore) Clear() {
	s.buckets.Range(func(key, value interface{}) bool {
		s.buckets.Delete(key)
		return true
	})
}
