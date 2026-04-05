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

type sqlQueryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type Store struct {
	db *sql.DB
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
