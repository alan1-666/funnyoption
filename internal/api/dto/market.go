package dto

import (
	"bytes"
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

var ErrInvalidMarketOptions = errors.New("market options are invalid")

func NormalizeMarketCategoryKey(categoryKey string, rawMetadata json.RawMessage) string {
	if normalized, ok := lookupCategoryKey(categoryKey); ok {
		return normalized
	}

	metadata := parseOperatorMetadata(rawMetadata)
	for _, key := range []string{"categoryKey", "category_key", "category"} {
		if normalized, ok := lookupCategoryKey(stringFromMetadata(metadata, key)); ok {
			return normalized
		}
	}
	return "CRYPTO"
}

func NormalizeMarketCategoryFilter(value string) string {
	if normalized, ok := lookupCategoryKey(value); ok {
		return normalized
	}
	return strings.ToUpper(cleanOperatorText(value))
}

func DefaultBinaryMarketOptions() []MarketOption {
	return []MarketOption{
		{
			Key:        "YES",
			Label:      "是",
			ShortLabel: "是",
			SortOrder:  10,
			IsActive:   true,
		},
		{
			Key:        "NO",
			Label:      "否",
			ShortLabel: "否",
			SortOrder:  20,
			IsActive:   true,
		},
	}
}

func NormalizeMarketOptions(input []MarketOption) ([]MarketOption, error) {
	if len(input) == 0 {
		return DefaultBinaryMarketOptions(), nil
	}

	options := make([]MarketOption, 0, len(input))
	seen := make(map[string]struct{}, len(input))

	for index, option := range input {
		key := normalizeMarketOptionKey(option.Key)
		label := cleanOperatorText(option.Label)
		shortLabel := cleanOperatorText(option.ShortLabel)
		if key == "" || label == "" {
			return nil, ErrInvalidMarketOptions
		}
		if _, exists := seen[key]; exists {
			return nil, ErrInvalidMarketOptions
		}
		seen[key] = struct{}{}
		if shortLabel == "" {
			shortLabel = label
		}
		sortOrder := option.SortOrder
		if sortOrder <= 0 {
			sortOrder = (index + 1) * 10
		}

		normalized := MarketOption{
			Key:        key,
			Label:      label,
			ShortLabel: shortLabel,
			SortOrder:  sortOrder,
			IsActive:   true,
			Metadata:   normalizeJSON(option.Metadata),
		}
		options = append(options, normalized)
	}

	sort.SliceStable(options, func(i, j int) bool {
		if options[i].SortOrder == options[j].SortOrder {
			return options[i].Key < options[j].Key
		}
		return options[i].SortOrder < options[j].SortOrder
	})
	return options, nil
}

func IsBinaryTradingOptions(options []MarketOption) bool {
	if len(options) != 2 {
		return false
	}
	seen := map[string]bool{}
	for _, option := range options {
		if !option.IsActive {
			return false
		}
		seen[option.Key] = true
	}
	return seen["YES"] && seen["NO"]
}

func normalizeMarketOptionKey(value string) string {
	cleaned := strings.ToUpper(cleanOperatorText(value))
	cleaned = strings.ReplaceAll(cleaned, " ", "_")
	return cleaned
}

func normalizeJSON(raw json.RawMessage) json.RawMessage {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}
	return json.RawMessage(trimmed)
}

func lookupCategoryKey(value string) (string, bool) {
	normalized := strings.ToUpper(cleanOperatorText(value))
	normalized = strings.ReplaceAll(normalized, " ", "")
	switch normalized {
	case "CRYPTO", "加密", "POLYMARKET", "MARKET", "EVENT", "OPERATIONS", "MANUAL", "FLOW", "MACRO", "LOCALQA", "LOCAL":
		return "CRYPTO", true
	case "SPORTS", "SPORT", "体育":
		return "SPORTS", true
	default:
		return "", false
	}
}
