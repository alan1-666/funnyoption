package service

import (
	"encoding/json"
	"testing"
)

func TestFinalizeResolutionRecordPreservesObservedOracleOwnership(t *testing.T) {
	current := resolutionRecord{
		Status:          "OBSERVED",
		ResolvedOutcome: "YES",
		ResolverType:    "ORACLE_PRICE",
		ResolverRef:     "oracle_price:BINANCE:BTCUSDT:1775886400",
		Evidence:        json.RawMessage(`{"version":1,"retry":{"attempt_count":1}}`),
	}

	final := finalizeResolutionRecord(current, "YES")

	if final.ResolverType != "ORACLE_PRICE" {
		t.Fatalf("expected oracle ownership to be preserved, got %s", final.ResolverType)
	}
	if final.ResolverRef != current.ResolverRef {
		t.Fatalf("expected resolver_ref %s, got %s", current.ResolverRef, final.ResolverRef)
	}
	if string(final.Evidence) != string(current.Evidence) {
		t.Fatalf("expected oracle evidence to be preserved, got %s", string(final.Evidence))
	}
}

func TestFinalizeResolutionRecordOverwritesErrorStateWithAdminOwnership(t *testing.T) {
	current := resolutionRecord{
		Status:          "RETRYABLE_ERROR",
		ResolvedOutcome: "",
		ResolverType:    "ORACLE_PRICE",
		ResolverRef:     "oracle_price:BINANCE:BTCUSDT:1775886400",
		Evidence:        json.RawMessage(`{"retry":{"attempt_count":3,"last_error_code":"SOURCE_UNAVAILABLE"}}`),
	}

	final := finalizeResolutionRecord(current, "NO")

	if final.ResolverType != "ADMIN" {
		t.Fatalf("expected manual fallback to overwrite resolver_type, got %s", final.ResolverType)
	}
	if final.ResolverRef != "" {
		t.Fatalf("expected resolver_ref to be cleared, got %s", final.ResolverRef)
	}
	if string(final.Evidence) != "{}" {
		t.Fatalf("expected stale oracle evidence to be cleared, got %s", string(final.Evidence))
	}
}
