package ha

import (
	"sync/atomic"
)

// EpochManager tracks the current epoch for the matching service instance.
// The epoch is incremented on every leadership transition (rebalance).
// All output messages carry the epoch so downstream consumers can fence
// stale messages from a deposed primary.
type EpochManager struct {
	current atomic.Uint64
}

func NewEpochManager(initial uint64) *EpochManager {
	em := &EpochManager{}
	em.current.Store(initial)
	return em
}

// Current returns the active epoch.
func (em *EpochManager) Current() uint64 {
	return em.current.Load()
}

// Advance increments the epoch and returns the new value.
// Called when this node becomes the primary after a rebalance.
func (em *EpochManager) Advance() uint64 {
	return em.current.Add(1)
}

// Set sets the epoch to a specific value (used during restore from snapshot).
func (em *EpochManager) Set(epoch uint64) {
	em.current.Store(epoch)
}
