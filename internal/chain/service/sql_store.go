package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"funnyoption/internal/api/dto"
	chainmodel "funnyoption/internal/chain/model"
	claimmodel "funnyoption/internal/chain/model"
)

var ErrWalletSessionNotFound = errors.New("wallet session not found")

type DepositStore interface {
	UpsertDeposit(ctx context.Context, deposit chainmodel.Deposit) (chainmodel.Deposit, error)
	MarkDepositCredited(ctx context.Context, depositID string, creditedAt int64) error
	UpsertWithdrawal(ctx context.Context, withdrawal chainmodel.Withdrawal) (chainmodel.Withdrawal, error)
	MarkWithdrawalDebited(ctx context.Context, withdrawalID string, debitedAt int64) error
	LookupActiveUserByWallet(ctx context.Context, walletAddress string) (int64, error)
	LoadVaultScanCursor(ctx context.Context, chainName string, networkName string, vaultAddress string) (uint64, bool, error)
	SaveVaultScanCursor(ctx context.Context, chainName string, networkName string, vaultAddress string, nextBlock uint64) error
	ListPendingClaims(ctx context.Context, limit int) ([]claimmodel.ClaimTask, error)
	MarkClaimSubmitted(ctx context.Context, id int64, txHash string) error
	MarkClaimFailed(ctx context.Context, id int64, errMsg string) error
	MarkClaimConfirmedByTxHash(ctx context.Context, txHash string) error
}

type SQLStore struct {
	db *sql.DB
}

func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) UpsertDeposit(ctx context.Context, deposit chainmodel.Deposit) (chainmodel.Deposit, error) {
	var item chainmodel.Deposit
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO chain_deposits (
			deposit_id, user_id, wallet_address, vault_address, asset, amount,
			chain_name, network_name, tx_hash, log_index, block_number,
			status, credited_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
		        EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
		ON CONFLICT (tx_hash, log_index) DO UPDATE
		SET user_id = EXCLUDED.user_id,
			wallet_address = EXCLUDED.wallet_address,
			vault_address = EXCLUDED.vault_address,
			asset = EXCLUDED.asset,
			amount = EXCLUDED.amount,
			chain_name = EXCLUDED.chain_name,
			network_name = EXCLUDED.network_name,
			block_number = EXCLUDED.block_number,
			updated_at = EXCLUDED.updated_at
		RETURNING deposit_id, user_id, wallet_address, vault_address, asset, amount,
		          chain_name, network_name, tx_hash, log_index, block_number,
		          status, credited_at, created_at, updated_at
	`,
		deposit.DepositID,
		deposit.UserID,
		deposit.WalletAddress,
		deposit.VaultAddress,
		deposit.Asset,
		deposit.Amount,
		deposit.ChainName,
		deposit.NetworkName,
		deposit.TxHash,
		deposit.LogIndex,
		deposit.BlockNumber,
		deposit.Status,
		deposit.CreditedAt,
	).Scan(
		&item.DepositID,
		&item.UserID,
		&item.WalletAddress,
		&item.VaultAddress,
		&item.Asset,
		&item.Amount,
		&item.ChainName,
		&item.NetworkName,
		&item.TxHash,
		&item.LogIndex,
		&item.BlockNumber,
		&item.Status,
		&item.CreditedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

func (s *SQLStore) MarkDepositCredited(ctx context.Context, depositID string, creditedAt int64) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE chain_deposits
		SET status = 'CREDITED',
			credited_at = $2,
			updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE deposit_id = $1
	`, depositID, creditedAt)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("deposit not found")
	}
	return nil
}

func (s *SQLStore) UpsertWithdrawal(ctx context.Context, withdrawal chainmodel.Withdrawal) (chainmodel.Withdrawal, error) {
	var item chainmodel.Withdrawal
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO chain_withdrawals (
			withdrawal_id, user_id, wallet_address, recipient_address, vault_address, asset, amount,
			chain_name, network_name, tx_hash, log_index, block_number,
			status, debited_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
		        EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
		ON CONFLICT (tx_hash, log_index) DO UPDATE
		SET withdrawal_id = EXCLUDED.withdrawal_id,
			user_id = EXCLUDED.user_id,
			wallet_address = EXCLUDED.wallet_address,
			recipient_address = EXCLUDED.recipient_address,
			vault_address = EXCLUDED.vault_address,
			asset = EXCLUDED.asset,
			amount = EXCLUDED.amount,
			chain_name = EXCLUDED.chain_name,
			network_name = EXCLUDED.network_name,
			block_number = EXCLUDED.block_number,
			updated_at = EXCLUDED.updated_at
		RETURNING withdrawal_id, user_id, wallet_address, recipient_address, vault_address, asset, amount,
		          chain_name, network_name, tx_hash, log_index, block_number,
		          status, debited_at, created_at, updated_at
	`,
		withdrawal.WithdrawalID,
		withdrawal.UserID,
		withdrawal.WalletAddress,
		withdrawal.RecipientAddress,
		withdrawal.VaultAddress,
		withdrawal.Asset,
		withdrawal.Amount,
		withdrawal.ChainName,
		withdrawal.NetworkName,
		withdrawal.TxHash,
		withdrawal.LogIndex,
		withdrawal.BlockNumber,
		withdrawal.Status,
		withdrawal.DebitedAt,
	).Scan(
		&item.WithdrawalID,
		&item.UserID,
		&item.WalletAddress,
		&item.RecipientAddress,
		&item.VaultAddress,
		&item.Asset,
		&item.Amount,
		&item.ChainName,
		&item.NetworkName,
		&item.TxHash,
		&item.LogIndex,
		&item.BlockNumber,
		&item.Status,
		&item.DebitedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

func (s *SQLStore) MarkWithdrawalDebited(ctx context.Context, withdrawalID string, debitedAt int64) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE chain_withdrawals
		SET status = 'DEBITED',
			debited_at = $2,
			updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE withdrawal_id = $1
	`, withdrawalID, debitedAt)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("withdrawal not found")
	}
	return nil
}

func (s *SQLStore) LookupActiveUserByWallet(ctx context.Context, walletAddress string) (int64, error) {
	var userID int64
	err := s.db.QueryRowContext(ctx, `
		SELECT user_id
		FROM user_profiles
		WHERE wallet_address = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`, strings.ToLower(strings.TrimSpace(walletAddress))).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrWalletSessionNotFound
		}
		return 0, err
	}
	return userID, nil
}

func (s *SQLStore) LoadVaultScanCursor(ctx context.Context, chainName string, networkName string, vaultAddress string) (uint64, bool, error) {
	var nextBlock int64
	err := s.db.QueryRowContext(ctx, `
		SELECT next_block
		FROM chain_listener_cursors
		WHERE chain_name = $1
		  AND network_name = $2
		  AND vault_address = $3
	`, normalizeChainName(chainName), normalizeNetworkName(networkName), normalizeVaultAddress(vaultAddress)).Scan(&nextBlock)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, err
	}
	if nextBlock <= 0 {
		return 0, false, nil
	}
	return uint64(nextBlock), true, nil
}

func (s *SQLStore) SaveVaultScanCursor(ctx context.Context, chainName string, networkName string, vaultAddress string, nextBlock uint64) error {
	if nextBlock > math.MaxInt64 {
		return fmt.Errorf("vault scan cursor exceeds int64: %d", nextBlock)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO chain_listener_cursors (
			chain_name, network_name, vault_address, next_block, updated_at
		)
		VALUES ($1, $2, $3, $4, EXTRACT(EPOCH FROM NOW())::BIGINT)
		ON CONFLICT (chain_name, network_name, vault_address) DO UPDATE
		SET next_block = GREATEST(chain_listener_cursors.next_block, EXCLUDED.next_block),
			updated_at = EXCLUDED.updated_at
	`, normalizeChainName(chainName), normalizeNetworkName(networkName), normalizeVaultAddress(vaultAddress), int64(nextBlock))
	return err
}

func (s *SQLStore) ListPendingClaims(ctx context.Context, limit int) ([]claimmodel.ClaimTask, error) {
	return s.listClaimsByStatus(ctx, limit, "PENDING")
}

func (s *SQLStore) ListSubmittedClaims(ctx context.Context, limit int) ([]claimmodel.ClaimTask, error) {
	return s.listClaimsByStatus(ctx, limit, "SUBMITTED")
}

func (s *SQLStore) listClaimsByStatus(ctx context.Context, limit int, status string) ([]claimmodel.ClaimTask, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, biz_type, ref_id, chain_name, network_name, wallet_address, tx_hash, status,
		       payload, attempt_count, error_message, created_at, updated_at
		FROM chain_transactions
		WHERE biz_type IN ('CLAIM', 'WITHDRAWAL_CLAIM')
		  AND status = $2
		ORDER BY created_at ASC, id ASC
		LIMIT $1
	`, limit, strings.ToUpper(strings.TrimSpace(status)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []claimmodel.ClaimTask
	for rows.Next() {
		var (
			item    claimmodel.ClaimTask
			payload []byte
		)
		if err := rows.Scan(
			&item.ID,
			&item.BizType,
			&item.RefID,
			&item.ChainName,
			&item.NetworkName,
			&item.WalletAddress,
			&item.TxHash,
			&item.Status,
			&payload,
			&item.AttemptCount,
			&item.ErrorMessage,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}

		var claimPayload dto.ClaimPayoutRequest
		if err := json.Unmarshal(payload, &claimPayload); err != nil {
			return nil, err
		}
		item.RecipientAddress = strings.ToLower(strings.TrimSpace(claimPayload.RecipientAddress))
		item.PayoutAsset = strings.ToUpper(strings.TrimSpace(claimPayload.PayoutAsset))
		item.PayoutAmount = claimPayload.PayoutAmount
		tasks = append(tasks, item)
	}
	return tasks, rows.Err()
}

func (s *SQLStore) MarkClaimSubmitted(ctx context.Context, id int64, txHash string) error {
	return withPostgresDeadlockRetry(ctx, 3, func() error {
		return s.markClaimSubmittedOnce(ctx, id, txHash)
	})
}

func (s *SQLStore) markClaimSubmittedOnce(ctx context.Context, id int64, txHash string) error {
	txHash = strings.ToLower(strings.TrimSpace(txHash))
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var (
		bizType string
		refID   string
	)
	err = tx.QueryRowContext(ctx, `
		UPDATE chain_transactions
		SET status = 'SUBMITTED',
			tx_hash = $2,
			error_message = '',
			attempt_count = attempt_count + 1,
			updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE id = $1
		RETURNING biz_type, ref_id
	`, id, txHash).Scan(&bizType, &refID)
	if err != nil {
		return err
	}
	if bizType == "WITHDRAWAL_CLAIM" {
		if _, err := tx.ExecContext(ctx, `
			UPDATE rollup_accepted_withdrawals
			SET claim_status = 'CLAIM_SUBMITTED',
			    claim_tx_hash = $2,
			    claim_submitted_at = EXTRACT(EPOCH FROM NOW())::BIGINT,
			    last_error = '',
			    last_error_at = 0,
			    updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
			WHERE withdrawal_id = $1
		`, refID, txHash); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLStore) MarkClaimFailed(ctx context.Context, id int64, errMsg string) error {
	return withPostgresDeadlockRetry(ctx, 3, func() error {
		return s.markClaimFailedOnce(ctx, id, errMsg)
	})
}

func (s *SQLStore) markClaimFailedOnce(ctx context.Context, id int64, errMsg string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var (
		bizType string
		refID   string
	)
	err = tx.QueryRowContext(ctx, `
		UPDATE chain_transactions
		SET status = 'FAILED',
			error_message = $2,
			attempt_count = attempt_count + 1,
			updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE id = $1
		RETURNING biz_type, ref_id
	`, id, truncateString(errMsg, 255)).Scan(&bizType, &refID)
	if err != nil {
		return err
	}
	if bizType == "WITHDRAWAL_CLAIM" {
		if _, err := tx.ExecContext(ctx, `
			UPDATE rollup_accepted_withdrawals
			SET claim_status = 'FAILED',
			    last_error = $2,
			    last_error_at = EXTRACT(EPOCH FROM NOW())::BIGINT,
			    updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
			WHERE withdrawal_id = $1
		`, refID, truncateString(errMsg, 255)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLStore) MarkClaimConfirmedByTxHash(ctx context.Context, txHash string) error {
	return withPostgresDeadlockRetry(ctx, 3, func() error {
		return s.markClaimConfirmedByTxHashOnce(ctx, txHash)
	})
}

func (s *SQLStore) markClaimConfirmedByTxHashOnce(ctx context.Context, txHash string) error {
	txHash = strings.ToLower(strings.TrimSpace(txHash))
	if txHash == "" {
		return fmt.Errorf("tx hash is required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var (
		bizType string
		refID   string
	)
	err = tx.QueryRowContext(ctx, `
		UPDATE chain_transactions
		SET status = 'CONFIRMED',
			error_message = '',
			updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE tx_hash = $1
		  AND status = 'SUBMITTED'
		RETURNING biz_type, ref_id
	`, txHash).Scan(&bizType, &refID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	if bizType == "WITHDRAWAL_CLAIM" {
		if _, err := tx.ExecContext(ctx, `
			UPDATE rollup_accepted_withdrawals
			SET claim_status = 'CLAIMED',
			    claimed_at = EXTRACT(EPOCH FROM NOW())::BIGINT,
			    last_error = '',
			    last_error_at = 0,
			    updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
			WHERE withdrawal_id = $1
		`, refID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLStore) UpsertRollupForcedWithdrawalRequest(
	ctx context.Context,
	request chainmodel.RollupForcedWithdrawalRequest,
) error {
	satisfactionStatus := strings.TrimSpace(request.SatisfactionStatus)
	if satisfactionStatus == "" {
		satisfactionStatus = chainmodel.ForcedWithdrawalSatisfactionStatusNone
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO rollup_forced_withdrawal_requests (
			request_id, wallet_address, recipient_address, amount,
			requested_at, deadline_at, satisfied_claim_id, satisfied_at,
			frozen_at, status, matched_withdrawal_id, matched_claim_id,
			satisfaction_status, satisfaction_tx_hash, satisfaction_submitted_at,
			satisfaction_last_error, satisfaction_last_error_at, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11, $12,
			$13, $14, $15,
			$16, $17, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
		)
		ON CONFLICT (request_id) DO UPDATE
		SET wallet_address = EXCLUDED.wallet_address,
			recipient_address = EXCLUDED.recipient_address,
			amount = EXCLUDED.amount,
			requested_at = EXCLUDED.requested_at,
			deadline_at = EXCLUDED.deadline_at,
			satisfied_claim_id = EXCLUDED.satisfied_claim_id,
			satisfied_at = EXCLUDED.satisfied_at,
			frozen_at = EXCLUDED.frozen_at,
			status = EXCLUDED.status,
			matched_claim_id = CASE
				WHEN EXCLUDED.satisfied_claim_id <> '' THEN EXCLUDED.satisfied_claim_id
				ELSE rollup_forced_withdrawal_requests.matched_claim_id
			END,
			satisfaction_status = CASE
				WHEN EXCLUDED.status = 'SATISFIED' THEN 'SATISFIED'
				ELSE rollup_forced_withdrawal_requests.satisfaction_status
			END,
			updated_at = EXCLUDED.updated_at
	`,
		request.RequestID,
		request.WalletAddress,
		request.RecipientAddress,
		request.Amount,
		request.RequestedAt,
		request.DeadlineAt,
		request.SatisfiedClaimID,
		request.SatisfiedAt,
		request.FrozenAt,
		request.Status,
		request.MatchedWithdrawalID,
		request.MatchedClaimID,
		satisfactionStatus,
		request.SatisfactionTxHash,
		request.SatisfactionSubmittedAt,
		request.SatisfactionLastError,
		request.SatisfactionLastErrorAt,
	)
	return err
}

func (s *SQLStore) UpsertRollupFreezeState(ctx context.Context, state chainmodel.RollupFreezeState) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO rollup_freeze_state (
			id, frozen, frozen_at, request_id, updated_at
		)
		VALUES (
			TRUE, $1, $2, $3, EXTRACT(EPOCH FROM NOW())::BIGINT
		)
		ON CONFLICT (id) DO UPDATE
		SET frozen = EXCLUDED.frozen,
			frozen_at = EXCLUDED.frozen_at,
			request_id = EXCLUDED.request_id,
			updated_at = EXCLUDED.updated_at
	`, state.Frozen, state.FrozenAt, state.RequestID)
	return err
}

func (s *SQLStore) ListPendingRollupForcedWithdrawalRequests(
	ctx context.Context,
	limit int,
) ([]chainmodel.RollupForcedWithdrawalRequest, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT request_id, wallet_address, recipient_address, amount,
		       requested_at, deadline_at, satisfied_claim_id, satisfied_at,
		       frozen_at, status, matched_withdrawal_id, matched_claim_id,
		       satisfaction_status, satisfaction_tx_hash, satisfaction_submitted_at,
		       satisfaction_last_error, satisfaction_last_error_at, created_at, updated_at
		FROM rollup_forced_withdrawal_requests
		WHERE status = 'REQUESTED'
		ORDER BY request_id ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]chainmodel.RollupForcedWithdrawalRequest, 0, limit)
	for rows.Next() {
		var item chainmodel.RollupForcedWithdrawalRequest
		if err := rows.Scan(
			&item.RequestID,
			&item.WalletAddress,
			&item.RecipientAddress,
			&item.Amount,
			&item.RequestedAt,
			&item.DeadlineAt,
			&item.SatisfiedClaimID,
			&item.SatisfiedAt,
			&item.FrozenAt,
			&item.Status,
			&item.MatchedWithdrawalID,
			&item.MatchedClaimID,
			&item.SatisfactionStatus,
			&item.SatisfactionTxHash,
			&item.SatisfactionSubmittedAt,
			&item.SatisfactionLastError,
			&item.SatisfactionLastErrorAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *SQLStore) ListForcedWithdrawalClaimMatches(
	ctx context.Context,
	requestID int64,
	limit int,
) ([]chainmodel.ForcedWithdrawalClaimMatch, error) {
	if requestID <= 0 {
		return nil, fmt.Errorf("request_id must be positive")
	}
	if limit <= 0 {
		limit = 2
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT aw.withdrawal_id, aw.claim_id, aw.amount, aw.claimed_at
		FROM rollup_forced_withdrawal_requests r
		JOIN rollup_accepted_withdrawals aw
		  ON aw.wallet_address = r.wallet_address
		 AND aw.recipient_address = r.recipient_address
		 AND aw.claim_status = 'CLAIMED'
		 AND aw.claim_id <> ''
		WHERE r.request_id = $1
		  AND r.status = 'REQUESTED'
		  AND NOT EXISTS (
		    SELECT 1
		    FROM rollup_forced_withdrawal_requests other
		    WHERE other.request_id <> r.request_id
		      AND (
		        other.satisfied_claim_id = aw.claim_id
		        OR other.matched_claim_id = aw.claim_id
		      )
		  )
		ORDER BY aw.claimed_at ASC, aw.withdrawal_id ASC
		LIMIT $2
	`, requestID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]chainmodel.ForcedWithdrawalClaimMatch, 0, limit)
	for rows.Next() {
		var item chainmodel.ForcedWithdrawalClaimMatch
		if err := rows.Scan(&item.WithdrawalID, &item.ClaimID, &item.Amount, &item.ClaimedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *SQLStore) UpdateRollupForcedWithdrawalMatch(
	ctx context.Context,
	requestID int64,
	withdrawalID string,
	claimID string,
	status string,
	errMsg string,
) error {
	if requestID <= 0 {
		return fmt.Errorf("request_id must be positive")
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE rollup_forced_withdrawal_requests
		SET matched_withdrawal_id = $2,
		    matched_claim_id = $3,
		    satisfaction_status = $4,
		    satisfaction_tx_hash = CASE
		        WHEN $4 IN ('NONE', 'READY', 'AMBIGUOUS') THEN ''
		        ELSE satisfaction_tx_hash
		    END,
		    satisfaction_submitted_at = CASE
		        WHEN $4 IN ('NONE', 'READY', 'AMBIGUOUS') THEN 0
		        ELSE satisfaction_submitted_at
		    END,
		    satisfaction_last_error = $5,
		    satisfaction_last_error_at = CASE
		        WHEN $5 <> '' THEN EXTRACT(EPOCH FROM NOW())::BIGINT
		        ELSE 0
		    END,
		    updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE request_id = $1
	`, requestID, strings.TrimSpace(withdrawalID), strings.TrimSpace(claimID), strings.TrimSpace(status), truncateString(errMsg, 255))
	return err
}

func (s *SQLStore) MarkRollupForcedWithdrawalSatisfactionSubmitted(
	ctx context.Context,
	requestID int64,
	txHash string,
) error {
	if requestID <= 0 {
		return fmt.Errorf("request_id must be positive")
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE rollup_forced_withdrawal_requests
		SET satisfaction_status = 'SUBMITTED',
		    satisfaction_tx_hash = $2,
		    satisfaction_submitted_at = EXTRACT(EPOCH FROM NOW())::BIGINT,
		    satisfaction_last_error = '',
		    satisfaction_last_error_at = 0,
		    updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE request_id = $1
	`, requestID, normalizeChainTxHash(txHash))
	return err
}

func (s *SQLStore) MarkRollupForcedWithdrawalSatisfactionFailed(
	ctx context.Context,
	requestID int64,
	errMsg string,
) error {
	if requestID <= 0 {
		return fmt.Errorf("request_id must be positive")
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE rollup_forced_withdrawal_requests
		SET satisfaction_status = 'FAILED',
		    satisfaction_last_error = $2,
		    satisfaction_last_error_at = EXTRACT(EPOCH FROM NOW())::BIGINT,
		    updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE request_id = $1
	`, requestID, truncateString(errMsg, 255))
	return err
}

func truncateString(value string, size int) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= size {
		return trimmed
	}
	return trimmed[:size]
}

func withPostgresDeadlockRetry(ctx context.Context, attempts int, fn func() error) error {
	if attempts <= 0 {
		attempts = 1
	}
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := fn(); err != nil {
			lastErr = err
			if !isPostgresDeadlockError(err) || attempt == attempts {
				return err
			}
			if err := sleepWithContext(ctx, 50*time.Millisecond); err != nil {
				return err
			}
			continue
		}
		return nil
	}
	return lastErr
}

func isPostgresDeadlockError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(strings.TrimSpace(err.Error())), "deadlock detected")
}

func normalizeVaultAddress(vaultAddress string) string {
	return strings.ToLower(strings.TrimSpace(vaultAddress))
}
