package rollup

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	sharedkafka "funnyoption/internal/shared/kafka"
)

var ErrNoPendingBatch = errors.New("no pending rollup batch")
var ErrNoPendingSubmission = errors.New("no pending rollup submission")

const submissionSelectColumns = `
		submission_id, batch_id, encoding_version, status,
		batch_data_hash, next_state_root, auth_proof_hash,
		verifier_gate_hash, record_calldata, accept_calldata,
		submission_data, submission_hash, record_tx_hash, accept_tx_hash,
		record_submitted_at, accept_submitted_at, accepted_at,
		last_error, last_error_at, created_at, updated_at
`

type sqlQueryer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type Store struct {
	db *sql.DB
}

type acceptedWithdrawalClaimPayload struct {
	EventID          string `json:"event_id"`
	UserID           int64  `json:"user_id"`
	WalletAddress    string `json:"wallet_address"`
	RecipientAddress string `json:"recipient_address"`
	PayoutAsset      string `json:"payout_asset"`
	PayoutAmount     int64  `json:"payout_amount"`
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) AppendEntries(ctx context.Context, entries []JournalAppend) error {
	if s == nil || len(entries) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := s.AppendEntriesTx(ctx, tx, entries); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) AppendEntriesTx(ctx context.Context, tx *sql.Tx, entries []JournalAppend) error {
	if s == nil || len(entries) == 0 {
		return nil
	}
	for _, entry := range entries {
		if err := appendEntry(ctx, tx, entry); err != nil {
			return err
		}
	}
	return nil
}

func appendEntry(ctx context.Context, q sqlQueryer, entry JournalAppend) error {
	if strings.TrimSpace(entry.EntryType) == "" {
		return fmt.Errorf("rollup entry type is required")
	}
	if strings.TrimSpace(entry.SourceType) == "" {
		return fmt.Errorf("rollup source type is required")
	}
	if strings.TrimSpace(entry.SourceRef) == "" {
		return fmt.Errorf("rollup source ref is required")
	}
	payload, err := json.Marshal(entry.Payload)
	if err != nil {
		return err
	}
	entryID := strings.TrimSpace(entry.EntryID)
	if entryID == "" {
		entryID = sharedkafka.NewID("rj")
	}

	row := q.QueryRowContext(ctx, `
		INSERT INTO rollup_shadow_journal_entries (
			entry_id, entry_type, source_type, source_ref,
			occurred_at_millis, payload, created_at
		)
		VALUES (
			$1, $2, $3, $4,
			$5, $6, EXTRACT(EPOCH FROM NOW())::BIGINT
		)
		ON CONFLICT (entry_type, source_type, source_ref) DO NOTHING
		RETURNING sequence_no
	`, entryID, strings.TrimSpace(entry.EntryType), strings.TrimSpace(entry.SourceType), strings.TrimSpace(entry.SourceRef), entry.OccurredAtMillis, payload)

	var sequence int64
	err = row.Scan(&sequence)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	return err
}

func (s *Store) MaterializeNextBatch(ctx context.Context, limit int) (StoredBatch, error) {
	if s == nil {
		return StoredBatch{}, fmt.Errorf("rollup store is not configured")
	}
	if limit <= 0 {
		limit = 256
	}

	lastBatch, hasLast, err := s.latestBatch(ctx)
	if err != nil {
		return StoredBatch{}, err
	}

	entries, err := s.loadJournalEntriesAfter(ctx, lastBatch.LastSequence, limit)
	if err != nil {
		return StoredBatch{}, err
	}
	if len(entries) == 0 {
		return StoredBatch{}, ErrNoPendingBatch
	}

	inputData, inputHash, err := EncodeBatchInput(entries)
	if err != nil {
		return StoredBatch{}, err
	}

	existing, err := s.ListBatches(ctx)
	if err != nil {
		return StoredBatch{}, err
	}
	replayBatches := append(existing, StoredBatch{
		EncodingVersion: BatchEncodingVersion,
		InputData:       inputData,
		InputHash:       inputHash,
		PrevStateRoot:   ZeroStateRoot(),
	})
	if hasLast {
		replayBatches[len(replayBatches)-1].PrevStateRoot = lastBatch.StateRoot
	}
	roots, err := ReplayStoredBatches(replayBatches)
	if err != nil {
		return StoredBatch{}, err
	}

	batch := StoredBatch{
		EncodingVersion:      BatchEncodingVersion,
		FirstSequence:        entries[0].Sequence,
		LastSequence:         entries[len(entries)-1].Sequence,
		EntryCount:           len(entries),
		InputData:            inputData,
		InputHash:            inputHash,
		PrevStateRoot:        ZeroStateRoot(),
		BalancesRoot:         roots.BalancesRoot,
		OrdersRoot:           roots.OrdersRoot,
		PositionsFundingRoot: roots.PositionsFundingRoot,
		WithdrawalsRoot:      roots.WithdrawalsRoot,
		StateRoot:            roots.StateRoot,
	}
	if hasLast {
		batch.PrevStateRoot = lastBatch.StateRoot
	}

	row := s.db.QueryRowContext(ctx, `
		INSERT INTO rollup_shadow_batches (
			encoding_version, first_sequence_no, last_sequence_no, entry_count,
			input_data, input_hash, prev_state_root, balances_root,
			orders_root, positions_funding_root, withdrawals_root, state_root,
			created_at
		)
		VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11, $12,
			EXTRACT(EPOCH FROM NOW())::BIGINT
		)
		RETURNING batch_id, created_at
	`, batch.EncodingVersion, batch.FirstSequence, batch.LastSequence, batch.EntryCount, batch.InputData, batch.InputHash, batch.PrevStateRoot, batch.BalancesRoot, batch.OrdersRoot, batch.PositionsFundingRoot, batch.WithdrawalsRoot, batch.StateRoot)
	if err := row.Scan(&batch.BatchID, &batch.CreatedAt); err != nil {
		return StoredBatch{}, err
	}
	return batch, nil
}

func (s *Store) PrepareNextSubmission(ctx context.Context, limit int) (PreparedShadowSubmission, error) {
	if s == nil {
		return PreparedShadowSubmission{}, fmt.Errorf("rollup store is not configured")
	}

	batches, err := s.ListBatches(ctx)
	if err != nil {
		return PreparedShadowSubmission{}, err
	}
	submissions, err := s.ListSubmissions(ctx)
	if err != nil {
		return PreparedShadowSubmission{}, err
	}

	submissionByBatch := make(map[int64]StoredSubmission, len(submissions))
	for _, submission := range submissions {
		submissionByBatch[submission.BatchID] = submission
	}

	targetIndex := -1
	for index, batch := range batches {
		if _, exists := submissionByBatch[batch.BatchID]; !exists {
			targetIndex = index
			break
		}
	}

	if targetIndex == -1 {
		batch, err := s.MaterializeNextBatch(ctx, limit)
		if err != nil {
			if errors.Is(err, ErrNoPendingBatch) {
				return PreparedShadowSubmission{}, ErrNoPendingSubmission
			}
			return PreparedShadowSubmission{}, err
		}
		batches = append(batches, batch)
		targetIndex = len(batches) - 1
	}

	targetBatch := batches[targetIndex]
	bundle, err := BuildShadowBatchSubmissionBundle(append([]StoredBatch(nil), batches[:targetIndex]...), targetBatch)
	if err != nil {
		return PreparedShadowSubmission{}, err
	}
	stored, err := s.upsertSubmission(ctx, targetBatch.BatchID, bundle)
	if err != nil {
		return PreparedShadowSubmission{}, err
	}
	return PreparedShadowSubmission{
		StoredSubmission: stored,
		Bundle:           bundle,
	}, nil
}

func (s *Store) ListBatches(ctx context.Context) ([]StoredBatch, error) {
	if s == nil {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT batch_id, encoding_version, first_sequence_no, last_sequence_no,
		       entry_count, input_data, input_hash, prev_state_root,
		       balances_root, orders_root, positions_funding_root,
		       withdrawals_root, state_root, created_at
		FROM rollup_shadow_batches
		ORDER BY batch_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var batches []StoredBatch
	for rows.Next() {
		var batch StoredBatch
		if err := rows.Scan(
			&batch.BatchID,
			&batch.EncodingVersion,
			&batch.FirstSequence,
			&batch.LastSequence,
			&batch.EntryCount,
			&batch.InputData,
			&batch.InputHash,
			&batch.PrevStateRoot,
			&batch.BalancesRoot,
			&batch.OrdersRoot,
			&batch.PositionsFundingRoot,
			&batch.WithdrawalsRoot,
			&batch.StateRoot,
			&batch.CreatedAt,
		); err != nil {
			return nil, err
		}
		batches = append(batches, batch)
	}
	return batches, rows.Err()
}

func (s *Store) ListSubmissions(ctx context.Context) ([]StoredSubmission, error) {
	if s == nil {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+submissionSelectColumns+`
		FROM rollup_shadow_submissions
		ORDER BY batch_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var submissions []StoredSubmission
	for rows.Next() {
		var submission StoredSubmission
		if err := scanStoredSubmission(rows, &submission); err != nil {
			return nil, err
		}
		submissions = append(submissions, submission)
	}
	return submissions, rows.Err()
}

func (s *Store) RollupFrozen(ctx context.Context) (bool, error) {
	if s == nil {
		return false, nil
	}
	var frozen bool
	err := s.db.QueryRowContext(ctx, `
		SELECT frozen
		FROM rollup_freeze_state
		ORDER BY id DESC
		LIMIT 1
	`).Scan(&frozen)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return frozen, nil
}

func (s *Store) MaterializeAcceptedSubmissions(ctx context.Context) ([]AcceptedSubmissionMaterialization, error) {
	if s == nil {
		return nil, fmt.Errorf("rollup store is not configured")
	}
	submissions, err := s.ListSubmissions(ctx)
	if err != nil {
		return nil, err
	}
	materialized := make([]AcceptedSubmissionMaterialization, 0)
	for _, submission := range submissions {
		if submission.Status != SubmissionStatusAccepted {
			continue
		}
		item, err := s.MaterializeAcceptedSubmission(ctx, submission.SubmissionID)
		if err != nil {
			return nil, err
		}
		materialized = append(materialized, item)
	}
	return materialized, nil
}

func (s *Store) MaterializeAcceptedSubmission(ctx context.Context, submissionID string) (AcceptedSubmissionMaterialization, error) {
	if s == nil {
		return AcceptedSubmissionMaterialization{}, fmt.Errorf("rollup store is not configured")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AcceptedSubmissionMaterialization{}, err
	}
	defer tx.Rollback()

	submission, err := s.loadSubmissionByID(ctx, tx, submissionID)
	if err != nil {
		return AcceptedSubmissionMaterialization{}, err
	}
	if submission.Status != SubmissionStatusAccepted {
		return AcceptedSubmissionMaterialization{}, fmt.Errorf("submission %s is not accepted", strings.TrimSpace(submissionID))
	}

	batch, err := s.loadBatchByID(ctx, tx, submission.BatchID)
	if err != nil {
		return AcceptedSubmissionMaterialization{}, err
	}

	acceptedBatch, err := s.upsertAcceptedBatch(ctx, tx, batch, submission)
	if err != nil {
		return AcceptedSubmissionMaterialization{}, err
	}

	withdrawals, err := ExtractAcceptedWithdrawals(batch)
	if err != nil {
		return AcceptedSubmissionMaterialization{}, err
	}
	acceptedWithdrawals := make([]AcceptedWithdrawalRecord, 0, len(withdrawals))
	queuedClaimRefs := make([]string, 0, len(withdrawals))
	for _, withdrawal := range withdrawals {
		storedWithdrawal, claimQueued, err := s.upsertAcceptedWithdrawal(ctx, tx, withdrawal)
		if err != nil {
			return AcceptedSubmissionMaterialization{}, err
		}
		if claimQueued {
			queuedClaimRefs = append(queuedClaimRefs, storedWithdrawal.WithdrawalID)
		}
		acceptedWithdrawals = append(acceptedWithdrawals, storedWithdrawal)
	}

	replayBatches, err := s.listAcceptedBatchesForReplay(ctx, tx)
	if err != nil {
		return AcceptedSubmissionMaterialization{}, err
	}
	snapshot, err := BuildAcceptedReplaySnapshot(replayBatches)
	if err != nil {
		return AcceptedSubmissionMaterialization{}, err
	}
	latestAccepted, err := s.latestAcceptedBatch(ctx, tx)
	if err != nil {
		return AcceptedSubmissionMaterialization{}, err
	}
	if latestAccepted.BatchID > 0 {
		if snapshot.BatchID != latestAccepted.BatchID {
			return AcceptedSubmissionMaterialization{}, fmt.Errorf("accepted replay batch mismatch: have %d want %d", snapshot.BatchID, latestAccepted.BatchID)
		}
		if snapshot.Roots.StateRoot != latestAccepted.NextStateRoot {
			return AcceptedSubmissionMaterialization{}, fmt.Errorf("accepted replay state_root mismatch: have %s want %s", snapshot.Roots.StateRoot, latestAccepted.NextStateRoot)
		}
	}
	if err := s.replaceAcceptedReadTruth(ctx, tx, snapshot, latestAccepted.AcceptedAt); err != nil {
		return AcceptedSubmissionMaterialization{}, err
	}

	if err := tx.Commit(); err != nil {
		return AcceptedSubmissionMaterialization{}, err
	}

	return AcceptedSubmissionMaterialization{
		Batch:               acceptedBatch,
		AcceptedWithdrawals: acceptedWithdrawals,
		AcceptedBalances:    snapshot.Balances,
		AcceptedPositions:   snapshot.Positions,
		AcceptedPayouts:     snapshot.Payouts,
		QueuedClaimRefs:     queuedClaimRefs,
	}, nil
}

func (s *Store) latestBatch(ctx context.Context) (StoredBatch, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT batch_id, encoding_version, first_sequence_no, last_sequence_no,
		       entry_count, input_data, input_hash, prev_state_root,
		       balances_root, orders_root, positions_funding_root,
		       withdrawals_root, state_root, created_at
		FROM rollup_shadow_batches
		ORDER BY batch_id DESC
		LIMIT 1
	`)
	var batch StoredBatch
	if err := row.Scan(
		&batch.BatchID,
		&batch.EncodingVersion,
		&batch.FirstSequence,
		&batch.LastSequence,
		&batch.EntryCount,
		&batch.InputData,
		&batch.InputHash,
		&batch.PrevStateRoot,
		&batch.BalancesRoot,
		&batch.OrdersRoot,
		&batch.PositionsFundingRoot,
		&batch.WithdrawalsRoot,
		&batch.StateRoot,
		&batch.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return StoredBatch{}, false, nil
		}
		return StoredBatch{}, false, err
	}
	return batch, true, nil
}

func (s *Store) loadSubmissionByID(ctx context.Context, q sqlQueryer, submissionID string) (StoredSubmission, error) {
	row := q.QueryRowContext(ctx, `
		SELECT `+submissionSelectColumns+`
		FROM rollup_shadow_submissions
		WHERE submission_id = $1
	`, strings.TrimSpace(submissionID))
	var submission StoredSubmission
	if err := scanStoredSubmission(row, &submission); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return StoredSubmission{}, fmt.Errorf("rollup submission %s not found", strings.TrimSpace(submissionID))
		}
		return StoredSubmission{}, err
	}
	return submission, nil
}

func (s *Store) loadBatchByID(ctx context.Context, q sqlQueryer, batchID int64) (StoredBatch, error) {
	row := q.QueryRowContext(ctx, `
		SELECT batch_id, encoding_version, first_sequence_no, last_sequence_no,
		       entry_count, input_data, input_hash, prev_state_root,
		       balances_root, orders_root, positions_funding_root,
		       withdrawals_root, state_root, created_at
		FROM rollup_shadow_batches
		WHERE batch_id = $1
	`, batchID)
	var batch StoredBatch
	if err := row.Scan(
		&batch.BatchID,
		&batch.EncodingVersion,
		&batch.FirstSequence,
		&batch.LastSequence,
		&batch.EntryCount,
		&batch.InputData,
		&batch.InputHash,
		&batch.PrevStateRoot,
		&batch.BalancesRoot,
		&batch.OrdersRoot,
		&batch.PositionsFundingRoot,
		&batch.WithdrawalsRoot,
		&batch.StateRoot,
		&batch.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return StoredBatch{}, fmt.Errorf("rollup batch %d not found", batchID)
		}
		return StoredBatch{}, err
	}
	return batch, nil
}

func (s *Store) listAcceptedBatchesForReplay(ctx context.Context, q sqlQueryer) ([]StoredBatch, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT b.batch_id, b.encoding_version, b.first_sequence_no, b.last_sequence_no,
		       b.entry_count, b.input_data, b.input_hash, b.prev_state_root,
		       b.balances_root, b.orders_root, b.positions_funding_root,
		       b.withdrawals_root, b.state_root, b.created_at
		FROM rollup_shadow_batches b
		INNER JOIN rollup_accepted_batches ab ON ab.batch_id = b.batch_id
		ORDER BY b.batch_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var batches []StoredBatch
	for rows.Next() {
		var batch StoredBatch
		if err := rows.Scan(
			&batch.BatchID,
			&batch.EncodingVersion,
			&batch.FirstSequence,
			&batch.LastSequence,
			&batch.EntryCount,
			&batch.InputData,
			&batch.InputHash,
			&batch.PrevStateRoot,
			&batch.BalancesRoot,
			&batch.OrdersRoot,
			&batch.PositionsFundingRoot,
			&batch.WithdrawalsRoot,
			&batch.StateRoot,
			&batch.CreatedAt,
		); err != nil {
			return nil, err
		}
		batches = append(batches, batch)
	}
	return batches, rows.Err()
}

func (s *Store) latestAcceptedBatch(ctx context.Context, q sqlQueryer) (AcceptedBatchRecord, error) {
	row := q.QueryRowContext(ctx, `
		SELECT batch_id, submission_id, encoding_version, first_sequence_no, last_sequence_no,
		       entry_count, batch_data_hash, prev_state_root, balances_root, orders_root,
		       positions_funding_root, withdrawals_root, next_state_root, record_tx_hash,
		       accept_tx_hash, accepted_at, created_at, updated_at
		FROM rollup_accepted_batches
		ORDER BY batch_id DESC
		LIMIT 1
	`)
	var item AcceptedBatchRecord
	if err := row.Scan(
		&item.BatchID,
		&item.SubmissionID,
		&item.EncodingVersion,
		&item.FirstSequence,
		&item.LastSequence,
		&item.EntryCount,
		&item.BatchDataHash,
		&item.PrevStateRoot,
		&item.BalancesRoot,
		&item.OrdersRoot,
		&item.PositionsFundingRoot,
		&item.WithdrawalsRoot,
		&item.NextStateRoot,
		&item.RecordTxHash,
		&item.AcceptTxHash,
		&item.AcceptedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AcceptedBatchRecord{}, nil
		}
		return AcceptedBatchRecord{}, err
	}
	return item, nil
}

func (s *Store) loadJournalEntriesAfter(ctx context.Context, afterSequence int64, limit int) ([]JournalEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT sequence_no, entry_id, entry_type, source_type, source_ref,
		       occurred_at_millis, payload
		FROM rollup_shadow_journal_entries
		WHERE sequence_no > $1
		ORDER BY sequence_no
		LIMIT $2
	`, afterSequence, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []JournalEntry
	for rows.Next() {
		var entry JournalEntry
		if err := rows.Scan(
			&entry.Sequence,
			&entry.EntryID,
			&entry.EntryType,
			&entry.SourceType,
			&entry.SourceRef,
			&entry.OccurredAtMillis,
			&entry.Payload,
		); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (s *Store) upsertSubmission(ctx context.Context, batchID int64, bundle ShadowBatchSubmissionBundle) (StoredSubmission, error) {
	submissionData, submissionHash, err := buildSubmissionHash(bundle)
	if err != nil {
		return StoredSubmission{}, err
	}
	submissionID := fmt.Sprintf("rsub_%d", batchID)

	batchDataHash, err := solidityBytes32(bundle.Batch.InputHash, "bundle.batch.input_hash")
	if err != nil {
		return StoredSubmission{}, err
	}
	nextStateRoot, err := solidityBytes32(bundle.Batch.NextStateRoot, "bundle.batch.next_state_root")
	if err != nil {
		return StoredSubmission{}, err
	}

	row := s.db.QueryRowContext(ctx, `
		INSERT INTO rollup_shadow_submissions (
			submission_id, batch_id, encoding_version, status,
			batch_data_hash, next_state_root, auth_proof_hash,
			verifier_gate_hash, record_calldata, accept_calldata,
			submission_data, submission_hash, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4,
			$5, $6, $7,
			$8, $9, $10,
			$11::jsonb, $12, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
		)
		ON CONFLICT (batch_id) DO UPDATE SET
			encoding_version = EXCLUDED.encoding_version,
			status = EXCLUDED.status,
			batch_data_hash = EXCLUDED.batch_data_hash,
			next_state_root = EXCLUDED.next_state_root,
			auth_proof_hash = EXCLUDED.auth_proof_hash,
			verifier_gate_hash = EXCLUDED.verifier_gate_hash,
			record_calldata = EXCLUDED.record_calldata,
			accept_calldata = EXCLUDED.accept_calldata,
			submission_data = EXCLUDED.submission_data,
			submission_hash = EXCLUDED.submission_hash,
			updated_at = EXCLUDED.updated_at
		RETURNING `+submissionSelectColumns+`
	`,
		submissionID,
		batchID,
		SubmissionEncodingVersion,
		bundle.Status,
		batchDataHash,
		nextStateRoot,
		bundle.VerifierArtifactBundle.AuthProofDigest.AuthProofHash,
		bundle.VerifierArtifactBundle.VerifierGateDigest.VerifierGateHash,
		bundle.RecordBatchMetadataCall.Calldata,
		bundle.AcceptVerifiedBatchCall.Calldata,
		submissionData,
		submissionHash,
	)

	var submission StoredSubmission
	if err := scanStoredSubmission(row, &submission); err != nil {
		return StoredSubmission{}, err
	}
	return submission, nil
}

func (s *Store) MarkSubmissionRecordSubmitted(ctx context.Context, submissionID, txHash string) (StoredSubmission, error) {
	return s.updateSubmissionRuntime(ctx, submissionID, `
		UPDATE rollup_shadow_submissions
		SET status = $2,
		    record_tx_hash = $3,
		    record_submitted_at = EXTRACT(EPOCH FROM NOW())::BIGINT,
		    last_error = '',
		    last_error_at = 0,
		    updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE submission_id = $1
		RETURNING `+submissionSelectColumns, SubmissionStatusRecordSubmitted, normalizeSubmissionTxHash(txHash))
}

func (s *Store) MarkSubmissionAcceptSubmitted(ctx context.Context, submissionID, txHash string) (StoredSubmission, error) {
	return s.updateSubmissionRuntime(ctx, submissionID, `
		UPDATE rollup_shadow_submissions
		SET status = $2,
		    accept_tx_hash = $3,
		    accept_submitted_at = EXTRACT(EPOCH FROM NOW())::BIGINT,
		    last_error = '',
		    last_error_at = 0,
		    updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE submission_id = $1
		RETURNING `+submissionSelectColumns, SubmissionStatusAcceptSubmitted, normalizeSubmissionTxHash(txHash))
}

func (s *Store) MarkSubmissionAccepted(ctx context.Context, submissionID string) (StoredSubmission, error) {
	return s.updateSubmissionRuntime(ctx, submissionID, `
		UPDATE rollup_shadow_submissions
		SET status = $2,
		    accepted_at = EXTRACT(EPOCH FROM NOW())::BIGINT,
		    last_error = '',
		    last_error_at = 0,
		    updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE submission_id = $1
		RETURNING `+submissionSelectColumns, SubmissionStatusAccepted)
}

func (s *Store) MarkSubmissionFailed(ctx context.Context, submissionID, errMsg string) (StoredSubmission, error) {
	return s.updateSubmissionRuntime(ctx, submissionID, `
		UPDATE rollup_shadow_submissions
		SET status = $2,
		    last_error = $3,
		    last_error_at = EXTRACT(EPOCH FROM NOW())::BIGINT,
		    updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE submission_id = $1
		RETURNING `+submissionSelectColumns, SubmissionStatusFailed, normalizeSubmissionError(errMsg))
}

func (s *Store) RecordSubmissionError(ctx context.Context, submissionID, errMsg string) (StoredSubmission, error) {
	return s.updateSubmissionRuntime(ctx, submissionID, `
		UPDATE rollup_shadow_submissions
		SET last_error = $2,
		    last_error_at = EXTRACT(EPOCH FROM NOW())::BIGINT,
		    updated_at = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE submission_id = $1
		RETURNING `+submissionSelectColumns, normalizeSubmissionError(errMsg))
}

func (s *Store) updateSubmissionRuntime(ctx context.Context, submissionID, query string, args ...any) (StoredSubmission, error) {
	if s == nil {
		return StoredSubmission{}, fmt.Errorf("rollup store is not configured")
	}
	row := s.db.QueryRowContext(ctx, query, append([]any{submissionID}, args...)...)
	var submission StoredSubmission
	if err := scanStoredSubmission(row, &submission); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return StoredSubmission{}, fmt.Errorf("rollup submission %s not found", strings.TrimSpace(submissionID))
		}
		return StoredSubmission{}, err
	}
	return submission, nil
}

func (s *Store) upsertAcceptedBatch(ctx context.Context, q sqlQueryer, batch StoredBatch, submission StoredSubmission) (AcceptedBatchRecord, error) {
	row := q.QueryRowContext(ctx, `
		INSERT INTO rollup_accepted_batches (
			batch_id, submission_id, encoding_version, first_sequence_no, last_sequence_no,
			entry_count, batch_data_hash, prev_state_root, balances_root, orders_root,
			positions_funding_root, withdrawals_root, next_state_root, record_tx_hash,
			accept_tx_hash, accepted_at, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14,
			$15, $16, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
		)
		ON CONFLICT (batch_id) DO UPDATE SET
			submission_id = EXCLUDED.submission_id,
			encoding_version = EXCLUDED.encoding_version,
			first_sequence_no = EXCLUDED.first_sequence_no,
			last_sequence_no = EXCLUDED.last_sequence_no,
			entry_count = EXCLUDED.entry_count,
			batch_data_hash = EXCLUDED.batch_data_hash,
			prev_state_root = EXCLUDED.prev_state_root,
			balances_root = EXCLUDED.balances_root,
			orders_root = EXCLUDED.orders_root,
			positions_funding_root = EXCLUDED.positions_funding_root,
			withdrawals_root = EXCLUDED.withdrawals_root,
			next_state_root = EXCLUDED.next_state_root,
			record_tx_hash = EXCLUDED.record_tx_hash,
			accept_tx_hash = EXCLUDED.accept_tx_hash,
			accepted_at = EXCLUDED.accepted_at,
			updated_at = EXCLUDED.updated_at
		RETURNING batch_id, submission_id, encoding_version, first_sequence_no, last_sequence_no,
		          entry_count, batch_data_hash, prev_state_root, balances_root, orders_root,
		          positions_funding_root, withdrawals_root, next_state_root, record_tx_hash,
		          accept_tx_hash, accepted_at, created_at, updated_at
	`,
		batch.BatchID,
		submission.SubmissionID,
		batch.EncodingVersion,
		batch.FirstSequence,
		batch.LastSequence,
		batch.EntryCount,
		batch.InputHash,
		batch.PrevStateRoot,
		batch.BalancesRoot,
		batch.OrdersRoot,
		batch.PositionsFundingRoot,
		batch.WithdrawalsRoot,
		batch.StateRoot,
		submission.RecordTxHash,
		submission.AcceptTxHash,
		submission.AcceptedAt,
	)
	var item AcceptedBatchRecord
	if err := row.Scan(
		&item.BatchID,
		&item.SubmissionID,
		&item.EncodingVersion,
		&item.FirstSequence,
		&item.LastSequence,
		&item.EntryCount,
		&item.BatchDataHash,
		&item.PrevStateRoot,
		&item.BalancesRoot,
		&item.OrdersRoot,
		&item.PositionsFundingRoot,
		&item.WithdrawalsRoot,
		&item.NextStateRoot,
		&item.RecordTxHash,
		&item.AcceptTxHash,
		&item.AcceptedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return AcceptedBatchRecord{}, err
	}
	return item, nil
}

func (s *Store) upsertAcceptedWithdrawal(ctx context.Context, q sqlQueryer, withdrawal AcceptedWithdrawalRecord) (AcceptedWithdrawalRecord, bool, error) {
	row := q.QueryRowContext(ctx, `
		INSERT INTO rollup_accepted_withdrawals (
			withdrawal_id, batch_id, account_id, wallet_address, recipient_address, vault_address,
			asset, amount, lane, chain_name, network_name, request_sequence,
			claim_id, claim_status, claim_tx_hash, claim_submitted_at, claimed_at,
			last_error, last_error_at, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12,
			$13, $14, '', 0, 0,
			'', 0, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT
		)
		ON CONFLICT (withdrawal_id) DO UPDATE SET
			batch_id = EXCLUDED.batch_id,
			account_id = EXCLUDED.account_id,
			wallet_address = EXCLUDED.wallet_address,
			recipient_address = EXCLUDED.recipient_address,
			vault_address = EXCLUDED.vault_address,
			asset = EXCLUDED.asset,
			amount = EXCLUDED.amount,
			lane = EXCLUDED.lane,
			chain_name = EXCLUDED.chain_name,
			network_name = EXCLUDED.network_name,
			request_sequence = EXCLUDED.request_sequence,
			claim_id = EXCLUDED.claim_id,
			updated_at = EXCLUDED.updated_at
		RETURNING withdrawal_id, batch_id, account_id, wallet_address, recipient_address, vault_address,
		          asset, amount, lane, chain_name, network_name, request_sequence,
		          claim_id, claim_status, claim_tx_hash, claim_submitted_at, claimed_at,
		          last_error, last_error_at, created_at, updated_at
	`,
		withdrawal.WithdrawalID,
		withdrawal.BatchID,
		withdrawal.AccountID,
		withdrawal.WalletAddress,
		withdrawal.RecipientAddress,
		withdrawal.VaultAddress,
		withdrawal.Asset,
		withdrawal.Amount,
		withdrawal.Lane,
		withdrawal.ChainName,
		withdrawal.NetworkName,
		withdrawal.RequestSequence,
		withdrawal.ClaimID,
		AcceptedWithdrawalStatusClaimable,
	)
	var stored AcceptedWithdrawalRecord
	if err := row.Scan(
		&stored.WithdrawalID,
		&stored.BatchID,
		&stored.AccountID,
		&stored.WalletAddress,
		&stored.RecipientAddress,
		&stored.VaultAddress,
		&stored.Asset,
		&stored.Amount,
		&stored.Lane,
		&stored.ChainName,
		&stored.NetworkName,
		&stored.RequestSequence,
		&stored.ClaimID,
		&stored.ClaimStatus,
		&stored.ClaimTxHash,
		&stored.ClaimSubmittedAt,
		&stored.ClaimedAt,
		&stored.LastError,
		&stored.LastErrorAt,
		&stored.CreatedAt,
		&stored.UpdatedAt,
	); err != nil {
		return AcceptedWithdrawalRecord{}, false, err
	}

	payload, err := json.Marshal(acceptedWithdrawalClaimPayload{
		EventID:          stored.WithdrawalID,
		UserID:           stored.AccountID,
		WalletAddress:    stored.WalletAddress,
		RecipientAddress: stored.RecipientAddress,
		PayoutAsset:      stored.Asset,
		PayoutAmount:     stored.Amount,
	})
	if err != nil {
		return AcceptedWithdrawalRecord{}, false, err
	}

	result, err := q.ExecContext(ctx, `
		INSERT INTO chain_transactions (
			biz_type, ref_id, chain_name, network_name, wallet_address, tx_hash,
			status, payload, error_message, attempt_count, created_at, updated_at
		)
		VALUES ('WITHDRAWAL_CLAIM', $1, $2, $3, $4, '', 'PENDING', $5, '', 0,
		        EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
		ON CONFLICT (biz_type, ref_id) DO NOTHING
	`, stored.WithdrawalID, stored.ChainName, stored.NetworkName, stored.WalletAddress, payload)
	if err != nil {
		return AcceptedWithdrawalRecord{}, false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return AcceptedWithdrawalRecord{}, false, err
	}
	return stored, rowsAffected > 0, nil
}

func (s *Store) replaceAcceptedReadTruth(ctx context.Context, q sqlQueryer, snapshot AcceptedReplaySnapshot, acceptedAt int64) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM rollup_accepted_balances`); err != nil {
		return err
	}
	if _, err := q.ExecContext(ctx, `DELETE FROM rollup_accepted_positions`); err != nil {
		return err
	}
	if _, err := q.ExecContext(ctx, `DELETE FROM rollup_accepted_payouts`); err != nil {
		return err
	}

	for _, item := range snapshot.Balances {
		if _, err := q.ExecContext(ctx, `
			INSERT INTO rollup_accepted_balances (
				batch_id, account_id, asset, available, frozen, sequence_no, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		`, snapshot.BatchID, item.AccountID, item.Asset, item.Available, item.Frozen, item.SequenceNo, acceptedAt); err != nil {
			return err
		}
	}
	for _, item := range snapshot.Positions {
		if _, err := q.ExecContext(ctx, `
			INSERT INTO rollup_accepted_positions (
				batch_id, account_id, market_id, outcome, position_asset,
				quantity, settled_quantity, settlement_status, sequence_no, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
		`, snapshot.BatchID, item.AccountID, item.MarketID, item.Outcome, item.PositionAsset, item.Quantity, item.SettledQuantity, item.SettlementStatus, item.SequenceNo, acceptedAt); err != nil {
			return err
		}
	}
	for _, item := range snapshot.Payouts {
		if _, err := q.ExecContext(ctx, `
			INSERT INTO rollup_accepted_payouts (
				event_id, batch_id, market_id, user_id, winning_outcome,
				position_asset, settled_quantity, payout_asset, payout_amount,
				status, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11)
		`, item.EventID, item.BatchID, item.MarketID, item.UserID, item.WinningOutcome, item.PositionAsset, item.SettledQuantity, item.PayoutAsset, item.PayoutAmount, item.Status, acceptedAt); err != nil {
			return err
		}
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanStoredSubmission(scanner rowScanner, submission *StoredSubmission) error {
	return scanner.Scan(
		&submission.SubmissionID,
		&submission.BatchID,
		&submission.EncodingVersion,
		&submission.Status,
		&submission.BatchDataHash,
		&submission.NextStateRoot,
		&submission.AuthProofHash,
		&submission.VerifierGateHash,
		&submission.RecordCalldata,
		&submission.AcceptCalldata,
		&submission.SubmissionData,
		&submission.SubmissionHash,
		&submission.RecordTxHash,
		&submission.AcceptTxHash,
		&submission.RecordSubmittedAt,
		&submission.AcceptSubmittedAt,
		&submission.AcceptedAt,
		&submission.LastError,
		&submission.LastErrorAt,
		&submission.CreatedAt,
		&submission.UpdatedAt,
	)
}

func normalizeSubmissionTxHash(txHash string) string {
	trimmed := strings.ToLower(strings.TrimSpace(txHash))
	return strings.TrimPrefix(trimmed, "0x")
}

func normalizeSubmissionError(errMsg string) string {
	return strings.TrimSpace(errMsg)
}
