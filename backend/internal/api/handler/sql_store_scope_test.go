package handler

import (
	"context"
	"database/sql"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"funnyoption/internal/api/dto"
)

func TestSQLStoreRegisterTradingKeyScopesByVaultEvenWithSamePublicKey(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("FUNNYOPTION_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("FUNNYOPTION_POSTGRES_DSN is required for SQL store scope integration coverage")
	}

	ctx := context.Background()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("PingContext returned error: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		CREATE TEMP TABLE wallet_sessions (
			session_id         VARCHAR(64) PRIMARY KEY,
			user_id            BIGINT NOT NULL DEFAULT 0,
			wallet_address     VARCHAR(64) NOT NULL,
			session_public_key VARCHAR(256) NOT NULL,
			scope              VARCHAR(32) NOT NULL DEFAULT 'TRADE',
			chain_id           BIGINT NOT NULL DEFAULT 0,
			vault_address      VARCHAR(128) NOT NULL DEFAULT '',
			session_nonce      VARCHAR(64) NOT NULL DEFAULT '',
			last_order_nonce   BIGINT NOT NULL DEFAULT 0,
			status             VARCHAR(16) NOT NULL DEFAULT 'ACTIVE',
			issued_at          BIGINT NOT NULL DEFAULT 0,
			expires_at         BIGINT NOT NULL DEFAULT 0,
			revoked_at         BIGINT NOT NULL DEFAULT 0,
			created_at         BIGINT NOT NULL DEFAULT 0,
			updated_at         BIGINT NOT NULL DEFAULT 0,
			UNIQUE (wallet_address, chain_id, vault_address, session_public_key)
		);
		CREATE TEMP TABLE trading_key_challenges (
			challenge_id   VARCHAR(64) PRIMARY KEY,
			wallet_address VARCHAR(64) NOT NULL,
			chain_id       BIGINT NOT NULL DEFAULT 0,
			vault_address  VARCHAR(128) NOT NULL DEFAULT '',
			challenge      VARCHAR(128) NOT NULL,
			expires_at     BIGINT NOT NULL DEFAULT 0,
			consumed_at    BIGINT NOT NULL DEFAULT 0,
			created_at     BIGINT NOT NULL DEFAULT 0,
			updated_at     BIGINT NOT NULL DEFAULT 0,
			UNIQUE (wallet_address, chain_id, vault_address, challenge)
		);
		CREATE TEMP TABLE user_profiles (
			user_id        BIGINT PRIMARY KEY,
			wallet_address VARCHAR(64) NOT NULL UNIQUE,
			display_name   VARCHAR(64) NOT NULL DEFAULT '',
			avatar_preset  VARCHAR(32) NOT NULL DEFAULT '',
			created_at     BIGINT NOT NULL DEFAULT 0,
			updated_at     BIGINT NOT NULL DEFAULT 0
		);
	`); err != nil {
		t.Fatalf("create temp tables returned error: %v", err)
	}

	store := NewSQLStore(db)
	const (
		walletAddress    = "0x00000000000000000000000000000000000000aa"
		vaultA           = "0x00000000000000000000000000000000000000b1"
		vaultB           = "0x00000000000000000000000000000000000000b2"
		sharedPublicKey  = "0xaaa1"
		rotatedPublicKey = "0xaaa2"
		chainID          = int64(97)
	)

	firstVaultA := mustRegisterTradingKeyForVault(t, ctx, store, dto.RegisterTradingKeyRequest{
		SessionID:          "tk_scope_a_1",
		WalletAddress:      walletAddress,
		ChainID:            chainID,
		VaultAddress:       vaultA,
		TradingPublicKey:   sharedPublicKey,
		TradingKeyScheme:   "ED25519",
		Scope:              "TRADE",
		KeyExpiresAtMillis: 0,
	}, "tkc_scope_a_1", "0x1111111111111111111111111111111111111111111111111111111111111111")

	firstVaultB := mustRegisterTradingKeyForVault(t, ctx, store, dto.RegisterTradingKeyRequest{
		SessionID:          "tk_scope_b_1",
		WalletAddress:      walletAddress,
		ChainID:            chainID,
		VaultAddress:       vaultB,
		TradingPublicKey:   sharedPublicKey,
		TradingKeyScheme:   "ED25519",
		Scope:              "TRADE",
		KeyExpiresAtMillis: 0,
	}, "tkc_scope_b_1", "0x2222222222222222222222222222222222222222222222222222222222222222")

	if firstVaultA.SessionPublicKey != strings.ToLower(sharedPublicKey) {
		t.Fatalf("first vault A public key = %s, want %s", firstVaultA.SessionPublicKey, strings.ToLower(sharedPublicKey))
	}
	if firstVaultB.SessionPublicKey != strings.ToLower(sharedPublicKey) {
		t.Fatalf("first vault B public key = %s, want %s", firstVaultB.SessionPublicKey, strings.ToLower(sharedPublicKey))
	}

	var (
		activeSameKeyRows   int
		activeSameKeyVaults int
	)
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*), COUNT(DISTINCT vault_address)
		FROM wallet_sessions
		WHERE wallet_address = $1
		  AND chain_id = $2
		  AND session_public_key = $3
		  AND status = 'ACTIVE'
	`, strings.ToLower(walletAddress), chainID, strings.ToLower(sharedPublicKey)).Scan(&activeSameKeyRows, &activeSameKeyVaults); err != nil {
		t.Fatalf("count same-public-key active rows returned error: %v", err)
	}
	if activeSameKeyRows != 2 || activeSameKeyVaults != 2 {
		t.Fatalf("same-public-key active footprint = %d rows across %d vaults, want 2 rows across 2 vaults", activeSameKeyRows, activeSameKeyVaults)
	}

	activeAcrossVaults, err := store.ListSessions(ctx, dto.ListSessionsRequest{
		WalletAddress: walletAddress,
		Status:        "ACTIVE",
		Limit:         20,
	})
	if err != nil {
		t.Fatalf("ListSessions active across vaults returned error: %v", err)
	}
	if got, want := sortedSessionIDs(activeAcrossVaults), []string{firstVaultA.SessionID, firstVaultB.SessionID}; !equalStrings(got, want) {
		t.Fatalf("active session ids = %v, want %v", got, want)
	}

	secondVaultA := mustRegisterTradingKeyForVault(t, ctx, store, dto.RegisterTradingKeyRequest{
		SessionID:          "tk_scope_a_2",
		WalletAddress:      walletAddress,
		ChainID:            chainID,
		VaultAddress:       vaultA,
		TradingPublicKey:   rotatedPublicKey,
		TradingKeyScheme:   "ED25519",
		Scope:              "TRADE",
		KeyExpiresAtMillis: 0,
	}, "tkc_scope_a_2", "0x3333333333333333333333333333333333333333333333333333333333333333")

	rotatedVaultA, err := store.GetSession(ctx, firstVaultA.SessionID)
	if err != nil {
		t.Fatalf("GetSession first vault A returned error: %v", err)
	}
	if rotatedVaultA.Status != "ROTATED" {
		t.Fatalf("first vault A status = %s, want ROTATED", rotatedVaultA.Status)
	}
	if rotatedVaultA.VaultAddress != strings.ToLower(vaultA) {
		t.Fatalf("first vault A vault = %s, want %s", rotatedVaultA.VaultAddress, strings.ToLower(vaultA))
	}

	stillActiveVaultB, err := store.GetSession(ctx, firstVaultB.SessionID)
	if err != nil {
		t.Fatalf("GetSession first vault B returned error: %v", err)
	}
	if stillActiveVaultB.Status != "ACTIVE" {
		t.Fatalf("first vault B status = %s, want ACTIVE", stillActiveVaultB.Status)
	}
	if stillActiveVaultB.VaultAddress != strings.ToLower(vaultB) {
		t.Fatalf("first vault B vault = %s, want %s", stillActiveVaultB.VaultAddress, strings.ToLower(vaultB))
	}

	vaultAHistory, err := store.ListSessions(ctx, dto.ListSessionsRequest{
		WalletAddress: walletAddress,
		VaultAddress:  vaultA,
		Limit:         20,
	})
	if err != nil {
		t.Fatalf("ListSessions vault A history returned error: %v", err)
	}
	if got, want := sortedSessionIDs(vaultAHistory), []string{firstVaultA.SessionID, secondVaultA.SessionID}; !equalStrings(got, want) {
		t.Fatalf("vault A session ids = %v, want %v", got, want)
	}
	for _, item := range vaultAHistory {
		if item.VaultAddress != strings.ToLower(vaultA) {
			t.Fatalf("vault A history contained wrong vault %s", item.VaultAddress)
		}
	}

	activeVaultA, err := store.ListSessions(ctx, dto.ListSessionsRequest{
		WalletAddress: walletAddress,
		VaultAddress:  vaultA,
		Status:        "ACTIVE",
		Limit:         20,
	})
	if err != nil {
		t.Fatalf("ListSessions active vault A returned error: %v", err)
	}
	if len(activeVaultA) != 1 || activeVaultA[0].SessionID != secondVaultA.SessionID {
		t.Fatalf("active vault A sessions = %+v, want only %s", activeVaultA, secondVaultA.SessionID)
	}

	activeVaultB, err := store.ListSessions(ctx, dto.ListSessionsRequest{
		WalletAddress: walletAddress,
		VaultAddress:  vaultB,
		Status:        "ACTIVE",
		Limit:         20,
	})
	if err != nil {
		t.Fatalf("ListSessions active vault B returned error: %v", err)
	}
	if len(activeVaultB) != 1 || activeVaultB[0].SessionID != firstVaultB.SessionID {
		t.Fatalf("active vault B sessions = %+v, want only %s", activeVaultB, firstVaultB.SessionID)
	}
}

func mustRegisterTradingKeyForVault(
	t *testing.T,
	ctx context.Context,
	store *SQLStore,
	req dto.RegisterTradingKeyRequest,
	challengeID string,
	challenge string,
) dto.SessionResponse {
	t.Helper()

	expiresAt := time.Now().Add(5 * time.Minute).UnixMilli()
	if _, err := store.CreateTradingKeyChallenge(ctx, dto.CreateTradingKeyChallengeRequest{
		ChallengeID:        challengeID,
		Challenge:          challenge,
		ChallengeExpiresAt: expiresAt,
		WalletAddress:      req.WalletAddress,
		ChainID:            req.ChainID,
		VaultAddress:       req.VaultAddress,
	}); err != nil {
		t.Fatalf("CreateTradingKeyChallenge(%s) returned error: %v", challengeID, err)
	}

	req.ChallengeID = challengeID
	req.Challenge = challenge
	req.ChallengeExpiresAtMillis = expiresAt

	session, err := store.RegisterTradingKey(ctx, req)
	if err != nil {
		t.Fatalf("RegisterTradingKey(%s) returned error: %v", req.SessionID, err)
	}
	return session
}

func sortedSessionIDs(items []dto.SessionResponse) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.SessionID)
	}
	sort.Strings(ids)
	return ids
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
