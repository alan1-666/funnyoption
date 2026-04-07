package dto

import "testing"

func TestNormalizeMarketCategoryKey(t *testing.T) {
	if got := NormalizeMarketCategoryKey("体育", nil); got != "SPORTS" {
		t.Fatalf("expected SPORTS, got %s", got)
	}
	if got := NormalizeMarketCategoryKey("", nil); got != "CRYPTO" {
		t.Fatalf("expected default CRYPTO, got %s", got)
	}
}

func TestNormalizeMarketOptionsDefaultsToBinarySet(t *testing.T) {
	options, err := NormalizeMarketOptions(nil)
	if err != nil {
		t.Fatalf("expected default options, got error: %v", err)
	}
	if len(options) != 2 {
		t.Fatalf("expected 2 default options, got %d", len(options))
	}
	if !IsBinaryTradingOptions(options) {
		t.Fatalf("expected default options to be binary tradable: %+v", options)
	}
}

func TestNormalizeMarketOptionsRejectsDuplicates(t *testing.T) {
	_, err := NormalizeMarketOptions([]MarketOption{
		{Key: "yes", Label: "是"},
		{Key: "YES", Label: "Yes again"},
	})
	if err == nil {
		t.Fatal("expected duplicate option keys to fail validation")
	}
}
