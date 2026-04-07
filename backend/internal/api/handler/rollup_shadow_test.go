package handler

import (
	"strings"
	"testing"

	"funnyoption/internal/api/dto"
	"funnyoption/internal/rollup"
	sharedauth "funnyoption/internal/shared/auth"
)

func TestBuildNonceAdvanceEntry(t *testing.T) {
	session := dto.SessionResponse{
		SessionID:        "tk_live",
		UserID:           1001,
		WalletAddress:    "0x00000000000000000000000000000000000000aa",
		SessionPublicKey: "0x8f931f3d9d6a93f2b05a1e8ef8356d7408be0f2f5f63c2dbcbf6c227f5f1c5d2",
		Scope:            "trade",
		ChainID:          97,
		VaultAddress:     "0x00000000000000000000000000000000000000bb",
		SessionNonce:     "0x5fbe9af9d6ab53d4df3bcb43f9e6c5f26a4d9bc2a8f44a0ab2997f7dc2c5c94a",
		Status:           "active",
	}
	intent := sharedauth.OrderIntent{
		SessionID:         session.SessionID,
		WalletAddress:     session.WalletAddress,
		UserID:            session.UserID,
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
	witness := sharedauth.BuildOrderAuthorizationWitness(session.UserID, authorizedTradingKeyFromSession(session), intent, "0xfeedface")

	entry, err := buildNonceAdvanceEntry(session, 7, 1775886400000, &witness)
	if err != nil {
		t.Fatalf("buildNonceAdvanceEntry returned error: %v", err)
	}
	if entry.EntryType != rollup.EntryTypeNonceAdvanced {
		t.Fatalf("entry_type = %s, want %s", entry.EntryType, rollup.EntryTypeNonceAdvanced)
	}
	if entry.SourceType != rollup.SourceTypeAPIAuth {
		t.Fatalf("source_type = %s, want %s", entry.SourceType, rollup.SourceTypeAPIAuth)
	}
	if entry.SourceRef != "tk_live:7" {
		t.Fatalf("source_ref = %s, want tk_live:7", entry.SourceRef)
	}

	payload, ok := entry.Payload.(rollup.NonceAdvancedPayload)
	if !ok {
		t.Fatalf("payload type = %T, want rollup.NonceAdvancedPayload", entry.Payload)
	}
	if payload.AccountID != 1001 {
		t.Fatalf("account_id = %d, want 1001", payload.AccountID)
	}
	if payload.AuthKeyID != "tk_live" {
		t.Fatalf("auth_key_id = %s, want tk_live", payload.AuthKeyID)
	}
	if payload.AcceptedNonce != 7 || payload.NextNonce != 8 {
		t.Fatalf("nonce payload = %+v, want accepted=7 next=8", payload)
	}
	if payload.Scope != "TRADE" || payload.KeyStatus != "ACTIVE" {
		t.Fatalf("scope/status = %s/%s, want TRADE/ACTIVE", payload.Scope, payload.KeyStatus)
	}
	if payload.OrderAuthorization == nil {
		t.Fatalf("expected order authorization witness")
	}
	if !payload.OrderAuthorization.VerifierEligible {
		t.Fatalf("expected verifier-eligible order authorization witness, got %+v", payload.OrderAuthorization)
	}
	if payload.OrderAuthorization.AuthorizationRef != "tk_live:0x5fbe9af9d6ab53d4df3bcb43f9e6c5f26a4d9bc2a8f44a0ab2997f7dc2c5c94a" {
		t.Fatalf("unexpected authorization_ref: %s", payload.OrderAuthorization.AuthorizationRef)
	}
}

func TestBuildNonceAdvanceEntryRejectsMissingSessionID(t *testing.T) {
	_, err := buildNonceAdvanceEntry(dto.SessionResponse{UserID: 1001}, 7, 1775886400000, nil)
	if err == nil {
		t.Fatalf("expected missing session_id to fail")
	}
}

func TestBuildNonceAdvanceEntryMarksLegacyCompatWitnessIneligible(t *testing.T) {
	session := dto.SessionResponse{
		SessionID:        "sess_legacy",
		UserID:           1001,
		WalletAddress:    "0x00000000000000000000000000000000000000aa",
		SessionPublicKey: "0x8f931f3d9d6a93f2b05a1e8ef8356d7408be0f2f5f63c2dbcbf6c227f5f1c5d2",
		Scope:            "trade",
		ChainID:          97,
		SessionNonce:     "sess_legacy_nonce",
		Status:           "active",
	}
	intent := sharedauth.OrderIntent{
		SessionID:         session.SessionID,
		WalletAddress:     session.WalletAddress,
		UserID:            session.UserID,
		MarketID:          88,
		Outcome:           "YES",
		Side:              "BUY",
		OrderType:         "LIMIT",
		TimeInForce:       "GTC",
		Price:             10,
		Quantity:          20,
		ClientOrderID:     "cli-legacy",
		Nonce:             7,
		RequestedAtMillis: 1775886400000,
	}
	witness := sharedauth.BuildOrderAuthorizationWitness(session.UserID, authorizedTradingKeyFromSession(session), intent, "0xfeedface")

	entry, err := buildNonceAdvanceEntry(session, 7, 1775886400000, &witness)
	if err != nil {
		t.Fatalf("buildNonceAdvanceEntry returned error: %v", err)
	}
	payload := entry.Payload.(rollup.NonceAdvancedPayload)
	if payload.OrderAuthorization == nil || payload.OrderAuthorization.VerifierEligible {
		t.Fatalf("expected legacy witness to stay non-verifier-eligible, got %+v", payload.OrderAuthorization)
	}
	if !strings.Contains(payload.OrderAuthorization.IneligibleReason, "/api/v1/sessions") {
		t.Fatalf("unexpected ineligible reason: %s", payload.OrderAuthorization.IneligibleReason)
	}
}
