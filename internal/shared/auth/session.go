package auth

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethmath "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

const (
	DefaultSessionScope               = "TRADE"
	DefaultTradingKeyScheme           = "ED25519"
	DefaultWalletSignatureStandard    = "EIP712_V4"
	TradingAuthorizationDomainName    = "FunnyOption Trading Authorization"
	TradingAuthorizationDomainVersion = "2"
	AuthorizeTradingKeyAction         = "AUTHORIZE_TRADING_KEY"
	CanonicalTradingKeyAuthVersion    = "TRADING_KEY_V2"
	LegacySessionCompatAuthVersion    = "LEGACY_SESSION_COMPAT"
)

type SessionGrant struct {
	WalletAddress    string
	SessionPublicKey string
	Scope            string
	ChainID          int64
	Nonce            string
	IssuedAtMillis   int64
	ExpiresAtMillis  int64
}

type OrderIntent struct {
	SessionID         string
	WalletAddress     string
	UserID            int64
	MarketID          int64
	Outcome           string
	Side              string
	OrderType         string
	TimeInForce       string
	Price             int64
	Quantity          int64
	ClientOrderID     string
	Nonce             uint64
	RequestedAtMillis int64
}

type TradingKeyAuthorization struct {
	WalletAddress            string
	TradingPublicKey         string
	TradingKeyScheme         string
	Scope                    string
	Challenge                string
	ChallengeExpiresAtMillis int64
	KeyExpiresAtMillis       int64
	ChainID                  int64
	VaultAddress             string
}

type AuthorizedTradingKey struct {
	TradingKeyID       string
	AccountID          int64
	WalletAddress      string
	TradingPublicKey   string
	TradingKeyScheme   string
	Scope              string
	ChainID            int64
	VaultAddress       string
	Status             string
	ExpiresAtMillis    int64
	AuthorizationNonce string
}

type TradingKeyAuthorizationWitness struct {
	AuthVersion              string `json:"auth_version"`
	VerifierEligible         bool   `json:"verifier_eligible"`
	AuthorizationRef         string `json:"authorization_ref"`
	TradingKeyID             string `json:"trading_key_id"`
	AccountID                int64  `json:"account_id"`
	WalletAddress            string `json:"wallet_address"`
	ChainID                  int64  `json:"chain_id"`
	VaultAddress             string `json:"vault_address"`
	TradingPublicKey         string `json:"trading_public_key"`
	TradingKeyScheme         string `json:"trading_key_scheme"`
	Scope                    string `json:"scope"`
	KeyStatus                string `json:"key_status"`
	Challenge                string `json:"challenge"`
	ChallengeExpiresAtMillis int64  `json:"challenge_expires_at_millis"`
	KeyExpiresAtMillis       int64  `json:"key_expires_at_millis"`
	AuthorizedAtMillis       int64  `json:"authorized_at_millis"`
	WalletSignatureStandard  string `json:"wallet_signature_standard"`
	WalletTypedDataHash      string `json:"wallet_typed_data_hash"`
	WalletSignature          string `json:"wallet_signature"`
}

type OrderIntentWitness struct {
	SessionID         string `json:"session_id"`
	WalletAddress     string `json:"wallet_address"`
	UserID            int64  `json:"user_id"`
	MarketID          int64  `json:"market_id"`
	Outcome           string `json:"outcome"`
	Side              string `json:"side"`
	OrderType         string `json:"order_type"`
	TimeInForce       string `json:"time_in_force"`
	Price             int64  `json:"price"`
	Quantity          int64  `json:"quantity"`
	ClientOrderID     string `json:"client_order_id,omitempty"`
	Nonce             uint64 `json:"nonce"`
	RequestedAtMillis int64  `json:"requested_at_millis"`
	Message           string `json:"message"`
	MessageHash       string `json:"message_hash"`
	Signature         string `json:"signature"`
}

type OrderAuthorizationWitness struct {
	AuthVersion        string             `json:"auth_version"`
	VerifierEligible   bool               `json:"verifier_eligible"`
	IneligibleReason   string             `json:"ineligible_reason,omitempty"`
	AuthorizationRef   string             `json:"authorization_ref,omitempty"`
	TradingKeyID       string             `json:"trading_key_id"`
	AccountID          int64              `json:"account_id"`
	WalletAddress      string             `json:"wallet_address"`
	ChainID            int64              `json:"chain_id"`
	VaultAddress       string             `json:"vault_address"`
	TradingPublicKey   string             `json:"trading_public_key"`
	TradingKeyScheme   string             `json:"trading_key_scheme"`
	Scope              string             `json:"scope"`
	KeyStatus          string             `json:"key_status"`
	KeyExpiresAtMillis int64              `json:"key_expires_at_millis"`
	Intent             OrderIntentWitness `json:"intent"`
}

type VerifierAuthBinding struct {
	AuthorizationRef string `json:"authorization_ref"`
	TradingKeyID     string `json:"trading_key_id"`
	AccountID        int64  `json:"account_id"`
	WalletAddress    string `json:"wallet_address"`
	ChainID          int64  `json:"chain_id"`
	VaultAddress     string `json:"vault_address"`
	TradingPublicKey string `json:"trading_public_key"`
	TradingKeyScheme string `json:"trading_key_scheme"`
	Scope            string `json:"scope"`
	KeyStatus        string `json:"key_status"`
}

func (g SessionGrant) Normalize() SessionGrant {
	g.WalletAddress = NormalizeHex(g.WalletAddress)
	g.SessionPublicKey = NormalizeHex(g.SessionPublicKey)
	g.Scope = strings.ToUpper(strings.TrimSpace(g.Scope))
	if g.Scope == "" {
		g.Scope = DefaultSessionScope
	}
	g.Nonce = strings.TrimSpace(g.Nonce)
	return g
}

func (g SessionGrant) Validate(now time.Time) error {
	normalized := g.Normalize()
	if normalized.WalletAddress == "" {
		return fmt.Errorf("wallet address is required")
	}
	if normalized.SessionPublicKey == "" {
		return fmt.Errorf("session public key is required")
	}
	if normalized.ChainID <= 0 {
		return fmt.Errorf("chain_id must be positive")
	}
	if normalized.Nonce == "" {
		return fmt.Errorf("nonce is required")
	}
	if normalized.IssuedAtMillis <= 0 || normalized.ExpiresAtMillis <= 0 {
		return fmt.Errorf("issued_at and expires_at are required")
	}
	if normalized.ExpiresAtMillis <= normalized.IssuedAtMillis {
		return fmt.Errorf("expires_at must be greater than issued_at")
	}
	if now.UnixMilli() > normalized.ExpiresAtMillis {
		return fmt.Errorf("session grant expired")
	}
	return nil
}

func (g SessionGrant) SessionID() string {
	normalized := g.Normalize()
	sum := sha256.Sum256([]byte(normalized.WalletAddress + ":" + normalized.SessionPublicKey))
	return "sess_" + hex.EncodeToString(sum[:16])
}

func (g SessionGrant) Message() string {
	normalized := g.Normalize()
	return fmt.Sprintf(
		"FunnyOption Session Authorization\n\nwallet: %s\nsession_public_key: %s\nscope: %s\nchain_id: %d\nissued_at: %d\nexpires_at: %d\nnonce: %s\n",
		normalized.WalletAddress,
		normalized.SessionPublicKey,
		normalized.Scope,
		normalized.ChainID,
		normalized.IssuedAtMillis,
		normalized.ExpiresAtMillis,
		normalized.Nonce,
	)
}

func (i OrderIntent) Normalize() OrderIntent {
	i.SessionID = strings.TrimSpace(i.SessionID)
	i.WalletAddress = NormalizeHex(i.WalletAddress)
	i.Outcome = strings.ToUpper(strings.TrimSpace(i.Outcome))
	i.Side = strings.ToUpper(strings.TrimSpace(i.Side))
	i.OrderType = strings.ToUpper(strings.TrimSpace(i.OrderType))
	i.TimeInForce = strings.ToUpper(strings.TrimSpace(i.TimeInForce))
	i.ClientOrderID = strings.TrimSpace(i.ClientOrderID)
	return i
}

func (i OrderIntent) Validate(now time.Time) error {
	normalized := i.Normalize()
	if normalized.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	if normalized.WalletAddress == "" {
		return fmt.Errorf("wallet address is required")
	}
	if normalized.UserID <= 0 {
		return fmt.Errorf("user_id must be positive")
	}
	if normalized.MarketID <= 0 {
		return fmt.Errorf("market_id must be positive")
	}
	if normalized.Outcome == "" || normalized.Side == "" {
		return fmt.Errorf("outcome and side are required")
	}
	if normalized.OrderType == "" || normalized.TimeInForce == "" {
		return fmt.Errorf("order_type and time_in_force are required")
	}
	if normalized.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	if normalized.Nonce == 0 {
		return fmt.Errorf("order nonce must be positive")
	}
	if normalized.RequestedAtMillis <= 0 {
		return fmt.Errorf("requested_at is required")
	}
	if now.UnixMilli()-normalized.RequestedAtMillis > int64((5*time.Minute)/time.Millisecond) {
		return fmt.Errorf("order intent expired")
	}
	return nil
}

func (i OrderIntent) Message() string {
	normalized := i.Normalize()
	return fmt.Sprintf(
		"FunnyOption Order Authorization\n\nsession_id: %s\nwallet: %s\nuser_id: %d\nmarket_id: %d\noutcome: %s\nside: %s\norder_type: %s\ntime_in_force: %s\nprice: %d\nquantity: %d\nclient_order_id: %s\nnonce: %d\nrequested_at: %d\n",
		normalized.SessionID,
		normalized.WalletAddress,
		normalized.UserID,
		normalized.MarketID,
		normalized.Outcome,
		normalized.Side,
		normalized.OrderType,
		normalized.TimeInForce,
		normalized.Price,
		normalized.Quantity,
		normalized.ClientOrderID,
		normalized.Nonce,
		normalized.RequestedAtMillis,
	)
}

func (a TradingKeyAuthorization) Normalize() TradingKeyAuthorization {
	a.WalletAddress = NormalizeHex(a.WalletAddress)
	a.TradingPublicKey = NormalizeHex(a.TradingPublicKey)
	a.TradingKeyScheme = strings.ToUpper(strings.TrimSpace(a.TradingKeyScheme))
	if a.TradingKeyScheme == "" {
		a.TradingKeyScheme = DefaultTradingKeyScheme
	}
	a.Scope = strings.ToUpper(strings.TrimSpace(a.Scope))
	if a.Scope == "" {
		a.Scope = DefaultSessionScope
	}
	a.Challenge = NormalizeHex(a.Challenge)
	a.VaultAddress = NormalizeHex(a.VaultAddress)
	return a
}

func (a TradingKeyAuthorization) Validate(now time.Time) error {
	normalized := a.Normalize()
	if !common.IsHexAddress(normalized.WalletAddress) {
		return fmt.Errorf("wallet address is invalid")
	}
	if !common.IsHexAddress(normalized.VaultAddress) {
		return fmt.Errorf("vault address is invalid")
	}
	if normalized.ChainID <= 0 {
		return fmt.Errorf("chain_id must be positive")
	}
	if normalized.TradingKeyScheme != DefaultTradingKeyScheme {
		return fmt.Errorf("trading_key_scheme must be %s", DefaultTradingKeyScheme)
	}
	if normalized.Scope == "" {
		return fmt.Errorf("scope is required")
	}
	if err := validateFixedHex("trading public key", normalized.TradingPublicKey, ed25519.PublicKeySize); err != nil {
		return err
	}
	if err := validateFixedHex("challenge", normalized.Challenge, 32); err != nil {
		return err
	}
	if normalized.ChallengeExpiresAtMillis <= 0 {
		return fmt.Errorf("challenge_expires_at is required")
	}
	if now.UnixMilli() > normalized.ChallengeExpiresAtMillis {
		return fmt.Errorf("challenge expired")
	}
	if normalized.KeyExpiresAtMillis < 0 {
		return fmt.Errorf("key_expires_at must be zero or positive")
	}
	if normalized.KeyExpiresAtMillis > 0 && normalized.KeyExpiresAtMillis <= now.UnixMilli() {
		return fmt.Errorf("key authorization expired")
	}
	return nil
}

func (k AuthorizedTradingKey) Normalize() AuthorizedTradingKey {
	k.TradingKeyID = strings.TrimSpace(k.TradingKeyID)
	k.WalletAddress = NormalizeHex(k.WalletAddress)
	k.TradingPublicKey = NormalizeHex(k.TradingPublicKey)
	k.TradingKeyScheme = strings.ToUpper(strings.TrimSpace(k.TradingKeyScheme))
	if k.TradingKeyScheme == "" {
		k.TradingKeyScheme = DefaultTradingKeyScheme
	}
	k.Scope = strings.ToUpper(strings.TrimSpace(k.Scope))
	if k.Scope == "" {
		k.Scope = DefaultSessionScope
	}
	k.VaultAddress = NormalizeHex(k.VaultAddress)
	k.Status = strings.ToUpper(strings.TrimSpace(k.Status))
	k.AuthorizationNonce = formatHexWithPrefix(k.AuthorizationNonce)
	return k
}

func (k AuthorizedTradingKey) AuthorizationRef() string {
	normalized := k.Normalize()
	if normalized.TradingKeyID == "" || normalized.AuthorizationNonce == "" {
		return ""
	}
	return normalized.TradingKeyID + ":" + normalized.AuthorizationNonce
}

func (k AuthorizedTradingKey) VerifierEligibility() (bool, string) {
	normalized := k.Normalize()
	if !strings.HasPrefix(normalized.TradingKeyID, "tk_") {
		return false, "deprecated /api/v1/sessions compatibility trading key"
	}
	if normalized.VaultAddress == "" {
		return false, "blank-vault auth rows are deprecated compatibility state"
	}
	if normalized.ChainID <= 0 {
		return false, "chain_id is missing from trading key scope"
	}
	if normalized.TradingPublicKey == "" {
		return false, "trading public key is missing from trading key scope"
	}
	return true, ""
}

func (a TradingKeyAuthorization) TradingKeyID() string {
	normalized := a.Normalize()
	sum := sha256.Sum256([]byte(
		normalized.WalletAddress +
			":" + fmt.Sprintf("%d", normalized.ChainID) +
			":" + normalized.VaultAddress +
			":" + normalized.TradingPublicKey,
	))
	return "tk_" + hex.EncodeToString(sum[:16])
}

func BuildTradingKeyAuthorizationWitness(accountID int64, authz TradingKeyAuthorization, key AuthorizedTradingKey, walletSignature string, authorizedAtMillis int64) (TradingKeyAuthorizationWitness, error) {
	normalizedAuthz := authz.Normalize()
	normalizedKey := key.Normalize()
	typedDataHash, _, err := apitypes.TypedDataAndHash(normalizedAuthz.TypedData())
	if err != nil {
		return TradingKeyAuthorizationWitness{}, err
	}
	eligible, _ := normalizedKey.VerifierEligibility()
	return TradingKeyAuthorizationWitness{
		AuthVersion:              CanonicalTradingKeyAuthVersion,
		VerifierEligible:         eligible,
		AuthorizationRef:         normalizedKey.AuthorizationRef(),
		TradingKeyID:             normalizedKey.TradingKeyID,
		AccountID:                accountID,
		WalletAddress:            normalizedAuthz.WalletAddress,
		ChainID:                  normalizedAuthz.ChainID,
		VaultAddress:             normalizedAuthz.VaultAddress,
		TradingPublicKey:         normalizedAuthz.TradingPublicKey,
		TradingKeyScheme:         normalizedAuthz.TradingKeyScheme,
		Scope:                    normalizedAuthz.Scope,
		KeyStatus:                normalizedKey.Status,
		Challenge:                normalizedAuthz.Challenge,
		ChallengeExpiresAtMillis: normalizedAuthz.ChallengeExpiresAtMillis,
		KeyExpiresAtMillis:       normalizedAuthz.KeyExpiresAtMillis,
		AuthorizedAtMillis:       authorizedAtMillis,
		WalletSignatureStandard:  DefaultWalletSignatureStandard,
		WalletTypedDataHash:      "0x" + hex.EncodeToString(typedDataHash),
		WalletSignature:          NormalizeHex(walletSignature),
	}, nil
}

func BuildOrderIntentWitness(intent OrderIntent, signature string) OrderIntentWitness {
	normalized := intent.Normalize()
	message := normalized.Message()
	return OrderIntentWitness{
		SessionID:         normalized.SessionID,
		WalletAddress:     normalized.WalletAddress,
		UserID:            normalized.UserID,
		MarketID:          normalized.MarketID,
		Outcome:           normalized.Outcome,
		Side:              normalized.Side,
		OrderType:         normalized.OrderType,
		TimeInForce:       normalized.TimeInForce,
		Price:             normalized.Price,
		Quantity:          normalized.Quantity,
		ClientOrderID:     normalized.ClientOrderID,
		Nonce:             normalized.Nonce,
		RequestedAtMillis: normalized.RequestedAtMillis,
		Message:           message,
		MessageHash:       messageHashHex(message),
		Signature:         NormalizeHex(signature),
	}
}

func BuildOrderAuthorizationWitness(accountID int64, key AuthorizedTradingKey, intent OrderIntent, signature string) OrderAuthorizationWitness {
	normalizedKey := key.Normalize()
	eligible, reason := normalizedKey.VerifierEligibility()
	authVersion := CanonicalTradingKeyAuthVersion
	authorizationRef := normalizedKey.AuthorizationRef()
	if !strings.HasPrefix(normalizedKey.TradingKeyID, "tk_") {
		authVersion = LegacySessionCompatAuthVersion
		authorizationRef = ""
	}
	return OrderAuthorizationWitness{
		AuthVersion:        authVersion,
		VerifierEligible:   eligible,
		IneligibleReason:   reason,
		AuthorizationRef:   authorizationRef,
		TradingKeyID:       normalizedKey.TradingKeyID,
		AccountID:          accountID,
		WalletAddress:      normalizedKey.WalletAddress,
		ChainID:            normalizedKey.ChainID,
		VaultAddress:       normalizedKey.VaultAddress,
		TradingPublicKey:   normalizedKey.TradingPublicKey,
		TradingKeyScheme:   normalizedKey.TradingKeyScheme,
		Scope:              normalizedKey.Scope,
		KeyStatus:          normalizedKey.Status,
		KeyExpiresAtMillis: normalizedKey.ExpiresAtMillis,
		Intent:             BuildOrderIntentWitness(intent, signature),
	}
}

func (w TradingKeyAuthorizationWitness) VerifierBinding() (VerifierAuthBinding, error) {
	if !w.VerifierEligible {
		return VerifierAuthBinding{}, fmt.Errorf("trading key authorization witness is not verifier-eligible")
	}
	if strings.TrimSpace(w.AuthVersion) != CanonicalTradingKeyAuthVersion {
		return VerifierAuthBinding{}, fmt.Errorf("trading key authorization witness auth_version must be %s", CanonicalTradingKeyAuthVersion)
	}
	return normalizeVerifierAuthBinding(VerifierAuthBinding{
		AuthorizationRef: w.AuthorizationRef,
		TradingKeyID:     w.TradingKeyID,
		AccountID:        w.AccountID,
		WalletAddress:    w.WalletAddress,
		ChainID:          w.ChainID,
		VaultAddress:     w.VaultAddress,
		TradingPublicKey: w.TradingPublicKey,
		TradingKeyScheme: w.TradingKeyScheme,
		Scope:            w.Scope,
		KeyStatus:        w.KeyStatus,
	}).validate("trading key authorization witness")
}

func (w OrderAuthorizationWitness) VerifierBinding() (VerifierAuthBinding, error) {
	if !w.VerifierEligible {
		return VerifierAuthBinding{}, fmt.Errorf("order authorization witness is not verifier-eligible")
	}
	if strings.TrimSpace(w.AuthVersion) != CanonicalTradingKeyAuthVersion {
		return VerifierAuthBinding{}, fmt.Errorf("order authorization witness auth_version must be %s", CanonicalTradingKeyAuthVersion)
	}
	return normalizeVerifierAuthBinding(VerifierAuthBinding{
		AuthorizationRef: w.AuthorizationRef,
		TradingKeyID:     w.TradingKeyID,
		AccountID:        w.AccountID,
		WalletAddress:    w.WalletAddress,
		ChainID:          w.ChainID,
		VaultAddress:     w.VaultAddress,
		TradingPublicKey: w.TradingPublicKey,
		TradingKeyScheme: w.TradingKeyScheme,
		Scope:            w.Scope,
		KeyStatus:        w.KeyStatus,
	}).validate("order authorization witness")
}

func ValidateVerifierBindingMatch(authorized, order VerifierAuthBinding) error {
	left, err := normalizeVerifierAuthBinding(authorized).validate("authorized binding")
	if err != nil {
		return err
	}
	right, err := normalizeVerifierAuthBinding(order).validate("order binding")
	if err != nil {
		return err
	}
	if left != right {
		return fmt.Errorf("verifier auth binding mismatch: authorized=%+v order=%+v", left, right)
	}
	return nil
}

func (a TradingKeyAuthorization) TypedData() apitypes.TypedData {
	normalized := a.Normalize()
	return apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": {
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"AuthorizeTradingKey": {
				{Name: "action", Type: "string"},
				{Name: "wallet", Type: "address"},
				{Name: "tradingPublicKey", Type: "bytes32"},
				{Name: "tradingKeyScheme", Type: "string"},
				{Name: "scope", Type: "string"},
				{Name: "challenge", Type: "bytes32"},
				{Name: "challengeExpiresAt", Type: "uint64"},
				{Name: "keyExpiresAt", Type: "uint64"},
			},
		},
		PrimaryType: "AuthorizeTradingKey",
		Domain: apitypes.TypedDataDomain{
			Name:              TradingAuthorizationDomainName,
			Version:           TradingAuthorizationDomainVersion,
			ChainId:           ethmath.NewHexOrDecimal256(normalized.ChainID),
			VerifyingContract: normalized.VaultAddress,
		},
		Message: apitypes.TypedDataMessage{
			"action":             AuthorizeTradingKeyAction,
			"wallet":             normalized.WalletAddress,
			"tradingPublicKey":   normalized.TradingPublicKey,
			"tradingKeyScheme":   normalized.TradingKeyScheme,
			"scope":              normalized.Scope,
			"challenge":          normalized.Challenge,
			"challengeExpiresAt": fmt.Sprintf("%d", normalized.ChallengeExpiresAtMillis),
			"keyExpiresAt":       fmt.Sprintf("%d", normalized.KeyExpiresAtMillis),
		},
	}
}

func VerifyGrantSignature(grant SessionGrant, signature string) (string, error) {
	normalized := grant.Normalize()
	recovered, err := RecoverPersonalSignAddress(normalized.Message(), signature)
	if err != nil {
		return "", err
	}
	if recovered != normalized.WalletAddress {
		return "", fmt.Errorf("wallet signature does not match wallet address")
	}
	return recovered, nil
}

func VerifyTradingKeyAuthorizationSignature(auth TradingKeyAuthorization, signature string) (string, error) {
	normalized := auth.Normalize()
	recovered, err := RecoverTypedDataAddress(normalized.TypedData(), signature)
	if err != nil {
		return "", err
	}
	if recovered != normalized.WalletAddress {
		return "", fmt.Errorf("wallet signature does not match wallet address")
	}
	return recovered, nil
}

func VerifyOrderIntentSignature(intent OrderIntent, sessionPublicKey, signature string) error {
	pubKey, err := decodeHexBytes(sessionPublicKey)
	if err != nil {
		return fmt.Errorf("decode session public key: %w", err)
	}
	if len(pubKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid session public key length")
	}

	sig, err := decodeHexBytes(signature)
	if err != nil {
		return fmt.Errorf("decode order signature: %w", err)
	}
	if len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("invalid order signature length")
	}
	if !ed25519.Verify(ed25519.PublicKey(pubKey), []byte(intent.Message()), sig) {
		return fmt.Errorf("invalid session signature")
	}
	return nil
}

func RecoverPersonalSignAddress(message, signature string) (string, error) {
	raw, err := decodeHexBytes(signature)
	if err != nil {
		return "", fmt.Errorf("decode signature: %w", err)
	}
	if len(raw) != crypto.SignatureLength {
		return "", fmt.Errorf("invalid signature length")
	}

	sig := make([]byte, len(raw))
	copy(sig, raw)
	switch sig[crypto.RecoveryIDOffset] {
	case 27, 28:
		sig[crypto.RecoveryIDOffset] -= 27
	case 0, 1:
	default:
		return "", fmt.Errorf("invalid signature recovery id")
	}

	hash := accounts.TextHash([]byte(message))
	pubKey, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return "", fmt.Errorf("recover signer: %w", err)
	}
	return NormalizeHex(crypto.PubkeyToAddress(*pubKey).Hex()), nil
}

func RecoverTypedDataAddress(typedData apitypes.TypedData, signature string) (string, error) {
	raw, err := decodeHexBytes(signature)
	if err != nil {
		return "", fmt.Errorf("decode signature: %w", err)
	}
	if len(raw) != crypto.SignatureLength {
		return "", fmt.Errorf("invalid signature length")
	}

	sig := make([]byte, len(raw))
	copy(sig, raw)
	switch sig[crypto.RecoveryIDOffset] {
	case 27, 28:
		sig[crypto.RecoveryIDOffset] -= 27
	case 0, 1:
	default:
		return "", fmt.Errorf("invalid signature recovery id")
	}

	hash, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		return "", fmt.Errorf("build typed data hash: %w", err)
	}
	pubKey, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return "", fmt.Errorf("recover signer: %w", err)
	}
	return NormalizeHex(crypto.PubkeyToAddress(*pubKey).Hex()), nil
}

func decodeHexBytes(value string) ([]byte, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, fmt.Errorf("hex value is required")
	}
	if !strings.HasPrefix(trimmed, "0x") {
		trimmed = "0x" + trimmed
	}
	return hexutil.Decode(trimmed)
}

func NormalizeHex(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return strings.ToLower(trimmed)
}

func messageHashHex(message string) string {
	sum := sha256.Sum256([]byte(message))
	return "0x" + hex.EncodeToString(sum[:])
}

func formatHexWithPrefix(value string) string {
	normalized := NormalizeHex(value)
	if normalized == "" {
		return ""
	}
	if strings.HasPrefix(normalized, "0x") {
		return normalized
	}
	return "0x" + normalized
}

func normalizeVerifierAuthBinding(binding VerifierAuthBinding) VerifierAuthBinding {
	binding.AuthorizationRef = strings.TrimSpace(binding.AuthorizationRef)
	binding.TradingKeyID = strings.TrimSpace(binding.TradingKeyID)
	binding.WalletAddress = NormalizeHex(binding.WalletAddress)
	binding.VaultAddress = NormalizeHex(binding.VaultAddress)
	binding.TradingPublicKey = NormalizeHex(binding.TradingPublicKey)
	binding.TradingKeyScheme = strings.ToUpper(strings.TrimSpace(binding.TradingKeyScheme))
	binding.Scope = strings.ToUpper(strings.TrimSpace(binding.Scope))
	binding.KeyStatus = strings.ToUpper(strings.TrimSpace(binding.KeyStatus))
	return binding
}

func (b VerifierAuthBinding) validate(label string) (VerifierAuthBinding, error) {
	if strings.TrimSpace(b.AuthorizationRef) == "" {
		return VerifierAuthBinding{}, fmt.Errorf("%s authorization_ref is required", label)
	}
	if strings.TrimSpace(b.TradingKeyID) == "" {
		return VerifierAuthBinding{}, fmt.Errorf("%s trading_key_id is required", label)
	}
	if b.AccountID <= 0 {
		return VerifierAuthBinding{}, fmt.Errorf("%s account_id must be positive", label)
	}
	if b.WalletAddress == "" {
		return VerifierAuthBinding{}, fmt.Errorf("%s wallet_address is required", label)
	}
	if b.ChainID <= 0 {
		return VerifierAuthBinding{}, fmt.Errorf("%s chain_id must be positive", label)
	}
	if b.VaultAddress == "" {
		return VerifierAuthBinding{}, fmt.Errorf("%s vault_address is required", label)
	}
	if b.TradingPublicKey == "" {
		return VerifierAuthBinding{}, fmt.Errorf("%s trading_public_key is required", label)
	}
	if b.TradingKeyScheme == "" {
		return VerifierAuthBinding{}, fmt.Errorf("%s trading_key_scheme is required", label)
	}
	if b.Scope == "" {
		return VerifierAuthBinding{}, fmt.Errorf("%s scope is required", label)
	}
	if b.KeyStatus == "" {
		return VerifierAuthBinding{}, fmt.Errorf("%s key_status is required", label)
	}
	return b, nil
}

func validateFixedHex(label, value string, size int) error {
	raw, err := decodeHexBytes(value)
	if err != nil {
		return fmt.Errorf("decode %s: %w", label, err)
	}
	if len(raw) != size {
		return fmt.Errorf("invalid %s length", label)
	}
	return nil
}
