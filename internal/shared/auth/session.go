package auth

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	DefaultSessionScope = "TRADE"
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
