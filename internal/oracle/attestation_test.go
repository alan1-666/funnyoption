package oracle

import (
	"crypto/ecdsa"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func testSignerKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate test key: %v", err)
	}
	return key
}

func TestBuildAndVerifyOracleAttestation(t *testing.T) {
	key := testSignerKey(t)
	signerAddr := crypto.PubkeyToAddress(key.PublicKey)

	att, err := BuildOracleAttestation("BTCUSDT", "67500.50", 1712345678, "BINANCE", key)
	if err != nil {
		t.Fatalf("BuildOracleAttestation: %v", err)
	}

	if att.Version != AttestationVersion {
		t.Fatalf("version mismatch: got %d, want %d", att.Version, AttestationVersion)
	}
	if att.AssetPair != "BTCUSDT" {
		t.Fatalf("asset_pair: got %q, want BTCUSDT", att.AssetPair)
	}
	if att.Price != "67500.50" {
		t.Fatalf("price: got %q, want 67500.50", att.Price)
	}
	if att.Timestamp != 1712345678 {
		t.Fatalf("timestamp: got %d, want 1712345678", att.Timestamp)
	}
	if att.Provider != "BINANCE" {
		t.Fatalf("provider: got %q, want BINANCE", att.Provider)
	}
	if att.Signature == "" {
		t.Fatal("signature is empty")
	}

	err = VerifyOracleAttestation(att, []common.Address{signerAddr})
	if err != nil {
		t.Fatalf("VerifyOracleAttestation (valid): %v", err)
	}
}

func TestVerifyOracleAttestationRejectsTamperedPrice(t *testing.T) {
	key := testSignerKey(t)
	signerAddr := crypto.PubkeyToAddress(key.PublicKey)

	att, err := BuildOracleAttestation("BTCUSDT", "67500.50", 1712345678, "BINANCE", key)
	if err != nil {
		t.Fatalf("BuildOracleAttestation: %v", err)
	}

	att.Price = "99999.99"
	err = VerifyOracleAttestation(att, []common.Address{signerAddr})
	if err == nil {
		t.Fatal("expected verification to fail for tampered price")
	}
}

func TestVerifyOracleAttestationRejectsUnknownSigner(t *testing.T) {
	key := testSignerKey(t)
	otherKey := testSignerKey(t)
	otherAddr := crypto.PubkeyToAddress(otherKey.PublicKey)

	att, err := BuildOracleAttestation("ETHUSDT", "3500.00", 1712345678, "BINANCE", key)
	if err != nil {
		t.Fatalf("BuildOracleAttestation: %v", err)
	}

	err = VerifyOracleAttestation(att, []common.Address{otherAddr})
	if err == nil {
		t.Fatal("expected verification to fail for unknown signer")
	}
}

func TestVerifyOracleAttestationRejectsEmptySignature(t *testing.T) {
	att := &SignedOracleAttestation{
		Version:   AttestationVersion,
		AssetPair: "BTCUSDT",
		Price:     "67500.50",
		Timestamp: 1712345678,
		Provider:  "BINANCE",
		Signature: "",
	}
	err := VerifyOracleAttestation(att, []common.Address{common.HexToAddress("0x1234")})
	if err == nil {
		t.Fatal("expected error for empty signature")
	}
}

func TestVerifyOracleAttestationRejectsNilAttestation(t *testing.T) {
	err := VerifyOracleAttestation(nil, []common.Address{common.HexToAddress("0x1234")})
	if err == nil {
		t.Fatal("expected error for nil attestation")
	}
}

func TestRecoverAttestationSigner(t *testing.T) {
	key := testSignerKey(t)
	expectedAddr := crypto.PubkeyToAddress(key.PublicKey)

	att, err := BuildOracleAttestation("BTCUSDT", "67500.50", 1712345678, "BINANCE", key)
	if err != nil {
		t.Fatalf("BuildOracleAttestation: %v", err)
	}

	recovered, err := RecoverAttestationSigner(att)
	if err != nil {
		t.Fatalf("RecoverAttestationSigner: %v", err)
	}
	if recovered != expectedAddr {
		t.Fatalf("recovered signer: got %s, want %s", recovered.Hex(), expectedAddr.Hex())
	}
}

func TestBuildOracleAttestationDeterministic(t *testing.T) {
	key := testSignerKey(t)

	att1, err := BuildOracleAttestation("BTCUSDT", "67500.50", 1712345678, "BINANCE", key)
	if err != nil {
		t.Fatalf("first build: %v", err)
	}
	att2, err := BuildOracleAttestation("BTCUSDT", "67500.50", 1712345678, "BINANCE", key)
	if err != nil {
		t.Fatalf("second build: %v", err)
	}

	if att1.Signature != att2.Signature {
		t.Fatalf("expected deterministic signatures, got different values")
	}
}

func TestBuildOracleAttestationNormalizesInput(t *testing.T) {
	key := testSignerKey(t)
	signerAddr := crypto.PubkeyToAddress(key.PublicKey)

	att1, err := BuildOracleAttestation("btcusdt", "67500.50", 1712345678, " binance ", key)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if att1.AssetPair != "BTCUSDT" {
		t.Fatalf("asset_pair not normalized: %q", att1.AssetPair)
	}
	if att1.Provider != "BINANCE" {
		t.Fatalf("provider not normalized: %q", att1.Provider)
	}

	err = VerifyOracleAttestation(att1, []common.Address{signerAddr})
	if err != nil {
		t.Fatalf("normalized attestation should verify: %v", err)
	}
}

func TestBuildOracleAttestationRejectsInvalidInputs(t *testing.T) {
	key := testSignerKey(t)

	tests := []struct {
		name      string
		assetPair string
		price     string
		timestamp int64
		provider  string
	}{
		{"empty asset pair", "", "67500.50", 1712345678, "BINANCE"},
		{"empty price", "BTCUSDT", "", 1712345678, "BINANCE"},
		{"zero timestamp", "BTCUSDT", "67500.50", 0, "BINANCE"},
		{"negative timestamp", "BTCUSDT", "67500.50", -1, "BINANCE"},
		{"empty provider", "BTCUSDT", "67500.50", 1712345678, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := BuildOracleAttestation(tc.assetPair, tc.price, tc.timestamp, tc.provider, key)
			if err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
		})
	}
}

func TestBuildOracleAttestationRejectsNilKey(t *testing.T) {
	_, err := BuildOracleAttestation("BTCUSDT", "67500.50", 1712345678, "BINANCE", nil)
	if err == nil {
		t.Fatal("expected error for nil key")
	}
}
