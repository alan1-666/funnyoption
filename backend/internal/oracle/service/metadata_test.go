package service

import (
	"encoding/json"
	"testing"
)

func TestParseContractCanonicalizesSupportedOracleMetadata(t *testing.T) {
	raw := json.RawMessage(`{
		"resolution": {
			"version": 1,
			"mode": "oracle_price",
			"market_kind": "crypto_price_threshold",
			"manual_fallback_allowed": true,
			"oracle": {
				"source_kind": "http_json",
				"provider_key": "binance",
				"instrument": {
					"kind": "spot",
					"base_asset": "btc",
					"quote_asset": "usdt",
					"symbol": "btcusdt"
				},
				"price": {
					"field": "last_price",
					"scale": 8,
					"rounding_mode": "round_half_up",
					"max_data_age_sec": 120
				},
				"window": {
					"anchor": "resolve_at",
					"before_sec": 300,
					"after_sec": 300
				},
				"rule": {
					"type": "price_threshold",
					"comparator": "gte",
					"threshold_price": "85000"
				}
			}
		}
	}`)

	contract, isOracle, err := ParseContract("CRYPTO", []string{"YES", "NO"}, 1775886400, raw)
	if err != nil {
		t.Fatalf("ParseContract returned error: %v", err)
	}
	if !isOracle {
		t.Fatalf("expected metadata to be recognized as oracle lane")
	}
	if contract.Metadata.Oracle.ProviderKey != OracleProviderKeyBinance {
		t.Fatalf("expected provider key %s, got %s", OracleProviderKeyBinance, contract.Metadata.Oracle.ProviderKey)
	}
	if contract.Metadata.Oracle.Rule.ThresholdPrice != "85000.00000000" {
		t.Fatalf("expected canonical threshold string, got %s", contract.Metadata.Oracle.Rule.ThresholdPrice)
	}
}

func TestParseContractRejectsUnsupportedProvider(t *testing.T) {
	raw := json.RawMessage(`{
		"resolution": {
			"version": 1,
			"mode": "ORACLE_PRICE",
			"market_kind": "CRYPTO_PRICE_THRESHOLD",
			"manual_fallback_allowed": true,
			"oracle": {
				"source_kind": "HTTP_JSON",
				"provider_key": "COINBASE",
				"instrument": {
					"kind": "SPOT",
					"base_asset": "BTC",
					"quote_asset": "USDT",
					"symbol": "BTCUSDT"
				},
				"price": {
					"field": "LAST_PRICE",
					"scale": 8,
					"rounding_mode": "ROUND_HALF_UP",
					"max_data_age_sec": 120
				},
				"window": {
					"anchor": "RESOLVE_AT",
					"before_sec": 300,
					"after_sec": 300
				},
				"rule": {
					"type": "PRICE_THRESHOLD",
					"comparator": "GTE",
					"threshold_price": "85000.00000000"
				}
			}
		}
	}`)

	_, _, err := ParseContract("CRYPTO", []string{"YES", "NO"}, 1775886400, raw)
	if err == nil {
		t.Fatalf("expected unsupported provider to be rejected")
	}
}
