package store

import "github.com/yourusername/signalfence/core"

// Store defines the interface for bucket state storage
type Store interface {
	Get(key string) *core.BucketState
	Set(key string, state *core.BucketState)
	Delete(key string)
	Clear()
}
