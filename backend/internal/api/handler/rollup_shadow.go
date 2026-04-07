package handler

import (
	"fmt"
	"strings"

	"funnyoption/internal/api/dto"
	"funnyoption/internal/rollup"
	sharedauth "funnyoption/internal/shared/auth"
)

func buildNonceAdvanceEntry(session dto.SessionResponse, nonce uint64, occurredAtMillis int64, orderAuthorization *sharedauth.OrderAuthorizationWitness) (rollup.JournalAppend, error) {
	sessionID := strings.TrimSpace(session.SessionID)
	if sessionID == "" {
		return rollup.JournalAppend{}, fmt.Errorf("session_id is required for nonce shadow entry")
	}
	if session.UserID <= 0 {
		return rollup.JournalAppend{}, fmt.Errorf("user_id is required for nonce shadow entry")
	}
	if nonce == 0 {
		return rollup.JournalAppend{}, fmt.Errorf("nonce must be positive for nonce shadow entry")
	}
	if nonce == ^uint64(0) {
		return rollup.JournalAppend{}, fmt.Errorf("nonce overflow for nonce shadow entry")
	}

	return rollup.JournalAppend{
		EntryType:        rollup.EntryTypeNonceAdvanced,
		SourceType:       rollup.SourceTypeAPIAuth,
		SourceRef:        fmt.Sprintf("%s:%d", sessionID, nonce),
		OccurredAtMillis: occurredAtMillis,
		Payload: rollup.NonceAdvancedPayload{
			AccountID:          session.UserID,
			AuthKeyID:          sessionID,
			Scope:              strings.ToUpper(strings.TrimSpace(session.Scope)),
			KeyStatus:          strings.ToUpper(strings.TrimSpace(session.Status)),
			AcceptedNonce:      nonce,
			NextNonce:          nonce + 1,
			OccurredAtMillis:   occurredAtMillis,
			OrderAuthorization: orderAuthorization,
		},
	}, nil
}

func buildTradingKeyAuthorizedEntry(session dto.SessionResponse, req dto.RegisterTradingKeyRequest, walletSignature string) (rollup.JournalAppend, error) {
	key := authorizedTradingKeyFromSession(session)
	authorizationRef := key.AuthorizationRef()
	if authorizationRef == "" {
		return rollup.JournalAppend{}, fmt.Errorf("authorization_ref is required for trading key auth witness entry")
	}
	authz := sharedauth.TradingKeyAuthorization{
		WalletAddress:            req.WalletAddress,
		TradingPublicKey:         req.TradingPublicKey,
		TradingKeyScheme:         req.TradingKeyScheme,
		Scope:                    req.Scope,
		Challenge:                req.Challenge,
		ChallengeExpiresAtMillis: req.ChallengeExpiresAtMillis,
		KeyExpiresAtMillis:       req.KeyExpiresAtMillis,
		ChainID:                  req.ChainID,
		VaultAddress:             req.VaultAddress,
	}
	witness, err := sharedauth.BuildTradingKeyAuthorizationWitness(session.UserID, authz, key, walletSignature, session.IssuedAtMillis)
	if err != nil {
		return rollup.JournalAppend{}, err
	}
	return rollup.JournalAppend{
		EntryType:        rollup.EntryTypeTradingKeyAuthorized,
		SourceType:       rollup.SourceTypeAPIAuth,
		SourceRef:        authorizationRef,
		OccurredAtMillis: session.IssuedAtMillis,
		Payload: rollup.TradingKeyAuthorizedPayload{
			AuthorizationWitness: witness,
		},
	}, nil
}

func authorizedTradingKeyFromSession(session dto.SessionResponse) sharedauth.AuthorizedTradingKey {
	return sharedauth.AuthorizedTradingKey{
		TradingKeyID:       strings.TrimSpace(session.SessionID),
		AccountID:          session.UserID,
		WalletAddress:      strings.TrimSpace(session.WalletAddress),
		TradingPublicKey:   strings.TrimSpace(session.SessionPublicKey),
		TradingKeyScheme:   sharedauth.DefaultTradingKeyScheme,
		Scope:              strings.TrimSpace(session.Scope),
		ChainID:            session.ChainID,
		VaultAddress:       strings.TrimSpace(session.VaultAddress),
		Status:             strings.TrimSpace(session.Status),
		ExpiresAtMillis:    session.ExpiresAtMillis,
		AuthorizationNonce: strings.TrimSpace(session.SessionNonce),
	}
}
