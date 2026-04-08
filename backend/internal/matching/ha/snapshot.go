package ha

import (
	"funnyoption/internal/matching/model"
)

// BookSnapshotData represents the complete state of a single order book
// for HA recovery (primary→standby handoff).
type BookSnapshotData struct {
	BookKey  string         `json:"book_key"`
	LocalSeq uint64        `json:"local_seq"`
	Orders   []*model.Order `json:"orders"`
}

// FullSnapshot captures the entire matching engine state for standby recovery.
type FullSnapshot struct {
	EpochID        uint64             `json:"epoch_id"`
	GlobalSequence uint64             `json:"global_sequence"`
	Books          []BookSnapshotData `json:"books"`
}

// BookSnapshotProvider can generate full snapshots of the engine state.
type BookSnapshotProvider interface {
	TakeSnapshot() FullSnapshot
}
