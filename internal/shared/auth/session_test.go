package auth

import (
	"crypto/ed25519"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

func TestSessionGrantValidateAndMessage(t *testing.T) {
	now := time.UnixMilli(1_711_972_800_000)
	grant := SessionGrant{
		WalletAddress:    "0xABCDEF",
		SessionPublicKey: "0x123456",
		ChainID:          97,
		Nonce:            "sess_123",
		IssuedAtMillis:   now.UnixMilli(),
		ExpiresAtMillis:  now.Add(2 * time.Hour).UnixMilli(),
	}

	if err := grant.Validate(now); err != nil {
		t.Fatalf("expected valid session grant, got error: %v", err)
	}

	msg := grant.Message()
	if !strings.Contains(msg, "FunnyOption Session Authorization") {
		t.Fatalf("unexpected message: %s", msg)
	}
	if !strings.HasPrefix(grant.SessionID(), "sess_") {
		t.Fatalf("unexpected session id: %s", grant.SessionID())
	}
}

func TestSessionGrantExpired(t *testing.T) {
	now := time.UnixMilli(1_711_972_800_000)
	grant := SessionGrant{
		WalletAddress:    "0xabc",
		SessionPublicKey: "0xdef",
		ChainID:          97,
		Nonce:            "sess_123",
		IssuedAtMillis:   now.Add(-2 * time.Hour).UnixMilli(),
		ExpiresAtMillis:  now.Add(-1 * time.Hour).UnixMilli(),
	}

	if err := grant.Validate(now); err == nil {
		t.Fatalf("expected expired error")
	}
}

func TestVerifyGrantSignature(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	grant := SessionGrant{
		WalletAddress:    crypto.PubkeyToAddress(key.PublicKey).Hex(),
		SessionPublicKey: "0x123456",
		ChainID:          97,
		Nonce:            "sess_123",
		IssuedAtMillis:   time.Now().Add(-time.Minute).UnixMilli(),
		ExpiresAtMillis:  time.Now().Add(time.Hour).UnixMilli(),
	}

	signature, err := crypto.Sign(accounts.TextHash([]byte(grant.Message())), key)
	if err != nil {
		t.Fatalf("Sign returned error: %v", err)
	}
	recovered, err := VerifyGrantSignature(grant, hexutil.Encode(signature))
	if err != nil {
		t.Fatalf("VerifyGrantSignature returned error: %v", err)
	}
	if recovered != NormalizeHex(grant.WalletAddress) {
		t.Fatalf("unexpected recovered wallet: %s", recovered)
	}
}

func TestVerifyOrderIntentSignature(t *testing.T) {
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}

	intent := OrderIntent{
		SessionID:         "sess_1",
		WalletAddress:     "0xabc",
		UserID:            1001,
		MarketID:          88,
		Outcome:           "YES",
		Side:              "BUY",
		OrderType:         "LIMIT",
		TimeInForce:       "GTC",
		Price:             10,
		Quantity:          20,
		ClientOrderID:     "cli-1",
		Nonce:             1,
		RequestedAtMillis: time.Now().UnixMilli(),
	}
	signature := ed25519.Sign(privKey, []byte(intent.Message()))

	if err := VerifyOrderIntentSignature(intent, hexutil.Encode(pubKey), hexutil.Encode(signature)); err != nil {
		t.Fatalf("VerifyOrderIntentSignature returned error: %v", err)
	}
}

func TestTradingKeyAuthorizationValidateAndID(t *testing.T) {
	now := time.UnixMilli(1_711_972_800_000)
	authz := TradingKeyAuthorization{
		WalletAddress:            "0x00000000000000000000000000000000000000aa",
		TradingPublicKey:         "0x8f931f3d9d6a93f2b05a1e8ef8356d7408be0f2f5f63c2dbcbf6c227f5f1c5d2",
		TradingKeyScheme:         "ed25519",
		Scope:                    "trade",
		Challenge:                "0x5fbe9af9d6ab53d4df3bcb43f9e6c5f26a4d9bc2a8f44a0ab2997f7dc2c5c94a",
		ChallengeExpiresAtMillis: now.Add(5 * time.Minute).UnixMilli(),
		KeyExpiresAtMillis:       0,
		ChainID:                  97,
		VaultAddress:             "0x00000000000000000000000000000000000000bb",
	}

	if err := authz.Validate(now); err != nil {
		t.Fatalf("expected valid trading key authorization, got error: %v", err)
	}
	if !strings.HasPrefix(authz.TradingKeyID(), "tk_") {
		t.Fatalf("unexpected trading key id: %s", authz.TradingKeyID())
	}
}

func TestVerifyTradingKeyAuthorizationSignature(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	authz := TradingKeyAuthorization{
		WalletAddress:            crypto.PubkeyToAddress(key.PublicKey).Hex(),
		TradingPublicKey:         "0x8f931f3d9d6a93f2b05a1e8ef8356d7408be0f2f5f63c2dbcbf6c227f5f1c5d2",
		TradingKeyScheme:         "ED25519",
		Scope:                    "TRADE",
		Challenge:                "0x5fbe9af9d6ab53d4df3bcb43f9e6c5f26a4d9bc2a8f44a0ab2997f7dc2c5c94a",
		ChallengeExpiresAtMillis: time.Now().Add(5 * time.Minute).UnixMilli(),
		KeyExpiresAtMillis:       0,
		ChainID:                  97,
		VaultAddress:             "0x00000000000000000000000000000000000000bb",
	}

	digest, _, err := apitypes.TypedDataAndHash(authz.TypedData())
	if err != nil {
		t.Fatalf("TypedDataAndHash returned error: %v", err)
	}
	signature, err := crypto.Sign(digest, key)
	if err != nil {
		t.Fatalf("Sign returned error: %v", err)
	}

	recovered, err := VerifyTradingKeyAuthorizationSignature(authz, hexutil.Encode(signature))
	if err != nil {
		t.Fatalf("VerifyTradingKeyAuthorizationSignature returned error: %v", err)
	}
	if recovered != NormalizeHex(authz.WalletAddress) {
		t.Fatalf("unexpected recovered wallet: %s", recovered)
	}
}
