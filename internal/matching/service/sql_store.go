package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"funnyoption/internal/matching/engine"
	"funnyoption/internal/matching/model"
	sharedkafka "funnyoption/internal/shared/kafka"
)

type SQLStore struct {
	db *sql.DB
}

func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) MaxTradeSequence(ctx context.Context) (uint64, error) {
	var sequence sql.NullInt64
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence_no), 0) FROM trades`).Scan(&sequence); err != nil {
		return 0, err
	}
	if !sequence.Valid || sequence.Int64 <= 0 {
		return 0, nil
	}
	return uint64(sequence.Int64), nil
}

func (s *SQLStore) LoadRestingOrders(ctx context.Context) ([]*model.Order, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT orders.order_id, orders.client_order_id, orders.user_id, orders.market_id, orders.outcome, orders.side,
		       orders.order_type, orders.time_in_force, orders.price, orders.quantity, orders.filled_quantity,
		       orders.status, orders.cancel_reason, orders.created_at, orders.updated_at
		FROM orders
		LEFT JOIN markets ON markets.market_id = orders.market_id
		WHERE orders.status IN ('NEW', 'PARTIALLY_FILLED')
		  AND orders.remaining_quantity > 0
		  AND orders.order_type = 'LIMIT'
		  AND COALESCE(markets.status, 'OPEN') = 'OPEN'
		ORDER BY orders.created_at, orders.order_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]*model.Order, 0)
	for rows.Next() {
		var (
			order        model.Order
			outcome      string
			side         string
			orderType    string
			timeInForce  string
			status       string
			cancelReason string
			createdAt    int64
			updatedAt    int64
		)
		if err := rows.Scan(
			&order.OrderID,
			&order.ClientOrderID,
			&order.UserID,
			&order.MarketID,
			&outcome,
			&side,
			&orderType,
			&timeInForce,
			&order.Price,
			&order.Quantity,
			&order.FilledQuantity,
			&status,
			&cancelReason,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, err
		}

		order.Outcome = strings.ToUpper(strings.TrimSpace(outcome))
		order.Side = model.OrderSide(strings.ToUpper(strings.TrimSpace(side)))
		order.Type = model.OrderType(strings.ToUpper(strings.TrimSpace(orderType)))
		order.TimeInForce = model.TimeInForce(strings.ToUpper(strings.TrimSpace(timeInForce)))
		order.Status = model.OrderStatus(strings.ToUpper(strings.TrimSpace(status)))
		order.CancelReason = model.CancelReason(strings.ToUpper(strings.TrimSpace(cancelReason)))
		order.CreatedAtMillis = createdAt * 1000
		order.UpdatedAtMillis = updatedAt * 1000
		orders = append(orders, &order)
	}
	return orders, rows.Err()
}

func (s *SQLStore) MarketIsTradable(ctx context.Context, marketID int64) (bool, error) {
	var status string
	if err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE((
			SELECT status
			FROM markets
			WHERE market_id = $1
		), 'OPEN')
	`, marketID).Scan(&status); err != nil {
		return false, err
	}
	return strings.ToUpper(strings.TrimSpace(status)) == "OPEN", nil
}

func (s *SQLStore) PersistResult(ctx context.Context, command sharedkafka.OrderCommand, result engine.Result) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := s.ensureMarket(ctx, tx, command.MarketID, command.CollateralAsset); err != nil {
		return err
	}

	if result.Order != nil {
		if err := s.upsertOrder(ctx, tx, command, result.Order, true); err != nil {
			return err
		}
	}
	for _, affected := range result.Affected {
		if err := s.upsertOrder(ctx, tx, command, affected, false); err != nil {
			return err
		}
	}
	for _, trade := range result.Trades {
		if err := s.insertTrade(ctx, tx, command.CollateralAsset, trade); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLStore) ensureMarket(ctx context.Context, tx *sql.Tx, marketID int64, collateralAsset string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO markets (
			market_id, title, description, collateral_asset, status,
			open_at, close_at, resolve_at, resolved_outcome, created_by, metadata, created_at, updated_at
		)
		VALUES ($1, $2, '', $3, 'OPEN', 0, 0, 0, '', 0, '{}'::jsonb, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
		ON CONFLICT (market_id) DO UPDATE
		SET collateral_asset = CASE
				WHEN markets.collateral_asset = '' THEN EXCLUDED.collateral_asset
				ELSE markets.collateral_asset
			END,
			updated_at = EXCLUDED.updated_at
	`, marketID, fmt.Sprintf("Market %d", marketID), normalizeAsset(collateralAsset))
	return err
}

func (s *SQLStore) upsertOrder(ctx context.Context, tx *sql.Tx, command sharedkafka.OrderCommand, order *model.Order, isCommandOrder bool) error {
	if order == nil {
		return nil
	}

	freezeID := ""
	freezeAsset := ""
	var freezeAmount int64
	commandID := ""
	if isCommandOrder {
		freezeID = command.FreezeID
		freezeAsset = command.FreezeAsset
		freezeAmount = command.FreezeAmount
		commandID = command.CommandID
	}

	_, err := tx.ExecContext(ctx, `
		INSERT INTO orders (
			order_id, client_order_id, command_id, user_id, market_id, outcome, side,
			order_type, time_in_force, collateral_asset, freeze_id, freeze_asset, freeze_amount,
			price, quantity, filled_quantity, remaining_quantity, status, cancel_reason, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13,
			$14, $15, $16, $17, $18, $19, $20, EXTRACT(EPOCH FROM NOW())::BIGINT
		)
		ON CONFLICT (order_id) DO UPDATE
		SET client_order_id = CASE WHEN EXCLUDED.client_order_id <> '' THEN EXCLUDED.client_order_id ELSE orders.client_order_id END,
			command_id = CASE WHEN EXCLUDED.command_id <> '' THEN EXCLUDED.command_id ELSE orders.command_id END,
			user_id = EXCLUDED.user_id,
			market_id = EXCLUDED.market_id,
			outcome = EXCLUDED.outcome,
			side = EXCLUDED.side,
			order_type = EXCLUDED.order_type,
			time_in_force = EXCLUDED.time_in_force,
			collateral_asset = EXCLUDED.collateral_asset,
			freeze_id = CASE WHEN EXCLUDED.freeze_id <> '' THEN EXCLUDED.freeze_id ELSE orders.freeze_id END,
			freeze_asset = CASE WHEN EXCLUDED.freeze_asset <> '' THEN EXCLUDED.freeze_asset ELSE orders.freeze_asset END,
			freeze_amount = CASE WHEN EXCLUDED.freeze_amount <> 0 THEN EXCLUDED.freeze_amount ELSE orders.freeze_amount END,
			price = EXCLUDED.price,
			quantity = EXCLUDED.quantity,
			filled_quantity = EXCLUDED.filled_quantity,
			remaining_quantity = EXCLUDED.remaining_quantity,
			status = EXCLUDED.status,
			cancel_reason = CASE WHEN EXCLUDED.cancel_reason <> '' THEN EXCLUDED.cancel_reason ELSE orders.cancel_reason END,
			updated_at = EXCLUDED.updated_at
	`,
		order.OrderID,
		order.ClientOrderID,
		commandID,
		order.UserID,
		order.MarketID,
		strings.ToUpper(strings.TrimSpace(order.Outcome)),
		string(order.Side),
		string(order.Type),
		string(order.TimeInForce),
		normalizeAsset(command.CollateralAsset),
		freezeID,
		normalizeAsset(freezeAsset),
		freezeAmount,
		order.Price,
		order.Quantity,
		order.FilledQuantity,
		order.RemainingQuantity(),
		string(order.Status),
		string(order.CancelReason),
		order.CreatedAtMillis/1000,
	)
	return err
}

func (s *SQLStore) insertTrade(ctx context.Context, tx *sql.Tx, collateralAsset string, trade model.Trade) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO trades (
			trade_id, sequence_no, market_id, outcome, collateral_asset, price, quantity,
			taker_order_id, maker_order_id, taker_user_id, maker_user_id,
			taker_side, maker_side, occurred_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (trade_id) DO NOTHING
	`,
		fmt.Sprintf("trd_%d", trade.Sequence),
		trade.Sequence,
		trade.MarketID,
		strings.ToUpper(strings.TrimSpace(trade.Outcome)),
		normalizeAsset(collateralAsset),
		trade.Price,
		trade.Quantity,
		trade.TakerOrderID,
		trade.MakerOrderID,
		trade.TakerUserID,
		trade.MakerUserID,
		string(trade.TakerSide),
		string(trade.MakerSide),
		trade.MatchedAtMillis/1000,
	)
	return err
}

func normalizeAsset(asset string) string {
	if strings.TrimSpace(asset) == "" {
		return "USDT"
	}
	return strings.ToUpper(strings.TrimSpace(asset))
}
