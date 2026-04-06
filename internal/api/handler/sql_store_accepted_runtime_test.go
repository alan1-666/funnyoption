package handler

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	"funnyoption/internal/api/dto"
)

func TestSQLStoreAcceptedReadTruthPrefersAcceptedMirrors(t *testing.T) {
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
	`); err != nil {
		t.Fatalf("seed accepted tables returned error: %v", err)
	}

	store := NewSQLStore(db)

	balances, err := store.ListBalances(ctx, dto.ListBalancesRequest{UserID: 1001, Limit: 20})
	if err != nil {
		t.Fatalf("ListBalances returned error: %v", err)
	}
	if len(balances) != 1 || balances[0].Available != 1390 || balances[0].Asset != "USDT" {
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
}
