package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"

	"funnyoption/internal/rollup"
	sharedauth "funnyoption/internal/shared/auth"

	gnarklogger "github.com/consensys/gnark/logger"
)

type fixtureEnvelope struct {
	Primary   rollup.VerifierArtifactBundle `json:"primary"`
	Secondary rollup.VerifierArtifactBundle `json:"secondary"`
}

func main() {
	gnarklogger.Disable()

	authBatch, targetBatch, err := verifierGateTestBatches()
	if err != nil {
		fatal(err)
	}
	primary, err := rollup.BuildVerifierArtifactBundle([]rollup.StoredBatch{authBatch}, targetBatch)
	if err != nil {
		fatal(err)
	}

	secondBatch := targetBatch
	secondBatch.StateRoot = "590e0e068f686f45ffe60ef2f14c2a832b7a4e6d250e99436dbed283118466a5"
	secondBatch.PrevStateRoot = targetBatch.PrevStateRoot
	secondBatch.BatchID = targetBatch.BatchID
	secondBatch.EncodingVersion = targetBatch.EncodingVersion
	secondary, err := rollup.BuildVerifierArtifactBundle([]rollup.StoredBatch{authBatch}, secondBatch)
	if err != nil {
		fatal(err)
	}

	if err := json.NewEncoder(os.Stdout).Encode(fixtureEnvelope{
		Primary:   primary,
		Secondary: secondary,
	}); err != nil {
		fatal(err)
	}
}

func verifierGateTestBatches() (rollup.StoredBatch, rollup.StoredBatch, error) {
	authz := sharedauth.TradingKeyAuthorization{
		WalletAddress:            "0x00000000000000000000000000000000000000aa",
		TradingPublicKey:         "0x8f931f3d9d6a93f2b05a1e8ef8356d7408be0f2f5f63c2dbcbf6c227f5f1c5d2",
		TradingKeyScheme:         sharedauth.DefaultTradingKeyScheme,
		Scope:                    sharedauth.DefaultSessionScope,
		Challenge:                "0x5fbe9af9d6ab53d4df3bcb43f9e6c5f26a4d9bc2a8f44a0ab2997f7dc2c5c94a",
		ChallengeExpiresAtMillis: 1775886700000,
		KeyExpiresAtMillis:       0,
		ChainID:                  97,
		VaultAddress:             "0x00000000000000000000000000000000000000bb",
	}
	key := sharedauth.AuthorizedTradingKey{
		TradingKeyID:       authz.TradingKeyID(),
		AccountID:          1001,
		WalletAddress:      authz.WalletAddress,
		TradingPublicKey:   authz.TradingPublicKey,
		TradingKeyScheme:   authz.TradingKeyScheme,
		Scope:              authz.Scope,
		ChainID:            authz.ChainID,
		VaultAddress:       authz.VaultAddress,
		Status:             "ACTIVE",
		ExpiresAtMillis:    0,
		AuthorizationNonce: authz.Challenge,
	}
	authWitness, err := sharedauth.BuildTradingKeyAuthorizationWitness(key.AccountID, authz, key, "0xdeadbeef", 1775886400000)
	if err != nil {
		return rollup.StoredBatch{}, rollup.StoredBatch{}, err
	}

	intent := sharedauth.OrderIntent{
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
		ClientOrderID:     "cli-1",
		Nonce:             7,
		RequestedAtMillis: 1775886400000,
	}
	orderWitness := sharedauth.BuildOrderAuthorizationWitness(key.AccountID, key, intent, "0xfeedface")

	authEntries := []rollup.JournalEntry{
		mustEntry(1, rollup.EntryTypeTradingKeyAuthorized, rollup.SourceTypeAPIAuth, key.AuthorizationRef(), rollup.TradingKeyAuthorizedPayload{
			AuthorizationWitness: authWitness,
		}),
	}
	authInput, authHash, err := rollup.EncodeBatchInput(authEntries)
	if err != nil {
		return rollup.StoredBatch{}, rollup.StoredBatch{}, err
	}

	targetEntries := []rollup.JournalEntry{
		mustEntry(7, rollup.EntryTypeNonceAdvanced, rollup.SourceTypeAPIAuth, fmt.Sprintf("%s:%d", key.TradingKeyID, intent.Nonce), rollup.NonceAdvancedPayload{
			AccountID:          key.AccountID,
			AuthKeyID:          key.TradingKeyID,
			Scope:              key.Scope,
			KeyStatus:          key.Status,
			AcceptedNonce:      intent.Nonce,
			NextNonce:          intent.Nonce + 1,
			OccurredAtMillis:   intent.RequestedAtMillis,
			OrderAuthorization: &orderWitness,
		}),
	}
	targetInput, targetHash, err := rollup.EncodeBatchInput(targetEntries)
	if err != nil {
		return rollup.StoredBatch{}, rollup.StoredBatch{}, err
	}

	return rollup.StoredBatch{
			BatchID:         1,
			EncodingVersion: rollup.BatchEncodingVersion,
			FirstSequence:   1,
			LastSequence:    1,
			EntryCount:      len(authEntries),
			InputData:       authInput,
			InputHash:       authHash,
			PrevStateRoot:   testHexRoot("prev_state_root_auth"),
			StateRoot:       testHexRoot("next_state_root_auth"),
		}, rollup.StoredBatch{
			BatchID:         2,
			EncodingVersion: rollup.BatchEncodingVersion,
			FirstSequence:   7,
			LastSequence:    7,
			EntryCount:      len(targetEntries),
			InputData:       targetInput,
			InputHash:       targetHash,
			PrevStateRoot:   testHexRoot("next_state_root_auth"),
			StateRoot:       testHexRoot("next_state_root_orders"),
		}, nil
}

func mustEntry(sequence uint64, entryType, sourceType, sourceRef string, payload any) rollup.JournalEntry {
	encoded, err := json.Marshal(payload)
	if err != nil {
		fatal(err)
	}
	return rollup.JournalEntry{
		Sequence:   int64(sequence),
		EntryType:  entryType,
		SourceType: sourceType,
		SourceRef:  sourceRef,
		Payload:    encoded,
	}
}

func testHexRoot(label string) string {
	return hashStrings("verifier_contract_test", label)
}

func hashStrings(parts ...string) string {
	h := sha256.New()
	var size [8]byte
	for _, part := range parts {
		binary.BigEndian.PutUint64(size[:], uint64(len(part)))
		_, _ = h.Write(size[:])
		_, _ = h.Write([]byte(part))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
