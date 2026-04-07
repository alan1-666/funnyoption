package service

import (
	"context"
	"database/sql"
	"errors"

	"funnyoption/internal/account/model"
)

type SQLStore struct {
	db *sql.DB
}

func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) LoadBalances(ctx context.Context) ([]model.Balance, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id, asset, available, frozen
		FROM account_balances
		ORDER BY user_id, asset
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []model.Balance
	for rows.Next() {
		var item model.Balance
		if err := rows.Scan(&item.UserID, &item.Asset, &item.Available, &item.Frozen); err != nil {
			return nil, err
		}
		balances = append(balances, item)
	}
	return balances, rows.Err()
}

func (s *SQLStore) RollupFrozen(ctx context.Context) (bool, error) {
	var frozen bool
	err := s.db.QueryRowContext(ctx, `
		SELECT frozen
		FROM rollup_freeze_state
		ORDER BY id DESC
		LIMIT 1
	`).Scan(&frozen)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return frozen, nil
}

func (s *SQLStore) LoadFreezes(ctx context.Context) ([]model.FreezeRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT freeze_id, user_id, asset, ref_type, ref_id, original_amount, remaining_amount, status
		FROM freeze_records
		WHERE status = 'ACTIVE'
		ORDER BY created_at, freeze_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var freezes []model.FreezeRecord
	for rows.Next() {
		var (
			item   model.FreezeRecord
			status string
		)
		if err := rows.Scan(&item.FreezeID, &item.UserID, &item.Asset, &item.RefType, &item.RefID, &item.OriginalAmount, &item.Amount, &status); err != nil {
			return nil, err
		}
		item.Released = status == "RELEASED"
		item.Consumed = status == "CONSUMED"
		freezes = append(freezes, item)
	}
	return freezes, rows.Err()
}

func (s *SQLStore) UpsertBalance(ctx context.Context, balance model.Balance) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO account_balances (user_id, asset, available, frozen, created_at, updated_at)
		VALUES ($1, $2, $3, $4, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
		ON CONFLICT (user_id, asset) DO UPDATE
		SET available = EXCLUDED.available,
			frozen = EXCLUDED.frozen,
			updated_at = EXCLUDED.updated_at
	`, balance.UserID, balance.Asset, balance.Available, balance.Frozen)
	return err
}

func (s *SQLStore) UpsertFreeze(ctx context.Context, record model.FreezeRecord) error {
	originalAmount := record.OriginalAmount
	if originalAmount <= 0 {
		originalAmount = record.Amount
	}
	status := "ACTIVE"
	if record.Released {
		status = "RELEASED"
	}
	if record.Consumed {
		status = "CONSUMED"
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO freeze_records (
			freeze_id, user_id, asset, ref_type, ref_id,
			original_amount, remaining_amount, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
		ON CONFLICT (freeze_id) DO UPDATE
		SET user_id = EXCLUDED.user_id,
			asset = EXCLUDED.asset,
			ref_type = EXCLUDED.ref_type,
			ref_id = EXCLUDED.ref_id,
			original_amount = CASE
				WHEN freeze_records.original_amount > 0 THEN freeze_records.original_amount
				ELSE EXCLUDED.original_amount
			END,
			remaining_amount = EXCLUDED.remaining_amount,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`, record.FreezeID, record.UserID, record.Asset, record.RefType, record.RefID, originalAmount, record.Amount, status)
	return err
}

func (s *SQLStore) UpsertBalanceAndFreeze(ctx context.Context, balance model.Balance, freeze model.FreezeRecord) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO account_balances (user_id, asset, available, frozen, created_at, updated_at)
		VALUES ($1, $2, $3, $4, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
		ON CONFLICT (user_id, asset) DO UPDATE
		SET available = EXCLUDED.available,
			frozen = EXCLUDED.frozen,
			updated_at = EXCLUDED.updated_at
	`, balance.UserID, balance.Asset, balance.Available, balance.Frozen); err != nil {
		return err
	}

	originalAmount := freeze.OriginalAmount
	if originalAmount <= 0 {
		originalAmount = freeze.Amount
	}
	status := "ACTIVE"
	if freeze.Released {
		status = "RELEASED"
	}
	if freeze.Consumed {
		status = "CONSUMED"
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO freeze_records (
			freeze_id, user_id, asset, ref_type, ref_id,
			original_amount, remaining_amount, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
		ON CONFLICT (freeze_id) DO UPDATE
		SET user_id = EXCLUDED.user_id,
			asset = EXCLUDED.asset,
			ref_type = EXCLUDED.ref_type,
			ref_id = EXCLUDED.ref_id,
			original_amount = CASE
				WHEN freeze_records.original_amount > 0 THEN freeze_records.original_amount
				ELSE EXCLUDED.original_amount
			END,
			remaining_amount = EXCLUDED.remaining_amount,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`, freeze.FreezeID, freeze.UserID, freeze.Asset, freeze.RefType, freeze.RefID, originalAmount, freeze.Amount, status); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SQLStore) LoadOrderState(ctx context.Context, orderID string) (*OrderState, error) {
	var state OrderState
	err := s.db.QueryRowContext(ctx, `
		SELECT order_id, user_id, side, price, freeze_id, freeze_asset, status, remaining_quantity
		FROM orders
		WHERE order_id = $1
	`, orderID).Scan(
		&state.OrderID,
		&state.UserID,
		&state.Side,
		&state.Price,
		&state.FreezeID,
		&state.FreezeAsset,
		&state.Status,
		&state.RemainingQuantity,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	state.FreezeApplied = state.FreezeID != ""
	return &state, nil
}

func (s *SQLStore) MirrorOrderState(ctx context.Context, state OrderState) error {
	if state.OrderID == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE orders
		SET user_id = CASE WHEN $2 <> 0 THEN $2 ELSE user_id END,
			side = CASE WHEN $3 <> '' THEN $3 ELSE side END,
			price = CASE WHEN $4 <> 0 THEN $4 ELSE price END,
			freeze_id = CASE WHEN $5 <> '' THEN $5 ELSE freeze_id END,
			freeze_asset = CASE WHEN $6 <> '' THEN $6 ELSE freeze_asset END,
			status = CASE WHEN $7 <> '' THEN $7 ELSE status END,
			remaining_quantity = CASE WHEN $8 <> 0 THEN $8 ELSE remaining_quantity END,
			updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE order_id = $1
	`, state.OrderID, state.UserID, state.Side, state.Price, state.FreezeID, state.FreezeAsset, state.Status, state.RemainingQuantity)
	return err
}

func (s *SQLStore) ApplyCreditEvent(ctx context.Context, req CreditRequest) (model.Balance, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.Balance{}, false, err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		INSERT INTO account_balance_events (event_type, ref_id, user_id, asset, direction, amount, created_at)
		VALUES ($1, $2, $3, $4, 'CREDIT', $5, EXTRACT(EPOCH FROM NOW())::BIGINT)
		ON CONFLICT (event_type, ref_id) DO NOTHING
	`, req.RefType, req.RefID, req.UserID, req.Asset, req.Amount)
	if err != nil {
		return model.Balance{}, false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return model.Balance{}, false, err
	}
	if affected == 0 {
		balance, err := s.loadBalanceTx(ctx, tx, req.UserID, req.Asset)
		if err != nil {
			return model.Balance{}, false, err
		}
		if err := tx.Commit(); err != nil {
			return model.Balance{}, false, err
		}
		return balance, false, nil
	}

	var balance model.Balance
	err = tx.QueryRowContext(ctx, `
		INSERT INTO account_balances (user_id, asset, available, frozen, created_at, updated_at)
		VALUES ($1, $2, $3, 0, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
		ON CONFLICT (user_id, asset) DO UPDATE
		SET available = account_balances.available + EXCLUDED.available,
			updated_at = EXCLUDED.updated_at
		RETURNING user_id, asset, available, frozen
	`, req.UserID, req.Asset, req.Amount).Scan(&balance.UserID, &balance.Asset, &balance.Available, &balance.Frozen)
	if err != nil {
		return model.Balance{}, false, err
	}

	if err := tx.Commit(); err != nil {
		return model.Balance{}, false, err
	}
	return balance, true, nil
}

func (s *SQLStore) ApplyDebitEvent(ctx context.Context, req DebitRequest) (model.Balance, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.Balance{}, false, err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		INSERT INTO account_balance_events (event_type, ref_id, user_id, asset, direction, amount, created_at)
		VALUES ($1, $2, $3, $4, 'DEBIT', $5, EXTRACT(EPOCH FROM NOW())::BIGINT)
		ON CONFLICT (event_type, ref_id) DO NOTHING
	`, req.RefType, req.RefID, req.UserID, req.Asset, req.Amount)
	if err != nil {
		return model.Balance{}, false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return model.Balance{}, false, err
	}
	if affected == 0 {
		balance, err := s.loadBalanceTx(ctx, tx, req.UserID, req.Asset)
		if err != nil {
			return model.Balance{}, false, err
		}
		if err := tx.Commit(); err != nil {
			return model.Balance{}, false, err
		}
		return balance, false, nil
	}

	current, err := s.loadBalanceTx(ctx, tx, req.UserID, req.Asset)
	if err != nil {
		return model.Balance{}, false, err
	}
	if current.Available < req.Amount {
		return model.Balance{}, false, ErrInsufficientBalance
	}

	var balance model.Balance
	err = tx.QueryRowContext(ctx, `
		UPDATE account_balances
		SET available = available - $3,
			updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE user_id = $1
		  AND asset = $2
		RETURNING user_id, asset, available, frozen
	`, req.UserID, req.Asset, req.Amount).Scan(&balance.UserID, &balance.Asset, &balance.Available, &balance.Frozen)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Balance{}, false, ErrInsufficientBalance
		}
		return model.Balance{}, false, err
	}

	if err := tx.Commit(); err != nil {
		return model.Balance{}, false, err
	}
	return balance, true, nil
}

func (s *SQLStore) loadBalanceTx(ctx context.Context, tx *sql.Tx, userID int64, asset string) (model.Balance, error) {
	var balance model.Balance
	err := tx.QueryRowContext(ctx, `
		SELECT user_id, asset, available, frozen
		FROM account_balances
		WHERE user_id = $1 AND asset = $2
	`, userID, asset).Scan(&balance.UserID, &balance.Asset, &balance.Available, &balance.Frozen)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Balance{UserID: userID, Asset: asset}, nil
		}
		return model.Balance{}, err
	}
	return balance, nil
}
