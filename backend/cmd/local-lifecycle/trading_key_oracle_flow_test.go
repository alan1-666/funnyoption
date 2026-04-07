package main

import (
	"testing"
	"time"

	sharedauth "funnyoption/internal/shared/auth"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestSignTradingKeyAuthorizationRoundTrip(t *testing.T) {
	key, err := crypto.HexToECDSA("59c6995e998f97a5a004497e5daef0d4f7dcd0cfd5401397dbeed52b21965b1d")
	if err != nil {
		t.Fatalf("parse wallet key: %v", err)
	}
	walletAddress := crypto.PubkeyToAddress(key.PublicKey).Hex()

	authz := sharedauth.TradingKeyAuthorization{
		WalletAddress:            walletAddress,
		TradingPublicKey:         "0x8f931f3d9d6a93f2b05a1e8ef8356d7408be0f2f5f63c2dbcbf6c227f5f1c5d2",
		TradingKeyScheme:         sharedauth.DefaultTradingKeyScheme,
		Scope:                    sharedauth.DefaultSessionScope,
		Challenge:                "0x5fbe9af9d6ab53d4df3bcb43f9e6c5f26a4d9bc2a8f44a0ab2997f7dc2c5c94a",
		ChallengeExpiresAtMillis: time.Now().Add(5 * time.Minute).UnixMilli(),
		KeyExpiresAtMillis:       0,
		ChainID:                  31337,
		VaultAddress:             "0x00000000000000000000000000000000000000bb",
	}

	signature, err := signTradingKeyAuthorization(authz, key)
	if err != nil {
		t.Fatalf("signTradingKeyAuthorization returned error: %v", err)
	}

	recovered, err := sharedauth.VerifyTradingKeyAuthorizationSignature(authz, signature)
	if err != nil {
		t.Fatalf("VerifyTradingKeyAuthorizationSignature returned error: %v", err)
	}
	if recovered != sharedauth.NormalizeHex(walletAddress) {
		t.Fatalf("expected recovered wallet %s, got %s", sharedauth.NormalizeHex(walletAddress), recovered)
	}
}
