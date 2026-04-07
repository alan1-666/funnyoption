package oracle

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	AttestationVersion  = 1
	attestationDomain   = "funny-oracle-attestation-v1"
	ethSignedMsgPrefix  = "\x19Ethereum Signed Message:\n32"
)

type SignedOracleAttestation struct {
	Version       int    `json:"version"`
	AssetPair     string `json:"asset_pair"`
	Price         string `json:"price"`
	Timestamp     int64  `json:"timestamp"`
	Provider      string `json:"provider"`
	Signature     string `json:"signature"`
	SignerAddress string `json:"signer_address"`
}

func BuildOracleAttestation(assetPair, price string, timestamp int64, provider string, signerKey *ecdsa.PrivateKey) (*SignedOracleAttestation, error) {
	if signerKey == nil {
		return nil, fmt.Errorf("signer key is required")
	}
	assetPair = strings.ToUpper(strings.TrimSpace(assetPair))
	provider = strings.ToUpper(strings.TrimSpace(provider))
	price = strings.TrimSpace(price)
	if assetPair == "" || price == "" || timestamp <= 0 || provider == "" {
		return nil, fmt.Errorf("asset_pair, price, timestamp, and provider are all required")
	}

	digest := attestationDigest(assetPair, price, timestamp, provider)
	prefixed := prefixedHash(digest)

	sig, err := crypto.Sign(prefixed.Bytes(), signerKey)
	if err != nil {
		return nil, fmt.Errorf("signing attestation: %w", err)
	}
	if len(sig) == 65 {
		sig[64] += 27
	}

	signerAddr := crypto.PubkeyToAddress(signerKey.PublicKey)

	return &SignedOracleAttestation{
		Version:       AttestationVersion,
		AssetPair:     assetPair,
		Price:         price,
		Timestamp:     timestamp,
		Provider:      provider,
		Signature:     common.Bytes2Hex(sig),
		SignerAddress:  strings.ToLower(signerAddr.Hex()),
	}, nil
}

func VerifyOracleAttestation(att *SignedOracleAttestation, trustedSigners []common.Address) error {
	if att == nil {
		return fmt.Errorf("attestation is nil")
	}
	if att.Version != AttestationVersion {
		return fmt.Errorf("unsupported attestation version %d", att.Version)
	}
	if att.AssetPair == "" || att.Price == "" || att.Timestamp <= 0 || att.Provider == "" {
		return fmt.Errorf("attestation fields are incomplete")
	}
	if att.Signature == "" {
		return fmt.Errorf("attestation signature is empty")
	}

	sig := common.FromHex(att.Signature)
	if len(sig) != 65 {
		return fmt.Errorf("attestation signature has invalid length %d", len(sig))
	}

	recoverySig := make([]byte, 65)
	copy(recoverySig, sig)
	if recoverySig[64] >= 27 {
		recoverySig[64] -= 27
	}

	digest := attestationDigest(
		strings.ToUpper(strings.TrimSpace(att.AssetPair)),
		strings.TrimSpace(att.Price),
		att.Timestamp,
		strings.ToUpper(strings.TrimSpace(att.Provider)),
	)
	prefixed := prefixedHash(digest)

	pubKey, err := crypto.SigToPub(prefixed.Bytes(), recoverySig)
	if err != nil {
		return fmt.Errorf("recovering signer from attestation: %w", err)
	}
	recovered := crypto.PubkeyToAddress(*pubKey)

	for _, trusted := range trustedSigners {
		if recovered == trusted {
			return nil
		}
	}
	return fmt.Errorf("attestation signer %s is not in trusted set", recovered.Hex())
}

func RecoverAttestationSigner(att *SignedOracleAttestation) (common.Address, error) {
	if att == nil {
		return common.Address{}, fmt.Errorf("attestation is nil")
	}
	sig := common.FromHex(att.Signature)
	if len(sig) != 65 {
		return common.Address{}, fmt.Errorf("invalid signature length %d", len(sig))
	}

	recoverySig := make([]byte, 65)
	copy(recoverySig, sig)
	if recoverySig[64] >= 27 {
		recoverySig[64] -= 27
	}

	digest := attestationDigest(
		strings.ToUpper(strings.TrimSpace(att.AssetPair)),
		strings.TrimSpace(att.Price),
		att.Timestamp,
		strings.ToUpper(strings.TrimSpace(att.Provider)),
	)
	prefixed := prefixedHash(digest)

	pubKey, err := crypto.SigToPub(prefixed.Bytes(), recoverySig)
	if err != nil {
		return common.Address{}, fmt.Errorf("recovering signer: %w", err)
	}
	return crypto.PubkeyToAddress(*pubKey), nil
}

func attestationDigest(assetPair, price string, timestamp int64, provider string) common.Hash {
	ts := new(big.Int).SetInt64(timestamp)
	packed := crypto.Keccak256(
		[]byte(attestationDomain),
		common.LeftPadBytes([]byte(assetPair), 32),
		crypto.Keccak256([]byte(price)),
		common.LeftPadBytes(ts.Bytes(), 32),
		common.LeftPadBytes([]byte(provider), 32),
	)
	return common.BytesToHash(packed)
}

func prefixedHash(digest common.Hash) common.Hash {
	return crypto.Keccak256Hash([]byte(ethSignedMsgPrefix), digest.Bytes())
}
