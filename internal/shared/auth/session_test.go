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

func TestBuildTradingKeyAuthorizationWitness(t *testing.T) {
	authz := TradingKeyAuthorization{
		WalletAddress:            "0x00000000000000000000000000000000000000aa",
		TradingPublicKey:         "0x8f931f3d9d6a93f2b05a1e8ef8356d7408be0f2f5f63c2dbcbf6c227f5f1c5d2",
		TradingKeyScheme:         DefaultTradingKeyScheme,
		Scope:                    DefaultSessionScope,
		Challenge:                "0x5fbe9af9d6ab53d4df3bcb43f9e6c5f26a4d9bc2a8f44a0ab2997f7dc2c5c94a",
		ChallengeExpiresAtMillis: time.Now().Add(5 * time.Minute).UnixMilli(),
		KeyExpiresAtMillis:       0,
		ChainID:                  97,
		VaultAddress:             "0x00000000000000000000000000000000000000bb",
	}
	key := AuthorizedTradingKey{
		TradingKeyID:       authz.TradingKeyID(),
		AccountID:          1001,
		WalletAddress:      authz.WalletAddress,
		TradingPublicKey:   authz.TradingPublicKey,
		TradingKeyScheme:   authz.TradingKeyScheme,
		Scope:              authz.Scope,
		ChainID:            authz.ChainID,
		VaultAddress:       authz.VaultAddress,
		Status:             "ACTIVE",
		AuthorizationNonce: authz.Challenge,
	}

	witness, err := BuildTradingKeyAuthorizationWitness(1001, authz, key, "0xdeadbeef", 1_775_886_400_000)
	if err != nil {
		t.Fatalf("BuildTradingKeyAuthorizationWitness returned error: %v", err)
	}
	if !witness.VerifierEligible {
		t.Fatalf("expected verifier-eligible witness, got %+v", witness)
	}
	if witness.AuthorizationRef != key.AuthorizationRef() {
		t.Fatalf("authorization_ref = %s, want %s", witness.AuthorizationRef, key.AuthorizationRef())
	}
	if witness.WalletTypedDataHash == "" {
		t.Fatalf("expected non-empty typed data hash")
	}
}

func TestVerifierBindingMatchesCanonicalTradingKeyAndOrderAuthorization(t *testing.T) {
	authz := TradingKeyAuthorization{
		WalletAddress:            "0x00000000000000000000000000000000000000aa",
		TradingPublicKey:         "0x8f931f3d9d6a93f2b05a1e8ef8356d7408be0f2f5f63c2dbcbf6c227f5f1c5d2",
		TradingKeyScheme:         DefaultTradingKeyScheme,
		Scope:                    DefaultSessionScope,
		Challenge:                "0x5fbe9af9d6ab53d4df3bcb43f9e6c5f26a4d9bc2a8f44a0ab2997f7dc2c5c94a",
		ChallengeExpiresAtMillis: time.Now().Add(5 * time.Minute).UnixMilli(),
		KeyExpiresAtMillis:       0,
		ChainID:                  97,
		VaultAddress:             "0x00000000000000000000000000000000000000bb",
	}
	key := AuthorizedTradingKey{
		TradingKeyID:       authz.TradingKeyID(),
		AccountID:          1001,
		WalletAddress:      authz.WalletAddress,
		TradingPublicKey:   authz.TradingPublicKey,
		TradingKeyScheme:   authz.TradingKeyScheme,
		Scope:              authz.Scope,
		ChainID:            authz.ChainID,
		VaultAddress:       authz.VaultAddress,
		Status:             "ACTIVE",
		AuthorizationNonce: authz.Challenge,
	}
	intent := OrderIntent{
		SessionID:         key.TradingKeyID,
		WalletAddress:     key.WalletAddress,
		UserID:            1001,
		MarketID:          88,
		Outcome:           "YES",
		Side:              "BUY",
		OrderType:         "LIMIT",
		TimeInForce:       "GTC",
		Price:             10,
		Quantity:          20,
		ClientOrderID:     "cli-1",
		Nonce:             7,
		RequestedAtMillis: time.Now().UnixMilli(),
	}

	authWitness, err := BuildTradingKeyAuthorizationWitness(1001, authz, key, "0xdeadbeef", 1_775_886_400_000)
	if err != nil {
		t.Fatalf("BuildTradingKeyAuthorizationWitness returned error: %v", err)
	}
	orderWitness := BuildOrderAuthorizationWitness(1001, key, intent, "0xfeedface")

	authBinding, err := authWitness.VerifierBinding()
	if err != nil {
		t.Fatalf("authWitness.VerifierBinding returned error: %v", err)
	}
	orderBinding, err := orderWitness.VerifierBinding()
	if err != nil {
		t.Fatalf("orderWitness.VerifierBinding returned error: %v", err)
	}
	if err := ValidateVerifierBindingMatch(authBinding, orderBinding); err != nil {
		t.Fatalf("ValidateVerifierBindingMatch returned error: %v", err)
	}
	if authBinding.AuthorizationRef != key.AuthorizationRef() {
		t.Fatalf("authorization_ref = %s, want %s", authBinding.AuthorizationRef, key.AuthorizationRef())
	}
}

func TestBuildOrderAuthorizationWitnessMarksLegacyCompatIneligible(t *testing.T) {
	key := AuthorizedTradingKey{
		TradingKeyID:       "sess_legacy",
		AccountID:          1001,
		WalletAddress:      "0x00000000000000000000000000000000000000aa",
		TradingPublicKey:   "0x8f931f3d9d6a93f2b05a1e8ef8356d7408be0f2f5f63c2dbcbf6c227f5f1c5d2",
		TradingKeyScheme:   DefaultTradingKeyScheme,
		Scope:              DefaultSessionScope,
		ChainID:            97,
		Status:             "ACTIVE",
		ExpiresAtMillis:    0,
		AuthorizationNonce: "sess_legacy_nonce",
	}
	intent := OrderIntent{
		SessionID:         key.TradingKeyID,
		WalletAddress:     key.WalletAddress,
		UserID:            1001,
		MarketID:          88,
		Outcome:           "YES",
		Side:              "BUY",
		OrderType:         "LIMIT",
		TimeInForce:       "GTC",
		Price:             10,
		Quantity:          20,
		ClientOrderID:     "cli-1",
		Nonce:             7,
		RequestedAtMillis: time.Now().UnixMilli(),
	}

	witness := BuildOrderAuthorizationWitness(1001, key, intent, "0xfeedface")
	if witness.VerifierEligible {
		t.Fatalf("expected legacy session witness to stay ineligible")
	}
	if witness.AuthVersion != LegacySessionCompatAuthVersion {
		t.Fatalf("auth_version = %s, want %s", witness.AuthVersion, LegacySessionCompatAuthVersion)
	}
	if !strings.Contains(witness.IneligibleReason, "/api/v1/sessions") {
		t.Fatalf("unexpected ineligible reason: %s", witness.IneligibleReason)
	}
	if witness.Intent.MessageHash == "" {
		t.Fatalf("expected non-empty order intent message hash")
	}
}

func TestOrderAuthorizationVerifierBindingRejectsLegacyCompatWitness(t *testing.T) {
	key := AuthorizedTradingKey{
		TradingKeyID:       "sess_legacy",
		AccountID:          1001,
		WalletAddress:      "0x00000000000000000000000000000000000000aa",
		TradingPublicKey:   "0x8f931f3d9d6a93f2b05a1e8ef8356d7408be0f2f5f63c2dbcbf6c227f5f1c5d2",
		TradingKeyScheme:   DefaultTradingKeyScheme,
		Scope:              DefaultSessionScope,
		ChainID:            97,
		Status:             "ACTIVE",
		ExpiresAtMillis:    0,
		AuthorizationNonce: "sess_legacy_nonce",
	}
	intent := OrderIntent{
		SessionID:         key.TradingKeyID,
		WalletAddress:     key.WalletAddress,
		UserID:            key.AccountID,
		MarketID:          88,
		Outcome:           "YES",
		Side:              "BUY",
		OrderType:         "LIMIT",
		TimeInForce:       "GTC",
		Price:             10,
		Quantity:          20,
		ClientOrderID:     "cli-legacy",
		Nonce:             7,
		RequestedAtMillis: time.Now().UnixMilli(),
	}

	witness := BuildOrderAuthorizationWitness(key.AccountID, key, intent, "0xfeedface")
	if _, err := witness.VerifierBinding(); err == nil {
		t.Fatalf("expected legacy compatibility witness to stay out of verifier binding contract")
	}
}
