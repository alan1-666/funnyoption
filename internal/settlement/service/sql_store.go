package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type SQLStore struct {
	db *sql.DB
}

func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
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

func (s *SQLStore) ResolveMarket(ctx context.Context, marketID int64, outcome string) error {
	if err := s.ensureMarket(ctx, marketID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO market_resolutions (
			market_id, status, resolved_outcome, resolver_type, resolver_ref, evidence, created_at, updated_at
		)
		VALUES ($1, 'RESOLVED', $2, 'ADMIN', '', '{}'::jsonb, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
		ON CONFLICT (market_id) DO UPDATE
		SET status = EXCLUDED.status,
			resolved_outcome = EXCLUDED.resolved_outcome,
			updated_at = EXCLUDED.updated_at
	`, marketID, outcome)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE markets
		SET status = 'RESOLVED',
			resolved_outcome = $2,
			updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE market_id = $1
	`, marketID, outcome)
	return err
}

func (s *SQLStore) CancelActiveOrders(ctx context.Context, marketID int64, reason string) ([]cancelledOrder, error) {
	rows, err := s.db.QueryContext(ctx, `
		UPDATE orders
		SET status = 'CANCELLED',
			cancel_reason = $2,
			updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE market_id = $1
		  AND status IN ('NEW', 'PARTIALLY_FILLED')
		  AND remaining_quantity > 0
		RETURNING order_id, command_id, client_order_id, user_id, market_id, outcome, side, order_type, time_in_force,
		          collateral_asset, freeze_id, freeze_asset, freeze_amount, price, quantity, filled_quantity, remaining_quantity,
		          status, cancel_reason
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
		); err != nil {
			return nil, err
		}
		cancelled = append(cancelled, item)
	}
	return cancelled, rows.Err()
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

func (s *SQLStore) MarkSettled(ctx context.Context, eventID string, marketID, userID int64, outcome string, quantity int64, payoutAsset string, payoutAmount int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	positionAsset := fmt.Sprintf("POSITION:%d:%s", marketID, outcome)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO settlement_payouts (
			event_id, market_id, user_id, winning_outcome, position_asset,
			settled_quantity, payout_asset, payout_amount, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'COMPLETED', EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
		ON CONFLICT (event_id) DO NOTHING
	`, eventID, marketID, userID, outcome, positionAsset, quantity, payoutAsset, payoutAmount)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE positions
		SET settled_quantity = settled_quantity + $4,
			updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE market_id = $1 AND user_id = $2 AND outcome = $3
	`, marketID, userID, outcome, quantity)
	if err != nil {
		return err
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
