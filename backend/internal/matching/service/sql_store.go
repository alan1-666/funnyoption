package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"funnyoption/internal/matching/engine"
	"funnyoption/internal/matching/model"
	"funnyoption/internal/posttrade"
	"funnyoption/internal/rollup"
	sharedkafka "funnyoption/internal/shared/kafka"
)

type SQLStore struct {
	db     *sql.DB
	rollup *rollup.Store
}

type ExpiredRestingOrder struct {
	Command sharedkafka.OrderCommand
	Order   *model.Order
}

func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) WithRollup(store *rollup.Store) *SQLStore {
	s.rollup = store
	return s
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
	frozen, err := s.rollupFrozen(ctx)
	if err != nil {
		return nil, err
	}
	if frozen {
		return []*model.Order{}, nil
	}

	nowUnix := time.Now().Unix()
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
		  AND (COALESCE(markets.close_at, 0) <= 0 OR markets.close_at > $1)
		ORDER BY orders.created_at, orders.order_id
	`, nowUnix)
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

func (s *SQLStore) LoadExpiredRestingOrders(ctx context.Context, nowUnix int64) ([]ExpiredRestingOrder, error) {
	frozen, err := s.rollupFrozen(ctx)
	if err != nil {
		return nil, err
	}
	if frozen {
		return []ExpiredRestingOrder{}, nil
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT orders.order_id, orders.client_order_id, orders.command_id, orders.user_id, orders.market_id, orders.outcome, orders.side,
		       orders.order_type, orders.time_in_force, orders.collateral_asset, orders.freeze_id, orders.freeze_asset, orders.freeze_amount,
		       orders.price, orders.quantity, orders.filled_quantity, orders.status, orders.cancel_reason, orders.created_at, orders.updated_at
		FROM orders
		INNER JOIN markets ON markets.market_id = orders.market_id
		WHERE orders.status IN ('NEW', 'PARTIALLY_FILLED')
		  AND orders.remaining_quantity > 0
		  AND orders.order_type = 'LIMIT'
		  AND COALESCE(markets.status, 'OPEN') = 'OPEN'
		  AND COALESCE(markets.close_at, 0) > 0
		  AND markets.close_at <= $1
		ORDER BY markets.close_at, orders.created_at, orders.order_id
	`, nowUnix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ExpiredRestingOrder, 0)
	for rows.Next() {
		var (
			order        model.Order
			command      sharedkafka.OrderCommand
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
			&command.CommandID,
			&order.UserID,
			&order.MarketID,
			&outcome,
			&side,
			&orderType,
			&timeInForce,
			&command.CollateralAsset,
			&command.FreezeID,
			&command.FreezeAsset,
			&command.FreezeAmount,
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

		command.OrderID = order.OrderID
		command.ClientOrderID = order.ClientOrderID
		command.UserID = order.UserID
		command.MarketID = order.MarketID
		command.Outcome = order.Outcome
		command.Side = string(order.Side)
		command.Type = string(order.Type)
		command.TimeInForce = string(order.TimeInForce)
		command.Price = order.Price
		command.Quantity = order.Quantity
		command.RequestedAtMillis = order.UpdatedAtMillis

		items = append(items, ExpiredRestingOrder{
			Command: command,
			Order:   &order,
		})
	}
	return items, rows.Err()
}

func (s *SQLStore) MarketIsTradable(ctx context.Context, marketID int64) (bool, error) {
	frozen, err := s.rollupFrozen(ctx)
	if err != nil {
		return false, err
	}
	return s.marketTradableWithFrozen(ctx, frozen, marketID)
}

func (s *SQLStore) marketTradableWithFrozen(ctx context.Context, frozen bool, marketID int64) (bool, error) {
	var (
		status  string
		closeAt int64
	)
	if err := s.db.QueryRowContext(ctx, `
		SELECT status, close_at
		FROM markets
		WHERE market_id = $1
	`, marketID).Scan(&status, &closeAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return !frozen, nil
		}
		return false, err
	}
	return marketTradingEnabled(frozen, status, closeAt, time.Now().Unix()), nil
}

func (s *SQLStore) MarketTradableNoFreeze(ctx context.Context, marketID int64) (bool, error) {
	return s.marketTradableWithFrozen(ctx, false, marketID)
}

func (s *SQLStore) RollupFrozen(ctx context.Context) (bool, error) {
	return s.rollupFrozen(ctx)
}

func (s *SQLStore) rollupFrozen(ctx context.Context) (bool, error) {
	var frozen bool
	err := s.db.QueryRowContext(ctx, `
		SELECT frozen
		FROM rollup_freeze_state
		WHERE id = TRUE
	`).Scan(&frozen)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return frozen, nil
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
	if s.rollup != nil {
		entries := buildRollupEntries(command, result)
		if err := s.rollup.AppendEntriesTx(ctx, tx, entries); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// PersistBatch persists multiple results in a single transaction using
// multi-row INSERT statements, amortising both the fsync cost and the
// per-query round-trip overhead.
func (s *SQLStore) PersistBatch(ctx context.Context, items []posttrade.PersistItem) error {
	if len(items) == 0 {
		return nil
	}
	if len(items) == 1 {
		return s.PersistResult(ctx, items[0].Command, items[0].Result)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := s.bulkEnsureMarkets(ctx, tx, items); err != nil {
		return err
	}
	if err := s.bulkUpsertOrders(ctx, tx, items); err != nil {
		return err
	}
	if err := s.bulkInsertTrades(ctx, tx, items); err != nil {
		return err
	}
	if s.rollup != nil {
		var entries []rollup.JournalAppend
		for _, item := range items {
			entries = append(entries, buildRollupEntries(item.Command, item.Result)...)
		}
		if err := s.rollup.AppendBatchTx(ctx, tx, entries); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// bulkEnsureMarkets inserts all unique markets in one multi-row upsert.
func (s *SQLStore) bulkEnsureMarkets(ctx context.Context, tx *sql.Tx, items []posttrade.PersistItem) error {
	seen := make(map[int64]bool, len(items))
	const pp = 3 // params per row
	args := make([]any, 0, len(items)*pp)
	n := 0
	for _, item := range items {
		if seen[item.Command.MarketID] {
			continue
		}
		seen[item.Command.MarketID] = true
		args = append(args, item.Command.MarketID, fmt.Sprintf("Market %d", item.Command.MarketID), normalizeAsset(item.Command.CollateralAsset))
		n++
	}
	if n == 0 {
		return nil
	}

	var b strings.Builder
	b.WriteString(`INSERT INTO markets (market_id,title,description,collateral_asset,status,open_at,close_at,resolve_at,resolved_outcome,created_by,metadata,created_at,updated_at) VALUES `)
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		p := i * pp
		fmt.Fprintf(&b, "($%d,$%d,'',$%d,'OPEN',0,0,0,'',0,'{}'::jsonb,EXTRACT(EPOCH FROM NOW())::BIGINT,EXTRACT(EPOCH FROM NOW())::BIGINT)", p+1, p+2, p+3)
	}
	b.WriteString(` ON CONFLICT (market_id) DO UPDATE SET collateral_asset=CASE WHEN markets.collateral_asset='' THEN EXCLUDED.collateral_asset ELSE markets.collateral_asset END,updated_at=EXCLUDED.updated_at`)
	_, err := tx.ExecContext(ctx, b.String(), args...)
	return err
}

// bulkUpsertOrders inserts/updates all orders (taker + affected) in one multi-row upsert.
// Duplicate order_ids are deduplicated — the last occurrence wins (most recent state).
func (s *SQLStore) bulkUpsertOrders(ctx context.Context, tx *sql.Tx, items []posttrade.PersistItem) error {
	const pp = 20 // params per row

	// Collect rows, deduplicating by order_id (last write wins).
	type orderArgs [pp]any
	dedup := make(map[string]int)     // order_id → index in rows
	rows := make([]orderArgs, 0, len(items)*2)

	addRow := func(cmd sharedkafka.OrderCommand, order *model.Order, isTaker bool) {
		if order == nil {
			return
		}
		freezeID, freezeAsset, commandID := "", "", ""
		var freezeAmount int64
		if isTaker {
			freezeID = cmd.FreezeID
			freezeAsset = cmd.FreezeAsset
			freezeAmount = cmd.FreezeAmount
			commandID = cmd.CommandID
		}
		row := orderArgs{
			order.OrderID, order.ClientOrderID, commandID,
			order.UserID, order.MarketID,
			strings.ToUpper(strings.TrimSpace(order.Outcome)),
			string(order.Side), string(order.Type), string(order.TimeInForce),
			normalizeAsset(cmd.CollateralAsset),
			freezeID, normalizeAsset(freezeAsset), freezeAmount,
			order.Price, order.Quantity, order.FilledQuantity, order.RemainingQuantity(),
			string(order.Status), string(order.CancelReason),
			order.CreatedAtMillis / 1000,
		}
		if idx, dup := dedup[order.OrderID]; dup {
			rows[idx] = row // overwrite with latest state
		} else {
			dedup[order.OrderID] = len(rows)
			rows = append(rows, row)
		}
	}

	for _, item := range items {
		addRow(item.Command, item.Result.Order, true)
		for _, affected := range item.Result.Affected {
			addRow(item.Command, affected, false)
		}
	}
	n := len(rows)
	if n == 0 {
		return nil
	}

	args := make([]any, 0, n*pp)
	for _, row := range rows {
		args = append(args, row[:]...)
	}

	var b strings.Builder
	b.WriteString(`INSERT INTO orders (order_id,client_order_id,command_id,user_id,market_id,outcome,side,order_type,time_in_force,collateral_asset,freeze_id,freeze_asset,freeze_amount,price,quantity,filled_quantity,remaining_quantity,status,cancel_reason,created_at,updated_at) VALUES `)
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		p := i * pp
		fmt.Fprintf(&b, "($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,EXTRACT(EPOCH FROM NOW())::BIGINT)",
			p+1, p+2, p+3, p+4, p+5, p+6, p+7, p+8, p+9, p+10,
			p+11, p+12, p+13, p+14, p+15, p+16, p+17, p+18, p+19, p+20)
	}
	b.WriteString(` ON CONFLICT (order_id) DO UPDATE SET client_order_id=CASE WHEN EXCLUDED.client_order_id<>'' THEN EXCLUDED.client_order_id ELSE orders.client_order_id END,command_id=CASE WHEN EXCLUDED.command_id<>'' THEN EXCLUDED.command_id ELSE orders.command_id END,user_id=EXCLUDED.user_id,market_id=EXCLUDED.market_id,outcome=EXCLUDED.outcome,side=EXCLUDED.side,order_type=EXCLUDED.order_type,time_in_force=EXCLUDED.time_in_force,collateral_asset=EXCLUDED.collateral_asset,freeze_id=CASE WHEN EXCLUDED.freeze_id<>'' THEN EXCLUDED.freeze_id ELSE orders.freeze_id END,freeze_asset=CASE WHEN EXCLUDED.freeze_asset<>'' THEN EXCLUDED.freeze_asset ELSE orders.freeze_asset END,freeze_amount=CASE WHEN EXCLUDED.freeze_amount<>0 THEN EXCLUDED.freeze_amount ELSE orders.freeze_amount END,price=EXCLUDED.price,quantity=EXCLUDED.quantity,filled_quantity=EXCLUDED.filled_quantity,remaining_quantity=EXCLUDED.remaining_quantity,status=EXCLUDED.status,cancel_reason=CASE WHEN EXCLUDED.cancel_reason<>'' THEN EXCLUDED.cancel_reason ELSE orders.cancel_reason END,updated_at=EXCLUDED.updated_at`)
	_, err := tx.ExecContext(ctx, b.String(), args...)
	return err
}

// bulkInsertTrades inserts all trades in one multi-row INSERT.
func (s *SQLStore) bulkInsertTrades(ctx context.Context, tx *sql.Tx, items []posttrade.PersistItem) error {
	const pp = 14 // params per row
	args := make([]any, 0, len(items)*pp)
	n := 0
	for _, item := range items {
		for _, trade := range item.Result.Trades {
			args = append(args,
				fmt.Sprintf("trd_%d", trade.Sequence), trade.Sequence,
				trade.MarketID, strings.ToUpper(strings.TrimSpace(trade.Outcome)),
				normalizeAsset(item.Command.CollateralAsset),
				trade.Price, trade.Quantity,
				trade.TakerOrderID, trade.MakerOrderID,
				trade.TakerUserID, trade.MakerUserID,
				string(trade.TakerSide), string(trade.MakerSide),
				trade.MatchedAtMillis/1000,
			)
			n++
		}
	}
	if n == 0 {
		return nil
	}

	var b strings.Builder
	b.WriteString(`INSERT INTO trades (trade_id,sequence_no,market_id,outcome,collateral_asset,price,quantity,taker_order_id,maker_order_id,taker_user_id,maker_user_id,taker_side,maker_side,occurred_at) VALUES `)
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		p := i * pp
		fmt.Fprintf(&b, "($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
			p+1, p+2, p+3, p+4, p+5, p+6, p+7, p+8, p+9, p+10, p+11, p+12, p+13, p+14)
	}
	b.WriteString(` ON CONFLICT (trade_id) DO NOTHING`)
	_, err := tx.ExecContext(ctx, b.String(), args...)
	return err
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
