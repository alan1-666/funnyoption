package dto

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

type OperatorAction struct {
	WalletAddress string `json:"wallet_address"`
	RequestedAt   int64  `json:"requested_at"`
	Signature     string `json:"signature"`
}

func NormalizeCreateMarketRequest(input CreateMarketRequest) CreateMarketRequest {
	input.Title = cleanOperatorText(input.Title)
	input.Description = cleanOperatorText(input.Description)
	input.CategoryKey = NormalizeMarketCategoryKey(input.CategoryKey, input.Metadata)
	input.CoverImageURL = strings.TrimSpace(input.CoverImageURL)
	input.CoverSourceURL = strings.TrimSpace(input.CoverSourceURL)
	input.CoverSourceName = cleanOperatorText(input.CoverSourceName)
	input.Status = strings.ToUpper(cleanOperatorText(input.Status))
	if input.Status == "" {
		input.Status = "OPEN"
	}
	input.CollateralAsset = strings.ToUpper(cleanOperatorText(input.CollateralAsset))
	if input.CollateralAsset == "" {
		input.CollateralAsset = "USDT"
	}
	if input.OpenAt < 0 {
		input.OpenAt = 0
	}
	if input.CloseAt < 0 {
		input.CloseAt = 0
	}
	if input.ResolveAt < 0 {
		input.ResolveAt = 0
	}
	return input
}

func NormalizeBinaryOutcome(value string) (string, bool) {
	outcome := strings.ToUpper(cleanOperatorText(value))
	switch outcome {
	case "YES", "NO":
		return outcome, true
	default:
		return "", false
	}
}

func (req CreateMarketRequest) OperatorMessage() string {
	normalized := NormalizeCreateMarketRequest(req)
	options, err := NormalizeMarketOptions(normalized.Options)
	if err != nil {
		options = DefaultBinaryMarketOptions()
	}
	metadata := parseOperatorMetadata(normalized.Metadata)

	sourceKind := strings.ToLower(cleanOperatorText(stringFromMetadata(metadata, "sourceKind", "source_kind")))
	if sourceKind == "" {
		sourceKind = "manual"
	}
	sourceURL := normalized.CoverSourceURL
	if sourceURL == "" {
		sourceURL = strings.TrimSpace(stringFromMetadata(metadata, "sourceUrl", "cover_source_url"))
	}
	sourceSlug := cleanOperatorText(stringFromMetadata(metadata, "sourceSlug", "source_slug"))
	sourceName := normalized.CoverSourceName
	if sourceName == "" {
		sourceName = cleanOperatorText(stringFromMetadata(metadata, "sourceName", "cover_source_name"))
	}
	if sourceName == "" {
		sourceName = "Polymarket"
	}
	coverImage := normalized.CoverImageURL
	if coverImage == "" {
		coverImage = strings.TrimSpace(stringFromMetadata(metadata, "coverImage", "cover_image_url"))
	}

	return fmt.Sprintf(
		"FunnyOption Operator Authorization\n\naction: CREATE_MARKET\nwallet: %s\ntitle: %s\ndescription: %s\ncategory: %s\nsource_kind: %s\nsource_url: %s\nsource_slug: %s\nsource_name: %s\ncover_image: %s\nstatus: %s\ncollateral_asset: %s\nopen_at: %d\nclose_at: %d\nresolve_at: %d\nrequested_at: %d\n",
		normalizeOperatorAddress(normalized.operatorWalletAddress()),
		normalized.Title,
		normalized.Description,
		normalized.CategoryKey,
		sourceKind,
		sourceURL,
		sourceSlug,
		sourceName,
		coverImage,
		normalized.Status,
		normalized.CollateralAsset,
		normalized.OpenAt,
		normalized.CloseAt,
		normalized.ResolveAt,
		normalized.operatorRequestedAt(),
	) + "options: " + buildMarketOptionSignatureFragment(options) + "\n"
}

func (req ResolveMarketRequest) OperatorMessage(marketID int64) string {
	return fmt.Sprintf(
		"FunnyOption Operator Authorization\n\naction: RESOLVE_MARKET\nwallet: %s\nmarket_id: %d\noutcome: %s\nrequested_at: %d\n",
		normalizeOperatorAddress(req.operatorWalletAddress()),
		marketID,
		strings.ToUpper(cleanOperatorText(req.Outcome)),
		req.operatorRequestedAt(),
	)
}

func buildBootstrapOperatorMessage(walletAddress string, marketID, userID, quantity int64, outcome string, price, requestedAt int64) string {
	return fmt.Sprintf(
		"FunnyOption Operator Authorization\n\naction: ISSUE_FIRST_LIQUIDITY\nwallet: %s\nmarket_id: %d\nuser_id: %d\nquantity: %d\noutcome: %s\nprice: %d\nrequested_at: %d\n",
		normalizeOperatorAddress(walletAddress),
		marketID,
		userID,
		quantity,
		strings.ToUpper(cleanOperatorText(outcome)),
		price,
		requestedAt,
	)
}

func (req CreateFirstLiquidityRequest) OperatorMessage(marketID int64) string {
	return buildBootstrapOperatorMessage(req.operatorWalletAddress(), marketID, req.UserID, req.Quantity, req.Outcome, req.Price, req.operatorRequestedAt())
}

func (req CreateFirstLiquidityRequest) BootstrapSemanticKey(marketID int64) string {
	return buildBootstrapSemanticKey(marketID, req.UserID, req.Quantity, req.Outcome, req.Price)
}

func (req CreateFirstLiquidityRequest) BootstrapOrderID(marketID int64) string {
	return buildBootstrapOrderID(req.BootstrapSemanticKey(marketID))
}

func (req CreateOrderRequest) BootstrapOperatorMessage() string {
	return buildBootstrapOperatorMessage(req.operatorWalletAddress(), req.MarketID, req.UserID, req.Quantity, req.Outcome, req.Price, req.operatorRequestedAt())
}

func (req CreateOrderRequest) BootstrapSemanticKey() string {
	return buildBootstrapSemanticKey(req.MarketID, req.UserID, req.Quantity, req.Outcome, req.Price)
}

func (req CreateOrderRequest) BootstrapOrderID() string {
	// `requested_at` stays inside the signed proof for freshness, but a fresh
	// timestamp alone must not create a second bootstrap action with identical terms.
	return buildBootstrapOrderID(req.BootstrapSemanticKey())
}

func buildBootstrapSemanticKey(marketID, userID, quantity int64, outcome string, price int64) string {
	normalizedOutcome, ok := NormalizeBinaryOutcome(outcome)
	if !ok {
		normalizedOutcome = strings.ToUpper(cleanOperatorText(outcome))
	}

	return fmt.Sprintf(
		"bootstrap-order:%d:%d:%d:%s:%d",
		marketID,
		userID,
		quantity,
		normalizedOutcome,
		price,
	)
}

func buildBootstrapOrderID(semanticKey string) string {
	sum := sha256.Sum256([]byte(semanticKey))
	return "ord_bootstrap_" + hex.EncodeToString(sum[:16])
}

func (req CreateMarketRequest) operatorWalletAddress() string {
	if req.Operator == nil {
		return ""
	}
	return req.Operator.WalletAddress
}

func (req ResolveMarketRequest) operatorWalletAddress() string {
	if req.Operator == nil {
		return ""
	}
	return req.Operator.WalletAddress
}

func (req CreateFirstLiquidityRequest) operatorWalletAddress() string {
	if req.Operator == nil {
		return ""
	}
	return req.Operator.WalletAddress
}

func (req CreateOrderRequest) operatorWalletAddress() string {
	if req.Operator == nil {
		return ""
	}
	return req.Operator.WalletAddress
}

func (req CreateMarketRequest) operatorRequestedAt() int64 {
	if req.Operator == nil {
		return 0
	}
	return req.Operator.RequestedAt
}

func (req ResolveMarketRequest) operatorRequestedAt() int64 {
	if req.Operator == nil {
		return 0
	}
	return req.Operator.RequestedAt
}

func (req CreateFirstLiquidityRequest) operatorRequestedAt() int64 {
	if req.Operator == nil {
		return 0
	}
	return req.Operator.RequestedAt
}

func (req CreateOrderRequest) operatorRequestedAt() int64 {
	if req.Operator == nil {
		return 0
	}
	return req.Operator.RequestedAt
}

func parseOperatorMetadata(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	metadata := map[string]any{}
	if err := json.Unmarshal(raw, &metadata); err != nil || metadata == nil {
		return map[string]any{}
	}
	return metadata
}

func stringFromMetadata(metadata map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := metadata[key]; ok {
			switch typed := value.(type) {
			case string:
				return typed
			case fmt.Stringer:
				return typed.String()
			}
		}
	}
	return ""
}

func buildMarketOptionSignatureFragment(options []MarketOption) string {
	parts := make([]string, 0, len(options))
	for _, option := range options {
		state := "0"
		if option.IsActive {
			state = "1"
		}
		parts = append(parts, fmt.Sprintf("%s:%s:%s:%d:%s", option.Key, option.Label, option.ShortLabel, option.SortOrder, state))
	}
	return strings.Join(parts, "|")
}

func cleanOperatorText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func normalizeOperatorAddress(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
