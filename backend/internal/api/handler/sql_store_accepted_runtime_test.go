package handler

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	"funnyoption/internal/api/dto"
)

func TestSQLStoreAcceptedReadTruthVisibleUsesAcceptedMirrorsBeforeFreeze(t *testing.T) {
	store, ctx := mustSetupAcceptedRuntimeStore(t, false, "CLAIMABLE")

	balances, err := store.ListBalances(ctx, dto.ListBalancesRequest{UserID: 1001, Limit: 20})
	if err != nil {
		t.Fatalf("ListBalances returned error: %v", err)
	}
	if len(balances) != 1 || balances[0].Available != 1390 || balances[0].Frozen != 0 || balances[0].Asset != "USDT" {
		t.Fatalf("unexpected accepted balances: %+v", balances)
	}

	positions, err := store.ListPositions(ctx, dto.ListPositionsRequest{UserID: 1001, Limit: 20})
	if err != nil {
		t.Fatalf("ListPositions returned error: %v", err)
	}
	if len(positions) != 1 || positions[0].MarketID != 88 || positions[0].Quantity != 6 || positions[0].SettledQuantity != 6 {
		t.Fatalf("unexpected accepted positions: %+v", positions)
	}

	payouts, err := store.ListPayouts(ctx, dto.ListPayoutsRequest{UserID: 1001, Limit: 20})
	if err != nil {
		t.Fatalf("ListPayouts returned error: %v", err)
	}
	if len(payouts) != 1 || payouts[0].EventID != "evt_settlement_88_1" || payouts[0].PayoutAmount != 600 {
		t.Fatalf("unexpected accepted payouts: %+v", payouts)
	}

	liabilities, err := store.BuildLiabilityReport(ctx)
	if err != nil {
		t.Fatalf("BuildLiabilityReport returned error: %v", err)
	}
	if len(liabilities) != 1 {
		t.Fatalf("expected 1 accepted liability line, got %+v", liabilities)
	}
	if liabilities[0].Asset != "USDT" || liabilities[0].UserAvailable != 1390 || liabilities[0].PendingWithdraw != 40 {
		t.Fatalf("unexpected accepted liability line: %+v", liabilities[0])
	}

	claimTx, err := store.CreateClaimRequest(ctx, dto.ClaimPayoutRequest{
		EventID:       "evt_settlement_88_1",
		UserID:        1001,
		WalletAddress: "0x0000000000000000000000000000000000000abc",
	})
	if err != nil {
		t.Fatalf("CreateClaimRequest returned error: %v", err)
	}
	if claimTx.RefID != "evt_settlement_88_1" || claimTx.BizType != "CLAIM" || claimTx.Status != "PENDING" {
		t.Fatalf("unexpected accepted claim transaction: %+v", claimTx)
	}
}

func TestSQLStoreFrozenAcceptedReadTruthZerosClaimedEscapeCollateral(t *testing.T) {
	store, ctx := mustSetupAcceptedRuntimeStore(t, true, "CLAIMED")

	balances, err := store.ListBalances(ctx, dto.ListBalancesRequest{UserID: 1001, Limit: 20})
	if err != nil {
		t.Fatalf("ListBalances returned error: %v", err)
	}
	if len(balances) != 1 || balances[0].Available != 0 || balances[0].Frozen != 0 {
		t.Fatalf("unexpected frozen accepted balances: %+v", balances)
	}

	liabilities, err := store.BuildLiabilityReport(ctx)
	if err != nil {
		t.Fatalf("BuildLiabilityReport returned error: %v", err)
	}
	if len(liabilities) != 1 {
		t.Fatalf("expected 1 liability line, got %+v", liabilities)
	}
	if liabilities[0].UserAvailable != 0 || liabilities[0].UserFrozen != 0 || liabilities[0].PendingWithdraw != 40 {
		t.Fatalf("unexpected frozen accepted liability line: %+v", liabilities[0])
	}
}

func mustSetupAcceptedRuntimeStore(t *testing.T, frozen bool, escapeClaimStatus string) (*SQLStore, context.Context) {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv("FUNNYOPTION_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("FUNNYOPTION_POSTGRES_DSN is required for accepted read-truth integration coverage")
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
		CREATE TEMP TABLE rollup_accepted_batches (
			batch_id BIGINT PRIMARY KEY,
			submission_id VARCHAR(64) NOT NULL,
			encoding_version VARCHAR(32) NOT NULL,
			first_sequence_no BIGINT NOT NULL DEFAULT 0,
			last_sequence_no BIGINT NOT NULL DEFAULT 0,
			entry_count INTEGER NOT NULL DEFAULT 0,
			batch_data_hash VARCHAR(66) NOT NULL DEFAULT '',
			prev_state_root VARCHAR(66) NOT NULL DEFAULT '',
			balances_root VARCHAR(66) NOT NULL DEFAULT '',
			orders_root VARCHAR(66) NOT NULL DEFAULT '',
			positions_funding_root VARCHAR(66) NOT NULL DEFAULT '',
			withdrawals_root VARCHAR(66) NOT NULL DEFAULT '',
			next_state_root VARCHAR(66) NOT NULL DEFAULT '',
			record_tx_hash VARCHAR(66) NOT NULL DEFAULT '',
			accept_tx_hash VARCHAR(66) NOT NULL DEFAULT '',
			accepted_at BIGINT NOT NULL DEFAULT 0,
			created_at BIGINT NOT NULL DEFAULT 0,
			updated_at BIGINT NOT NULL DEFAULT 0
		);
		CREATE TEMP TABLE rollup_accepted_balances (
			batch_id BIGINT NOT NULL,
			account_id BIGINT NOT NULL,
			asset VARCHAR(64) NOT NULL,
			available BIGINT NOT NULL DEFAULT 0,
			frozen BIGINT NOT NULL DEFAULT 0,
			sequence_no BIGINT NOT NULL DEFAULT 0,
			created_at BIGINT NOT NULL DEFAULT 0,
			updated_at BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY (account_id, asset)
		);
		CREATE TEMP TABLE rollup_accepted_positions (
			batch_id BIGINT NOT NULL,
			account_id BIGINT NOT NULL,
			market_id BIGINT NOT NULL,
			outcome VARCHAR(16) NOT NULL,
			position_asset VARCHAR(128) NOT NULL,
			quantity BIGINT NOT NULL DEFAULT 0,
			settled_quantity BIGINT NOT NULL DEFAULT 0,
			settlement_status VARCHAR(32) NOT NULL DEFAULT 'OPEN',
			sequence_no BIGINT NOT NULL DEFAULT 0,
			created_at BIGINT NOT NULL DEFAULT 0,
			updated_at BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY (account_id, market_id, outcome)
		);
		CREATE TEMP TABLE rollup_accepted_payouts (
			event_id VARCHAR(96) PRIMARY KEY,
			batch_id BIGINT NOT NULL,
			market_id BIGINT NOT NULL,
			user_id BIGINT NOT NULL,
			winning_outcome VARCHAR(16) NOT NULL,
			position_asset VARCHAR(128) NOT NULL,
			settled_quantity BIGINT NOT NULL DEFAULT 0,
			payout_asset VARCHAR(64) NOT NULL,
			payout_amount BIGINT NOT NULL DEFAULT 0,
			status VARCHAR(32) NOT NULL DEFAULT 'COMPLETED',
			created_at BIGINT NOT NULL DEFAULT 0,
			updated_at BIGINT NOT NULL DEFAULT 0
		);
		CREATE TEMP TABLE rollup_accepted_withdrawals (
			withdrawal_id VARCHAR(96) PRIMARY KEY,
			batch_id BIGINT NOT NULL,
			account_id BIGINT NOT NULL,
			wallet_address VARCHAR(66) NOT NULL DEFAULT '',
			recipient_address VARCHAR(66) NOT NULL DEFAULT '',
			vault_address VARCHAR(66) NOT NULL DEFAULT '',
			asset VARCHAR(64) NOT NULL,
			amount BIGINT NOT NULL DEFAULT 0,
			lane VARCHAR(32) NOT NULL DEFAULT '',
			chain_name VARCHAR(32) NOT NULL DEFAULT '',
			network_name VARCHAR(32) NOT NULL DEFAULT '',
			request_sequence BIGINT NOT NULL DEFAULT 0,
			claim_id VARCHAR(96) NOT NULL DEFAULT '',
			claim_status VARCHAR(32) NOT NULL DEFAULT 'CLAIMABLE',
			claim_tx_hash VARCHAR(66) NOT NULL DEFAULT '',
			claim_submitted_at BIGINT NOT NULL DEFAULT 0,
			claimed_at BIGINT NOT NULL DEFAULT 0,
			last_error TEXT NOT NULL DEFAULT '',
			last_error_at BIGINT NOT NULL DEFAULT 0,
			created_at BIGINT NOT NULL DEFAULT 0,
			updated_at BIGINT NOT NULL DEFAULT 0
		);
		CREATE TEMP TABLE rollup_accepted_escape_roots (
			batch_id BIGINT PRIMARY KEY,
			state_root VARCHAR(66) NOT NULL DEFAULT '',
			collateral_asset VARCHAR(64) NOT NULL DEFAULT '',
			merkle_root VARCHAR(66) NOT NULL DEFAULT '',
			leaf_count BIGINT NOT NULL DEFAULT 0,
			total_amount BIGINT NOT NULL DEFAULT 0,
			anchor_status VARCHAR(32) NOT NULL DEFAULT 'ANCHORED',
			anchor_tx_hash VARCHAR(66) NOT NULL DEFAULT '',
			anchor_submitted_at BIGINT NOT NULL DEFAULT 0,
			anchored_at BIGINT NOT NULL DEFAULT 0,
			last_error TEXT NOT NULL DEFAULT '',
			last_error_at BIGINT NOT NULL DEFAULT 0,
			created_at BIGINT NOT NULL DEFAULT 0,
			updated_at BIGINT NOT NULL DEFAULT 0
		);
		CREATE TEMP TABLE rollup_accepted_escape_leaves (
			claim_id VARCHAR(96) PRIMARY KEY,
			batch_id BIGINT NOT NULL,
			account_id BIGINT NOT NULL,
			wallet_address VARCHAR(66) NOT NULL DEFAULT '',
			collateral_asset VARCHAR(64) NOT NULL DEFAULT '',
			claim_amount BIGINT NOT NULL DEFAULT 0,
			leaf_index BIGINT NOT NULL DEFAULT 0,
			leaf_hash VARCHAR(66) NOT NULL DEFAULT '',
			proof_hashes JSONB NOT NULL DEFAULT '[]'::jsonb,
			claim_status VARCHAR(32) NOT NULL DEFAULT 'CLAIMABLE',
			claim_tx_hash VARCHAR(66) NOT NULL DEFAULT '',
			claim_submitted_at BIGINT NOT NULL DEFAULT 0,
			claimed_at BIGINT NOT NULL DEFAULT 0,
			last_error TEXT NOT NULL DEFAULT '',
			last_error_at BIGINT NOT NULL DEFAULT 0,
			created_at BIGINT NOT NULL DEFAULT 0,
			updated_at BIGINT NOT NULL DEFAULT 0
		);
		CREATE TEMP TABLE rollup_freeze_state (
			id BOOLEAN PRIMARY KEY DEFAULT TRUE,
			frozen BOOLEAN NOT NULL DEFAULT FALSE,
			frozen_at BIGINT NOT NULL DEFAULT 0,
			request_id BIGINT NOT NULL DEFAULT 0,
			updated_at BIGINT NOT NULL DEFAULT 0
		);
		CREATE TEMP TABLE chain_transactions (
			id BIGSERIAL PRIMARY KEY,
			biz_type VARCHAR(32) NOT NULL,
			ref_id VARCHAR(96) NOT NULL,
			chain_name VARCHAR(32) NOT NULL DEFAULT '',
			network_name VARCHAR(32) NOT NULL DEFAULT '',
			wallet_address VARCHAR(66) NOT NULL DEFAULT '',
			tx_hash VARCHAR(66) NOT NULL DEFAULT '',
			status VARCHAR(32) NOT NULL DEFAULT '',
			payload JSONB NOT NULL DEFAULT '{}'::jsonb,
			error_message TEXT NOT NULL DEFAULT '',
			attempt_count INTEGER NOT NULL DEFAULT 0,
			created_at BIGINT NOT NULL DEFAULT 0,
			updated_at BIGINT NOT NULL DEFAULT 0
		);
		CREATE TEMP TABLE account_balances (
			user_id BIGINT NOT NULL,
			asset VARCHAR(64) NOT NULL,
			available BIGINT NOT NULL DEFAULT 0,
			frozen BIGINT NOT NULL DEFAULT 0,
			created_at BIGINT NOT NULL DEFAULT 0,
			updated_at BIGINT NOT NULL DEFAULT 0
		);
		CREATE TEMP TABLE positions (
			market_id BIGINT NOT NULL,
			user_id BIGINT NOT NULL,
			outcome VARCHAR(16) NOT NULL,
			position_asset VARCHAR(128) NOT NULL,
			quantity BIGINT NOT NULL DEFAULT 0,
			settled_quantity BIGINT NOT NULL DEFAULT 0,
			created_at BIGINT NOT NULL DEFAULT 0,
			updated_at BIGINT NOT NULL DEFAULT 0
		);
		CREATE TEMP TABLE settlement_payouts (
			event_id VARCHAR(96) PRIMARY KEY,
			market_id BIGINT NOT NULL,
			user_id BIGINT NOT NULL,
			winning_outcome VARCHAR(16) NOT NULL,
			position_asset VARCHAR(128) NOT NULL,
			settled_quantity BIGINT NOT NULL DEFAULT 0,
			payout_asset VARCHAR(64) NOT NULL,
			payout_amount BIGINT NOT NULL DEFAULT 0,
			status VARCHAR(32) NOT NULL DEFAULT 'COMPLETED',
			created_at BIGINT NOT NULL DEFAULT 0,
			updated_at BIGINT NOT NULL DEFAULT 0
		);
		CREATE TEMP TABLE chain_withdrawals (
			withdrawal_id VARCHAR(96) PRIMARY KEY,
			user_id BIGINT NOT NULL,
			wallet_address VARCHAR(66) NOT NULL DEFAULT '',
			recipient_address VARCHAR(66) NOT NULL DEFAULT '',
			vault_address VARCHAR(66) NOT NULL DEFAULT '',
			asset VARCHAR(64) NOT NULL,
			amount BIGINT NOT NULL DEFAULT 0,
			chain_name VARCHAR(32) NOT NULL DEFAULT '',
			network_name VARCHAR(32) NOT NULL DEFAULT '',
			tx_hash VARCHAR(66) NOT NULL DEFAULT '',
			log_index BIGINT NOT NULL DEFAULT 0,
			block_number BIGINT NOT NULL DEFAULT 0,
			status VARCHAR(32) NOT NULL DEFAULT '',
			debited_at BIGINT NOT NULL DEFAULT 0,
			created_at BIGINT NOT NULL DEFAULT 0,
			updated_at BIGINT NOT NULL DEFAULT 0
		);
	`); err != nil {
		t.Fatalf("create temp accepted tables returned error: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO rollup_accepted_batches (
			batch_id, submission_id, encoding_version, accepted_at, created_at, updated_at
		)
		VALUES (2, 'rsub_2', 'shadow-submit-v1', 1775886401, 1775886401, 1775886401);
		INSERT INTO rollup_accepted_balances (
			batch_id, account_id, asset, available, frozen, sequence_no, created_at, updated_at
		)
		VALUES (2, 1001, 'USDT', 1390, 0, 8, 1775886401, 1775886401);
		INSERT INTO rollup_accepted_positions (
			batch_id, account_id, market_id, outcome, position_asset, quantity, settled_quantity, settlement_status, sequence_no, created_at, updated_at
		)
		VALUES (2, 1001, 88, 'YES', 'POSITION:88:YES', 6, 6, 'SETTLED', 8, 1775886401, 1775886401);
		INSERT INTO rollup_accepted_payouts (
			event_id, batch_id, market_id, user_id, winning_outcome, position_asset, settled_quantity, payout_asset, payout_amount, status, created_at, updated_at
		)
		VALUES ('evt_settlement_88_1', 2, 88, 1001, 'YES', 'POSITION:88:YES', 6, 'USDT', 600, 'COMPLETED', 1775886401, 1775886401);
		INSERT INTO rollup_accepted_withdrawals (
			withdrawal_id, batch_id, account_id, wallet_address, recipient_address, vault_address, asset, amount, lane, chain_name, network_name, request_sequence, claim_id, claim_status, created_at, updated_at
		)
		VALUES ('wd_1', 2, 1001, '0x0000000000000000000000000000000000000abc', '0x0000000000000000000000000000000000000abc', '0x0000000000000000000000000000000000000def', 'USDT', 40, 'SLOW', 'bsc', 'testnet', 8, 'claim_wd_1', 'CLAIMABLE', 1775886401, 1775886401);
		INSERT INTO rollup_accepted_escape_roots (
			batch_id, state_root, collateral_asset, merkle_root, leaf_count, total_amount, anchor_status, anchored_at, created_at, updated_at
		)
		VALUES (2, '0xstate', 'USDT', '0xroot', 1, 1390, 'ANCHORED', 1775886401, 1775886401, 1775886401);
		INSERT INTO rollup_accepted_escape_leaves (
			claim_id, batch_id, account_id, wallet_address, collateral_asset, claim_amount, leaf_index, leaf_hash, proof_hashes, claim_status, created_at, updated_at
		)
		VALUES ('esc_1', 2, 1001, '0x0000000000000000000000000000000000000abc', 'USDT', 1390, 0, '0xleaf', '[]'::jsonb, $1, 1775886401, 1775886401);
		INSERT INTO rollup_freeze_state (id, frozen, frozen_at, request_id, updated_at)
		VALUES (TRUE, $2, CASE WHEN $2 THEN 1775886401 ELSE 0 END, CASE WHEN $2 THEN 9 ELSE 0 END, 1775886401);

		INSERT INTO account_balances (user_id, asset, available, frozen, created_at, updated_at)
		VALUES (1001, 'USDT', 5, 7, 1775886300, 1775886300);
		INSERT INTO positions (market_id, user_id, outcome, position_asset, quantity, settled_quantity, created_at, updated_at)
		VALUES (88, 1001, 'YES', 'POSITION:88:YES', 1, 0, 1775886300, 1775886300);
		INSERT INTO settlement_payouts (
			event_id, market_id, user_id, winning_outcome, position_asset, settled_quantity, payout_asset, payout_amount, status, created_at, updated_at
		)
		VALUES ('evt_settlement_88_1', 88, 1001, 'YES', 'POSITION:88:YES', 1, 'USDT', 50, 'COMPLETED', 1775886300, 1775886300);
		INSERT INTO chain_withdrawals (
			withdrawal_id, user_id, wallet_address, recipient_address, vault_address, asset, amount, chain_name, network_name, tx_hash, log_index, block_number, status, debited_at, created_at, updated_at
		)
		VALUES ('live_wd_1', 1001, '0x0000000000000000000000000000000000000abc', '0x0000000000000000000000000000000000000abc', '0x0000000000000000000000000000000000000def', 'USDT', 999, 'bsc', 'testnet', '', 0, 0, 'DEBITED', 1775886300, 1775886300, 1775886300);
	`, escapeClaimStatus, frozen); err != nil {
		t.Fatalf("seed accepted tables returned error: %v", err)
	}

	return NewSQLStore(db), ctx
}
