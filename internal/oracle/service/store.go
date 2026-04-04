package service

import (
	"context"
	"encoding/json"
)

type MarketStore interface {
	ListEligibleMarkets(ctx context.Context, now int64, limit int) ([]EligibleMarket, error)
	UpsertResolution(ctx context.Context, update ResolutionUpdate) error
}

type EligibleMarket struct {
	MarketID         int64
	ResolveAt        int64
	MarketStatus     string
	CategoryKey      string
	Metadata         json.RawMessage
	OptionSchema     json.RawMessage
	ResolutionStatus string
	ResolvedOutcome  string
	ResolverType     string
	ResolverRef      string
	Evidence         json.RawMessage
}

type ResolutionUpdate struct {
	MarketID        int64
	Status          string
	ResolvedOutcome string
	ResolverType    string
	ResolverRef     string
	Evidence        json.RawMessage
}
