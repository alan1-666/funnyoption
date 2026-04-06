package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"
)

const (
	ResolutionModeOraclePrice       = "ORACLE_PRICE"
	ResolutionMarketKindCryptoPrice = "CRYPTO_PRICE_THRESHOLD"
	ResolverTypeOraclePrice         = "ORACLE_PRICE"
	OracleSourceKindHTTPJSON        = "HTTP_JSON"
	OracleProviderKeyBinance        = "BINANCE"
	OracleInstrumentKindSpot        = "SPOT"
	OraclePriceFieldLastPrice       = "LAST_PRICE"
	OracleRoundingModeRoundHalfUp   = "ROUND_HALF_UP"
	OracleWindowAnchorResolveAt     = "RESOLVE_AT"
	OracleRuleTypePriceThreshold    = "PRICE_THRESHOLD"
	ErrorCodeSourceTimeout          = "SOURCE_TIMEOUT"
	ErrorCodeSourceUnavailable      = "SOURCE_UNAVAILABLE"
	ErrorCodeStalePrice             = "STALE_PRICE"
	ErrorCodePriceNotInWindow       = "PRICE_NOT_IN_WINDOW"
	ErrorCodeUnsupportedSymbol      = "UNSUPPORTED_SYMBOL"
	ErrorCodeUnsupportedRule        = "UNSUPPORTED_RULE"
	ErrorCodeConflictingObservation = "CONFLICTING_OBSERVATION"
	ErrorCodeInvalidMetadata        = "INVALID_METADATA"
	DispatchStatusPending           = "PENDING"
	DispatchStatusDispatched        = "DISPATCHED"
	resolutionMetadataKey           = "resolution"
)

type ResolutionMetadata struct {
	Version               int                    `json:"version"`
	Mode                  string                 `json:"mode"`
	MarketKind            string                 `json:"market_kind"`
	ManualFallbackAllowed bool                   `json:"manual_fallback_allowed"`
	Oracle                ResolutionOracleConfig `json:"oracle"`
}

type ResolutionOracleConfig struct {
	SourceKind  string                `json:"source_kind"`
	ProviderKey string                `json:"provider_key"`
	Instrument  ResolutionInstrument  `json:"instrument"`
	Price       ResolutionPriceConfig `json:"price"`
	Window      ResolutionWindow      `json:"window"`
	Rule        ResolutionRule        `json:"rule"`
}

type ResolutionInstrument struct {
	Kind       string `json:"kind"`
	BaseAsset  string `json:"base_asset"`
	QuoteAsset string `json:"quote_asset"`
	Symbol     string `json:"symbol"`
}

type ResolutionPriceConfig struct {
	Field         string `json:"field"`
	Scale         int    `json:"scale"`
	RoundingMode  string `json:"rounding_mode"`
	MaxDataAgeSec int64  `json:"max_data_age_sec"`
}

type ResolutionWindow struct {
	Anchor    string `json:"anchor"`
	BeforeSec int64  `json:"before_sec"`
	AfterSec  int64  `json:"after_sec"`
}

type ResolutionRule struct {
	Type           string `json:"type"`
	Comparator     string `json:"comparator"`
	ThresholdPrice string `json:"threshold_price"`
}

type Contract struct {
	Metadata        ResolutionMetadata
	ThresholdScaled *big.Int
}

type Observation struct {
	FetchedAt      int64
	EffectiveAt    int64
	ObservedPrice  string
	ObservedScaled *big.Int
	RawPayload     any
	RawPayloadHash string
}

type Evidence struct {
	Version        int                    `json:"version"`
	ResolutionMode string                 `json:"resolution_mode"`
	Source         EvidenceSource         `json:"source"`
	Rule           EvidenceRule           `json:"rule"`
	Window         EvidenceWindow         `json:"window"`
	Observation    *EvidenceObservation   `json:"observation,omitempty"`
	Attestation    *EvidenceAttestation   `json:"attestation,omitempty"`
	Retry          EvidenceRetry          `json:"retry"`
	Dispatch       *EvidenceDispatch      `json:"dispatch,omitempty"`
}

type EvidenceAttestation struct {
	Version       int    `json:"version"`
	AssetPair     string `json:"asset_pair"`
	Price         string `json:"price"`
	Timestamp     int64  `json:"timestamp"`
	Provider      string `json:"provider"`
	Signature     string `json:"signature"`
	SignerAddress string `json:"signer_address"`
}

type EvidenceSource struct {
	SourceKind  string               `json:"source_kind"`
	ProviderKey string               `json:"provider_key"`
	Instrument  ResolutionInstrument `json:"instrument"`
	PriceField  string               `json:"price_field"`
	PriceScale  int                  `json:"price_scale"`
}

type EvidenceRule struct {
	Type           string `json:"type"`
	Comparator     string `json:"comparator"`
	ThresholdPrice string `json:"threshold_price"`
}

type EvidenceWindow struct {
	Anchor        string `json:"anchor"`
	TargetTime    int64  `json:"target_time"`
	BeforeSec     int64  `json:"before_sec"`
	AfterSec      int64  `json:"after_sec"`
	MaxDataAgeSec int64  `json:"max_data_age_sec"`
}

type EvidenceObservation struct {
	ObservationID   string `json:"observation_id"`
	FetchedAt       int64  `json:"fetched_at"`
	EffectiveAt     int64  `json:"effective_at"`
	ObservedPrice   string `json:"observed_price"`
	ResolvedOutcome string `json:"resolved_outcome"`
	RawPayloadHash  string `json:"raw_payload_hash"`
	RawPayload      any    `json:"raw_payload"`
}

type EvidenceRetry struct {
	AttemptCount  int    `json:"attempt_count"`
	LastAttemptAt int64  `json:"last_attempt_at"`
	NextRetryAt   int64  `json:"next_retry_at"`
	LastErrorCode string `json:"last_error_code"`
}

type EvidenceDispatch struct {
	Status        string `json:"status"`
	AttemptCount  int    `json:"attempt_count"`
	LastAttemptAt int64  `json:"last_attempt_at"`
	DispatchedAt  int64  `json:"dispatched_at"`
	LastError     string `json:"last_error,omitempty"`
}

type StoredEvidence struct {
	Observation *EvidenceObservation `json:"observation,omitempty"`
	Retry       EvidenceRetry        `json:"retry"`
	Dispatch    *EvidenceDispatch    `json:"dispatch,omitempty"`
}

func HasOracleResolutionMode(raw json.RawMessage) bool {
	metadata := parseMetadataObject(raw)
	if metadata == nil {
		return false
	}
	resolutionRaw, ok := metadata[resolutionMetadataKey]
	if !ok {
		return false
	}
	var resolution struct {
		Mode string `json:"mode"`
	}
	if err := json.Unmarshal(resolutionRaw, &resolution); err != nil {
		return false
	}
	return normalizeUpper(resolution.Mode) == ResolutionModeOraclePrice
}

func ParseContract(categoryKey string, optionKeys []string, resolveAt int64, raw json.RawMessage) (*Contract, bool, error) {
	metadata := parseMetadataObject(raw)
	if metadata == nil {
		return nil, false, nil
	}

	resolutionRaw, ok := metadata[resolutionMetadataKey]
	if !ok {
		return nil, false, nil
	}
	if len(bytes.TrimSpace(resolutionRaw)) == 0 || bytes.Equal(bytes.TrimSpace(resolutionRaw), []byte("null")) {
		return nil, false, fmt.Errorf("%s", invalidMetadataMessage("metadata.resolution must be an object"))
	}

	var contract Contract
	if err := json.Unmarshal(resolutionRaw, &contract.Metadata); err != nil {
		return nil, false, fmt.Errorf("%s", invalidMetadataMessage("metadata.resolution must be an object"))
	}
	normalizeContract(&contract.Metadata)
	if err := validateContract(categoryKey, optionKeys, resolveAt, &contract); err != nil {
		return nil, true, err
	}
	return &contract, true, nil
}

func CanonicalizeMetadata(raw json.RawMessage, resolution *ResolutionMetadata) json.RawMessage {
	if resolution == nil {
		return raw
	}
	metadata := parseMetadataAny(raw)
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata[resolutionMetadataKey] = resolution
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return raw
	}
	return encoded
}

func (c *Contract) ResolverRef(resolveAt int64) string {
	return fmt.Sprintf(
		"oracle_price:%s:%s:%d",
		c.Metadata.Oracle.ProviderKey,
		c.Metadata.Oracle.Instrument.Symbol,
		resolveAt,
	)
}

func (c *Contract) ResolveOutcome(observed *big.Int) (string, error) {
	if observed == nil || c.ThresholdScaled == nil {
		return "", fmt.Errorf("%s", invalidMetadataMessage("normalized oracle price is required"))
	}
	comparator := normalizeUpper(c.Metadata.Oracle.Rule.Comparator)
	switch comparator {
	case "GT":
		if observed.Cmp(c.ThresholdScaled) > 0 {
			return "YES", nil
		}
	case "GTE":
		if observed.Cmp(c.ThresholdScaled) >= 0 {
			return "YES", nil
		}
	case "LT":
		if observed.Cmp(c.ThresholdScaled) < 0 {
			return "YES", nil
		}
	case "LTE":
		if observed.Cmp(c.ThresholdScaled) <= 0 {
			return "YES", nil
		}
	default:
		return "", fmt.Errorf("%s", invalidMetadataMessage("metadata.resolution.oracle.rule.comparator is not supported"))
	}
	return "NO", nil
}

func (c *Contract) NormalizeObservedPrice(value string) (*big.Int, string, error) {
	return normalizeDecimal(value, c.Metadata.Oracle.Price.Scale)
}

func BuildEvidence(contract *Contract, resolveAt int64, observation *Observation, previous json.RawMessage, attemptCount int, nextRetryAt int64, lastErrorCode string, resolvedOutcome string, attestation *EvidenceAttestation) json.RawMessage {
	if contract == nil {
		return previous
	}
	if attemptCount <= 0 {
		attemptCount = nextAttemptCount(previous)
	}
	evidence := Evidence{
		Version:        1,
		ResolutionMode: ResolutionModeOraclePrice,
		Source: EvidenceSource{
			SourceKind:  contract.Metadata.Oracle.SourceKind,
			ProviderKey: contract.Metadata.Oracle.ProviderKey,
			Instrument:  contract.Metadata.Oracle.Instrument,
			PriceField:  contract.Metadata.Oracle.Price.Field,
			PriceScale:  contract.Metadata.Oracle.Price.Scale,
		},
		Rule: EvidenceRule{
			Type:           contract.Metadata.Oracle.Rule.Type,
			Comparator:     contract.Metadata.Oracle.Rule.Comparator,
			ThresholdPrice: contract.Metadata.Oracle.Rule.ThresholdPrice,
		},
		Window: EvidenceWindow{
			Anchor:        contract.Metadata.Oracle.Window.Anchor,
			TargetTime:    resolveAt,
			BeforeSec:     contract.Metadata.Oracle.Window.BeforeSec,
			AfterSec:      contract.Metadata.Oracle.Window.AfterSec,
			MaxDataAgeSec: contract.Metadata.Oracle.Price.MaxDataAgeSec,
		},
		Retry: EvidenceRetry{
			AttemptCount:  attemptCount,
			LastAttemptAt: nowUnix(),
			NextRetryAt:   nextRetryAt,
			LastErrorCode: lastErrorCode,
		},
	}
	if observation != nil {
		evidence.Observation = &EvidenceObservation{
			ObservationID:   contract.ResolverRef(resolveAt),
			FetchedAt:       observation.FetchedAt,
			EffectiveAt:     observation.EffectiveAt,
			ObservedPrice:   observation.ObservedPrice,
			ResolvedOutcome: resolvedOutcome,
			RawPayloadHash:  observation.RawPayloadHash,
			RawPayload:      observation.RawPayload,
		}
		evidence.Attestation = attestation
		evidence.Retry.LastAttemptAt = observation.FetchedAt
		evidence.Retry.NextRetryAt = 0
		evidence.Retry.LastErrorCode = lastErrorCode
		evidence.Dispatch = &EvidenceDispatch{
			Status: DispatchStatusPending,
		}
	}
	encoded, err := json.Marshal(evidence)
	if err != nil {
		return previous
	}
	return encoded
}

func nextAttemptCount(raw json.RawMessage) int {
	var stored StoredEvidence
	if len(bytes.TrimSpace(raw)) == 0 {
		return 1
	}
	if err := json.Unmarshal(raw, &stored); err != nil {
		return 1
	}
	if stored.Retry.AttemptCount <= 0 {
		return 1
	}
	return stored.Retry.AttemptCount + 1
}

func dispatchPending(raw json.RawMessage) bool {
	stored, ok := parseStoredEvidence(raw)
	if !ok {
		return true
	}
	return normalizeDispatchStatus(stored.Dispatch) != DispatchStatusDispatched
}

func recordDispatchAttempt(raw json.RawMessage, dispatched bool, attemptedAt int64, lastError string) json.RawMessage {
	evidence, ok := parseEvidence(raw)
	if !ok {
		return raw
	}
	if attemptedAt <= 0 {
		attemptedAt = nowUnix()
	}
	attemptCount := 1
	if evidence.Dispatch != nil && evidence.Dispatch.AttemptCount > 0 {
		attemptCount = evidence.Dispatch.AttemptCount + 1
	}
	dispatch := &EvidenceDispatch{
		Status:        DispatchStatusPending,
		AttemptCount:  attemptCount,
		LastAttemptAt: attemptedAt,
		LastError:     strings.TrimSpace(lastError),
	}
	if dispatched {
		dispatch.Status = DispatchStatusDispatched
		dispatch.DispatchedAt = attemptedAt
		dispatch.LastError = ""
	}
	evidence.Dispatch = dispatch
	encoded, err := json.Marshal(evidence)
	if err != nil {
		return raw
	}
	return encoded
}

func parseStoredEvidence(raw json.RawMessage) (StoredEvidence, bool) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return StoredEvidence{}, false
	}
	var stored StoredEvidence
	if err := json.Unmarshal(trimmed, &stored); err != nil {
		return StoredEvidence{}, false
	}
	return stored, true
}

func parseEvidence(raw json.RawMessage) (Evidence, bool) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return Evidence{}, false
	}
	var evidence Evidence
	if err := json.Unmarshal(trimmed, &evidence); err != nil {
		return Evidence{}, false
	}
	return evidence, true
}

func normalizeDispatchStatus(dispatch *EvidenceDispatch) string {
	if dispatch == nil {
		return ""
	}
	return normalizeUpper(dispatch.Status)
}

func normalizeContract(metadata *ResolutionMetadata) {
	if metadata == nil {
		return
	}
	metadata.Mode = normalizeUpper(metadata.Mode)
	metadata.MarketKind = normalizeUpper(metadata.MarketKind)
	metadata.Oracle.SourceKind = normalizeUpper(metadata.Oracle.SourceKind)
	metadata.Oracle.ProviderKey = normalizeUpper(metadata.Oracle.ProviderKey)
	metadata.Oracle.Instrument.Kind = normalizeUpper(metadata.Oracle.Instrument.Kind)
	metadata.Oracle.Instrument.BaseAsset = normalizeUpper(metadata.Oracle.Instrument.BaseAsset)
	metadata.Oracle.Instrument.QuoteAsset = normalizeUpper(metadata.Oracle.Instrument.QuoteAsset)
	metadata.Oracle.Instrument.Symbol = normalizeUpper(metadata.Oracle.Instrument.Symbol)
	metadata.Oracle.Price.Field = normalizeUpper(metadata.Oracle.Price.Field)
	metadata.Oracle.Price.RoundingMode = normalizeUpper(metadata.Oracle.Price.RoundingMode)
	metadata.Oracle.Window.Anchor = normalizeUpper(metadata.Oracle.Window.Anchor)
	metadata.Oracle.Rule.Type = normalizeUpper(metadata.Oracle.Rule.Type)
	metadata.Oracle.Rule.Comparator = normalizeUpper(metadata.Oracle.Rule.Comparator)
	metadata.Oracle.Rule.ThresholdPrice = strings.TrimSpace(metadata.Oracle.Rule.ThresholdPrice)
}

func validateContract(categoryKey string, optionKeys []string, resolveAt int64, contract *Contract) error {
	if contract == nil {
		return fmt.Errorf("%s", invalidMetadataMessage("metadata.resolution is required"))
	}
	if contract.Metadata.Mode != ResolutionModeOraclePrice {
		return fmt.Errorf("%s", invalidMetadataMessage("metadata.resolution.mode must be ORACLE_PRICE"))
	}
	if normalizeUpper(categoryKey) != "CRYPTO" {
		return fmt.Errorf("%s", invalidMetadataMessage("oracle-settled markets must use category_key = CRYPTO"))
	}
	if !isBinaryOptionKeys(optionKeys) {
		return fmt.Errorf("%s", invalidMetadataMessage("oracle-settled markets must use YES/NO binary options"))
	}
	if resolveAt <= 0 {
		return fmt.Errorf("%s", invalidMetadataMessage("oracle-settled markets require resolve_at > 0"))
	}
	if contract.Metadata.Version != 1 {
		return fmt.Errorf("%s", invalidMetadataMessage("metadata.resolution.version must be 1"))
	}
	if contract.Metadata.MarketKind != ResolutionMarketKindCryptoPrice {
		return fmt.Errorf("%s", invalidMetadataMessage("metadata.resolution.market_kind must be CRYPTO_PRICE_THRESHOLD"))
	}
	if !contract.Metadata.ManualFallbackAllowed {
		return fmt.Errorf("%s", invalidMetadataMessage("metadata.resolution.manual_fallback_allowed must be true"))
	}
	if contract.Metadata.Oracle.SourceKind != OracleSourceKindHTTPJSON {
		return fmt.Errorf("%s", invalidMetadataMessage("metadata.resolution.oracle.source_kind must be HTTP_JSON"))
	}
	if contract.Metadata.Oracle.ProviderKey != OracleProviderKeyBinance {
		return fmt.Errorf("%s", invalidMetadataMessage("metadata.resolution.oracle.provider_key must be BINANCE"))
	}
	if contract.Metadata.Oracle.Instrument.Kind != OracleInstrumentKindSpot {
		return fmt.Errorf("%s", invalidMetadataMessage("metadata.resolution.oracle.instrument.kind must be SPOT"))
	}
	if contract.Metadata.Oracle.Instrument.Symbol == "" {
		return fmt.Errorf("%s", invalidMetadataMessage("metadata.resolution.oracle.instrument.symbol is required"))
	}
	if contract.Metadata.Oracle.Price.Field != OraclePriceFieldLastPrice {
		return fmt.Errorf("%s", invalidMetadataMessage("metadata.resolution.oracle.price.field must be LAST_PRICE"))
	}
	if contract.Metadata.Oracle.Price.Scale < 0 || contract.Metadata.Oracle.Price.Scale > 18 {
		return fmt.Errorf("%s", invalidMetadataMessage("metadata.resolution.oracle.price.scale must be between 0 and 18"))
	}
	if contract.Metadata.Oracle.Price.RoundingMode != OracleRoundingModeRoundHalfUp {
		return fmt.Errorf("%s", invalidMetadataMessage("metadata.resolution.oracle.price.rounding_mode must be ROUND_HALF_UP"))
	}
	if contract.Metadata.Oracle.Price.MaxDataAgeSec <= 0 {
		return fmt.Errorf("%s", invalidMetadataMessage("metadata.resolution.oracle.price.max_data_age_sec must be greater than 0"))
	}
	if contract.Metadata.Oracle.Window.Anchor != OracleWindowAnchorResolveAt {
		return fmt.Errorf("%s", invalidMetadataMessage("metadata.resolution.oracle.window.anchor must be RESOLVE_AT"))
	}
	if contract.Metadata.Oracle.Window.BeforeSec < 0 || contract.Metadata.Oracle.Window.AfterSec < 0 {
		return fmt.Errorf("%s", invalidMetadataMessage("metadata.resolution.oracle.window.before_sec and after_sec must be non-negative"))
	}
	if contract.Metadata.Oracle.Rule.Type != OracleRuleTypePriceThreshold {
		return fmt.Errorf("%s", invalidMetadataMessage("metadata.resolution.oracle.rule.type must be PRICE_THRESHOLD"))
	}
	switch contract.Metadata.Oracle.Rule.Comparator {
	case "GT", "GTE", "LT", "LTE":
	default:
		return fmt.Errorf("%s", invalidMetadataMessage("metadata.resolution.oracle.rule.comparator must be GT, GTE, LT, or LTE"))
	}
	thresholdScaled, normalizedThreshold, err := normalizeDecimal(contract.Metadata.Oracle.Rule.ThresholdPrice, contract.Metadata.Oracle.Price.Scale)
	if err != nil {
		return fmt.Errorf("%s", invalidMetadataMessage("metadata.resolution.oracle.rule.threshold_price must be a decimal string compatible with price.scale"))
	}
	contract.Metadata.Oracle.Rule.ThresholdPrice = normalizedThreshold
	contract.ThresholdScaled = thresholdScaled
	return nil
}

func normalizeDecimal(value string, scale int) (*big.Int, string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, "", fmt.Errorf("decimal value is required")
	}
	rat, ok := new(big.Rat).SetString(trimmed)
	if !ok {
		return nil, "", fmt.Errorf("invalid decimal value")
	}
	if rat.Sign() < 0 {
		return nil, "", fmt.Errorf("decimal value must not be negative")
	}

	scaleFactor := pow10(scale)
	scaled := new(big.Rat).Mul(rat, new(big.Rat).SetInt(scaleFactor))
	num := new(big.Int).Set(scaled.Num())
	den := new(big.Int).Set(scaled.Denom())
	quotient := new(big.Int)
	remainder := new(big.Int)
	quotient.QuoRem(num, den, remainder)

	if remainder.Sign() != 0 {
		doubleRemainder := new(big.Int).Mul(remainder, big.NewInt(2))
		if doubleRemainder.Cmp(den) >= 0 {
			quotient.Add(quotient, big.NewInt(1))
		}
	}

	return quotient, formatScaled(quotient, scale), nil
}

func formatScaled(value *big.Int, scale int) string {
	if value == nil {
		return ""
	}
	if scale == 0 {
		return value.String()
	}
	sign := ""
	if value.Sign() < 0 {
		sign = "-"
		value = new(big.Int).Abs(value)
	}
	digits := value.String()
	if len(digits) <= scale {
		padding := strings.Repeat("0", scale-len(digits)+1)
		digits = padding + digits
	}
	cut := len(digits) - scale
	return sign + digits[:cut] + "." + digits[cut:]
}

func parseMetadataObject(raw json.RawMessage) map[string]json.RawMessage {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}
	metadata := map[string]json.RawMessage{}
	if err := json.Unmarshal(trimmed, &metadata); err != nil {
		return nil
	}
	return metadata
}

func parseMetadataAny(raw json.RawMessage) map[string]any {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}
	metadata := map[string]any{}
	if err := json.Unmarshal(trimmed, &metadata); err != nil {
		return nil
	}
	return metadata
}

func isBinaryOptionKeys(optionKeys []string) bool {
	if len(optionKeys) != 2 {
		return false
	}
	seen := map[string]bool{}
	for _, key := range optionKeys {
		seen[normalizeUpper(key)] = true
	}
	return seen["YES"] && seen["NO"]
}

func normalizeUpper(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func invalidMetadataMessage(detail string) string {
	return fmt.Sprintf("%s: %s", ErrorCodeInvalidMetadata, detail)
}

func pow10(scale int) *big.Int {
	value := big.NewInt(1)
	for i := 0; i < scale; i++ {
		value.Mul(value, big.NewInt(10))
	}
	return value
}

func nowUnix() int64 {
	return time.Now().Unix()
}
