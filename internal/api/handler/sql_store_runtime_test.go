package handler

import (
	"encoding/json"
	"testing"

	"funnyoption/internal/api/dto"
)

func TestMergeMarketMetadataResolvedMarket(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`{"category":"macro","yesOdds":0.57,"noOdds":0.43}`)
	merged := mergeMarketMetadata(raw, nil, "RESOLVED", "YES", dto.MarketRuntime{
		MatchedQuantity: 12,
		MatchedNotional: 732,
		TradeCount:      2,
		LastTradeAt:     1775048079,
	})

	var payload map[string]any
	if err := json.Unmarshal(merged, &payload); err != nil {
		t.Fatalf("unmarshal merged metadata: %v", err)
	}

	if got := payload["yesOdds"]; got != float64(1) {
		t.Fatalf("yesOdds = %v, want 1", got)
	}
	if got := payload["noOdds"]; got != float64(0) {
		t.Fatalf("noOdds = %v, want 0", got)
	}
	if got := payload["volume"]; got != float64(732) {
		t.Fatalf("volume = %v, want 732", got)
	}
}

func TestMergeMarketMetadataUsesLastTradePrice(t *testing.T) {
	t.Parallel()

	merged := mergeMarketMetadata(nil, nil, "OPEN", "", dto.MarketRuntime{
		LastPriceYes:    61,
		MatchedQuantity: 10,
		MatchedNotional: 610,
	})

	var payload map[string]any
	if err := json.Unmarshal(merged, &payload); err != nil {
		t.Fatalf("unmarshal merged metadata: %v", err)
	}

	if got := payload["yesOdds"]; got != 0.61 {
		t.Fatalf("yesOdds = %v, want 0.61", got)
	}
	if got := payload["noOdds"]; got != 0.39 {
		t.Fatalf("noOdds = %v, want 0.39", got)
	}
}

func TestMergeMarketMetadataPreservesExistingCategory(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`{"category":"flow"}`)
	merged := mergeMarketMetadata(raw, nil, "OPEN", "", dto.MarketRuntime{})

	var payload map[string]any
	if err := json.Unmarshal(merged, &payload); err != nil {
		t.Fatalf("unmarshal merged metadata: %v", err)
	}

	if got := payload["category"]; got != "flow" {
		t.Fatalf("category = %v, want flow", got)
	}
}

func TestMergeMarketMetadataAppliesCanonicalCategory(t *testing.T) {
	t.Parallel()

	merged := mergeMarketMetadata(nil, &dto.MarketCategory{
		CategoryID:  1,
		CategoryKey: "SPORTS",
		DisplayName: "体育",
	}, "OPEN", "", dto.MarketRuntime{})

	var payload map[string]any
	if err := json.Unmarshal(merged, &payload); err != nil {
		t.Fatalf("unmarshal merged metadata: %v", err)
	}

	if got := payload["category"]; got != "体育" {
		t.Fatalf("category = %v, want 体育", got)
	}
	if got := payload["categoryKey"]; got != "SPORTS" {
		t.Fatalf("categoryKey = %v, want SPORTS", got)
	}
}
