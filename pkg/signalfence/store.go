package signalfence

import (
	"fmt"
	"sync"
	"time"
)

// Store defines the interface for bucket storage implementations.
// This allows for different backends (in-memory, Redis, database, etc.)
type Store interface {
	// GetBucket retrieves a bucket for the given key.
	// If the bucket doesn't exist, it creates a new one with the default config.
	GetBucket(key string) (*Bucket, error)

	// Cleanup removes expired or idle buckets to prevent memory leaks.
	// Returns the number of buckets removed.
	Cleanup() (int, error)

	// Count returns the total number of buckets in the store.
	Count() int
}

// BucketConfig holds the configuration for creating new buckets.
type BucketConfig struct {
	Capacity   int64   // Maximum tokens (burst size)
	RefillRate float64 // Tokens added per second
}

// InMemoryStore implements Store using an in-memory map.
// It's thread-safe and suitable for single-instance deployments.
type InMemoryStore struct {
	buckets     map[string]*bucketEntry
	config      BucketConfig
	mu          sync.RWMutex
	cleanupAge  time.Duration // Buckets idle longer than this are cleaned up
	lastCleanup time.Time
}

// bucketEntry wraps a bucket with metadata for cleanup.
type bucketEntry struct {
	bucket       *Bucket
	lastAccessed time.Time
	mu           sync.Mutex // Protects lastAccessed
}

// NewInMemoryStore creates a new in-memory store with the given bucket configuration.
// cleanupAge determines how long idle buckets are kept before cleanup (0 = no cleanup).
func NewInMemoryStore(config BucketConfig, cleanupAge time.Duration) (*InMemoryStore, error) {
	if config.Capacity <= 0 {
		return nil, ErrNegativeCapacity
	}
	if config.RefillRate <= 0 {
		return nil, ErrNegativeRefillRate
	}

	return &InMemoryStore{
		buckets:     make(map[string]*bucketEntry),
		config:      config,
		cleanupAge:  cleanupAge,
		lastCleanup: time.Now(),
	}, nil
}

// GetBucket retrieves or creates a bucket for the given key.
// This method is thread-safe.
func (s *InMemoryStore) GetBucket(key string) (*Bucket, error) {
	if key == "" {
		return nil, ErrInvalidKey
	}

	// Try read lock first (fast path - bucket exists)
	s.mu.RLock()
	entry, exists := s.buckets[key]
	s.mu.RUnlock()

	if exists {
		// Update last accessed time
		entry.mu.Lock()
		entry.lastAccessed = time.Now()
		entry.mu.Unlock()
		return entry.bucket, nil
	}

	// Bucket doesn't exist, acquire write lock to create it
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check: another goroutine might have created it
	entry, exists = s.buckets[key]
	if exists {
		entry.mu.Lock()
		entry.lastAccessed = time.Now()
		entry.mu.Unlock()
		return entry.bucket, nil
	}

	// Create new bucket
	bucket, err := NewBucket(s.config.Capacity, s.config.RefillRate)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create bucket: %v", ErrStoreFailed, err)
	}

	s.buckets[key] = &bucketEntry{
		bucket:       bucket,
		lastAccessed: time.Now(),
	}

	return bucket, nil
}

// Cleanup removes buckets that haven't been accessed recently.
// Returns the number of buckets removed.
func (s *InMemoryStore) Cleanup() (int, error) {
	if s.cleanupAge == 0 {
		return 0, nil // Cleanup disabled
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-s.cleanupAge)
	removed := 0

	for key, entry := range s.buckets {
		entry.mu.Lock()
		lastAccessed := entry.lastAccessed
		entry.mu.Unlock()

		if lastAccessed.Before(cutoff) {
			delete(s.buckets, key)
			removed++
		}
	}

	s.lastCleanup = now
	return removed, nil
}

// Count returns the total number of buckets in the store.
func (s *InMemoryStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.buckets)
}

// StartBackgroundCleanup starts a goroutine that periodically cleans up idle buckets.
// Call the returned function to stop the cleanup goroutine.
func (s *InMemoryStore) StartBackgroundCleanup(interval time.Duration) func() {
	if s.cleanupAge == 0 || interval == 0 {
		// Return no-op function if cleanup is disabled
		return func() {}
	}

	ticker := time.NewTicker(interval)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				s.Cleanup()
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()

	return func() {
		close(done)
	}
}
