package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"funnyoption/internal/rollup"
	sharedkafka "funnyoption/internal/shared/kafka"
)

type SQLStore struct {
	db     *sql.DB
	rollup *rollup.Store
}

type resolutionRecord struct {
	Status          string
	ResolvedOutcome string
	ResolverType    string
	ResolverRef     string
	Evidence        json.RawMessage
}

func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) WithRollup(store *rollup.Store) *SQLStore {
	s.rollup = store
	return s
}

func (s *SQLStore) ApplyDelta(ctx context.Context, marketID, userID int64, outcome, positionAsset string, delta int64) error {
	if err := s.ensureMarket(ctx, marketID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO positions (
			market_id, user_id, outcome, position_asset, quantity, settled_quantity, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, 0, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
		ON CONFLICT (market_id, user_id, outcome) DO UPDATE
		SET position_asset = EXCLUDED.position_asset,
			quantity = positions.quantity + EXCLUDED.quantity,
			updated_at = EXCLUDED.updated_at
	`, marketID, userID, outcome, positionAsset, delta)
	return err
}

func (s *SQLStore) ResolveMarket(ctx context.Context, input ResolveMarketInput) (bool, error) {
	if err := s.ensureMarket(ctx, input.MarketID); err != nil {
		return false, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	result, err := tx.ExecContext(ctx, `
		UPDATE markets
		SET status = 'RESOLVED',
			resolved_outcome = $2,
			updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE market_id = $1
		  AND status <> 'RESOLVED'
	`, input.MarketID, input.ResolvedOutcome)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if rowsAffected == 0 {
		var currentStatus string
		var currentOutcome string
		row := tx.QueryRowContext(ctx, `
			SELECT status, resolved_outcome
			FROM markets
			WHERE market_id = $1
		`, input.MarketID)
		if err := row.Scan(&currentStatus, &currentOutcome); err != nil {
			return false, err
		}
		if strings.ToUpper(strings.TrimSpace(currentStatus)) == "RESOLVED" &&
			strings.ToUpper(strings.TrimSpace(currentOutcome)) == strings.ToUpper(strings.TrimSpace(input.ResolvedOutcome)) {
			if err := tx.Commit(); err != nil {
				return false, err
			}
			return false, nil
		}
		return false, fmt.Errorf("market %d already resolved with outcome %s", input.MarketID, currentOutcome)
	}

	current, err := loadResolutionRecordTx(ctx, tx, input.MarketID)
	if err != nil {
		return false, err
	}
	finalRecord := finalizeResolutionRecord(current, input.ResolvedOutcome)

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO market_resolutions (
			market_id, status, resolved_outcome, resolver_type, resolver_ref, evidence, created_at, updated_at
		)
		VALUES ($1, 'RESOLVED', $2, $3, $4, $5, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
		ON CONFLICT (market_id) DO UPDATE
		SET status = EXCLUDED.status,
			resolved_outcome = EXCLUDED.resolved_outcome,
			resolver_type = EXCLUDED.resolver_type,
			resolver_ref = EXCLUDED.resolver_ref,
			evidence = EXCLUDED.evidence,
			updated_at = EXCLUDED.updated_at
	`, input.MarketID, input.ResolvedOutcome, finalRecord.ResolverType, finalRecord.ResolverRef, normalizeResolutionEvidence(finalRecord.Evidence)); err != nil {
		return false, err
	}
	if s.rollup != nil {
		entry := buildMarketResolvedEntry(normalizeResolveMarketInput(input), finalRecord)
		if err := s.rollup.AppendEntriesTx(ctx, tx, []rollup.JournalAppend{entry}); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func loadResolutionRecordTx(ctx context.Context, tx *sql.Tx, marketID int64) (resolutionRecord, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT status, resolved_outcome, resolver_type, resolver_ref, evidence
		FROM market_resolutions
		WHERE market_id = $1
	`, marketID)
	var record resolutionRecord
	if err := row.Scan(&record.Status, &record.ResolvedOutcome, &record.ResolverType, &record.ResolverRef, &record.Evidence); err != nil {
		if err == sql.ErrNoRows {
			return resolutionRecord{}, nil
		}
		return resolutionRecord{}, err
	}
	return record, nil
}

func finalizeResolutionRecord(current resolutionRecord, outcome string) resolutionRecord {
	final := resolutionRecord{
		Status:          "RESOLVED",
		ResolvedOutcome: outcome,
		ResolverType:    "ADMIN",
		ResolverRef:     "",
		Evidence:        json.RawMessage(`{}`),
	}
	if shouldPreserveOracleOwnership(current, outcome) {
		final.ResolverType = current.ResolverType
		final.ResolverRef = current.ResolverRef
		final.Evidence = normalizeResolutionEvidence(current.Evidence)
	}
	return final
}

func shouldPreserveOracleOwnership(current resolutionRecord, outcome string) bool {
	status := strings.ToUpper(strings.TrimSpace(current.Status))
	resolverType := strings.ToUpper(strings.TrimSpace(current.ResolverType))
	resolvedOutcome := strings.ToUpper(strings.TrimSpace(current.ResolvedOutcome))
	if resolverType != "ORACLE_PRICE" {
		return false
	}
	if status != "OBSERVED" && status != "RESOLVED" {
		return false
	}
	return resolvedOutcome != "" && resolvedOutcome == strings.ToUpper(strings.TrimSpace(outcome))
}

func normalizeResolutionEvidence(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return []byte(`{}`)
	}
	return raw
}

func (s *SQLStore) CancelActiveOrders(ctx context.Context, marketID int64, reason string) ([]cancelledOrder, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		UPDATE orders
		SET status = 'CANCELLED',
			cancel_reason = $2,
			updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE market_id = $1
		  AND status IN ('NEW', 'PARTIALLY_FILLED')
		  AND remaining_quantity > 0
		RETURNING order_id, command_id, client_order_id, user_id, market_id, outcome, side, order_type, time_in_force,
		          collateral_asset, freeze_id, freeze_asset, freeze_amount, price, quantity, filled_quantity, remaining_quantity,
		          status, cancel_reason, updated_at
	`, marketID, strings.ToUpper(strings.TrimSpace(reason)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cancelled := make([]cancelledOrder, 0)
	for rows.Next() {
		var item cancelledOrder
		if err := rows.Scan(
			&item.OrderID,
			&item.CommandID,
			&item.ClientOrderID,
			&item.UserID,
			&item.MarketID,
			&item.Outcome,
			&item.Side,
			&item.OrderType,
			&item.TimeInForce,
			&item.CollateralAsset,
			&item.FreezeID,
			&item.FreezeAsset,
			&item.FreezeAmount,
			&item.Price,
			&item.Quantity,
			&item.FilledQuantity,
			&item.RemainingQuantity,
			&item.Status,
			&item.CancelReason,
			&item.UpdatedAtMillis,
		); err != nil {
			return nil, err
		}
		item.UpdatedAtMillis *= 1000
		cancelled = append(cancelled, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if s.rollup != nil {
		if err := s.rollup.AppendEntriesTx(ctx, tx, buildSettlementCancellationEntries(cancelled)); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return cancelled, nil
}

func (s *SQLStore) WinningPositions(ctx context.Context, marketID int64, outcome string) ([]winningPosition, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT market_id, user_id, outcome, quantity - settled_quantity AS remaining_quantity
		FROM positions
		WHERE market_id = $1
		  AND outcome = $2
		  AND quantity > settled_quantity
		ORDER BY user_id
	`, marketID, outcome)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []winningPosition
	for rows.Next() {
		var item winningPosition
		if err := rows.Scan(&item.MarketID, &item.UserID, &item.Outcome, &item.Quantity); err != nil {
			return nil, err
		}
		positions = append(positions, item)
	}
	return positions, rows.Err()
}

func (s *SQLStore) MarkSettled(ctx context.Context, event sharedkafka.SettlementCompletedEvent) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	positionAsset := strings.ToUpper(strings.TrimSpace(event.PositionAsset))
	if positionAsset == "" {
		positionAsset = fmt.Sprintf("POSITION:%d:%s", event.MarketID, strings.ToUpper(strings.TrimSpace(event.WinningOutcome)))
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO settlement_payouts (
			event_id, market_id, user_id, winning_outcome, position_asset,
			settled_quantity, payout_asset, payout_amount, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'COMPLETED', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
		ON CONFLICT (event_id) DO NOTHING
	`, strings.TrimSpace(event.EventID), event.MarketID, event.UserID, strings.ToUpper(strings.TrimSpace(event.WinningOutcome)), positionAsset, event.SettledQuantity, strings.ToUpper(strings.TrimSpace(event.PayoutAsset)), event.PayoutAmount)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return tx.Commit()
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE positions
		SET settled_quantity = settled_quantity + $4,
			updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE market_id = $1 AND user_id = $2 AND outcome = $3
	`, event.MarketID, event.UserID, strings.ToUpper(strings.TrimSpace(event.WinningOutcome)), event.SettledQuantity)
	if err != nil {
		return err
	}
	if s.rollup != nil {
		if err := s.rollup.AppendEntriesTx(ctx, tx, []rollup.JournalAppend{buildSettlementPayoutEntry(normalizeSettlementEvent(event, positionAsset))}); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLStore) ensureMarket(ctx context.Context, marketID int64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO markets (
			market_id, title, description, collateral_asset, status,
			open_at, close_at, resolve_at, resolved_outcome, created_by, metadata, created_at, updated_at
		)
		VALUES ($1, $2, '', 'USDT', 'OPEN', 0, 0, 0, '', 0, '{}'::jsonb, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
		ON CONFLICT (market_id) DO NOTHING
	`, marketID, fmt.Sprintf("Market %d", marketID))
	return err
}

func normalizeResolveMarketInput(input ResolveMarketInput) ResolveMarketInput {
	input.ResolvedOutcome = strings.ToUpper(strings.TrimSpace(input.ResolvedOutcome))
	if input.OccurredAtMillis <= 0 {
		input.OccurredAtMillis = time.Now().UnixMilli()
	}
	return input
}

func normalizeSettlementEvent(event sharedkafka.SettlementCompletedEvent, positionAsset string) sharedkafka.SettlementCompletedEvent {
	event.EventID = strings.TrimSpace(event.EventID)
	event.WinningOutcome = strings.ToUpper(strings.TrimSpace(event.WinningOutcome))
	event.PositionAsset = positionAsset
	event.PayoutAsset = strings.ToUpper(strings.TrimSpace(event.PayoutAsset))
	if event.PayoutAsset == "" {
		event.PayoutAsset = "USDT"
	}
	if event.OccurredAtMillis <= 0 {
		event.OccurredAtMillis = time.Now().UnixMilli()
	}
	return event
}
