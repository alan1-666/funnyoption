package service

import (
	"context"
	"database/sql"

	"funnyoption/internal/ledger/model"
)

type JournalStore interface {
	AppendEntry(ctx context.Context, entry model.Entry) error
}

type SQLStore struct {
	db *sql.DB
}

func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) AppendEntry(ctx context.Context, entry model.Entry) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO ledger_entries (entry_id, biz_type, ref_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (entry_id) DO NOTHING
	`, entry.EntryID, string(entry.BizType), entry.RefID, string(entry.Status), entry.CreatedAt.Unix())
	if err != nil {
		return err
	}

	for _, posting := range entry.Postings {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO ledger_postings (entry_id, account_ref, asset, direction, amount, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, entry.EntryID, posting.Account, posting.Asset, string(posting.Direction), posting.Amount, entry.CreatedAt.Unix()); err != nil {
			return err
		}
	}

	return tx.Commit()
}
