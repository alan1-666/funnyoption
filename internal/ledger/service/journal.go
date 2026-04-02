package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"funnyoption/internal/ledger/model"
	sharedkafka "funnyoption/internal/shared/kafka"
)

var (
	ErrEmptyEntry       = errors.New("ledger entry must contain postings")
	ErrInvalidPosting   = errors.New("ledger posting is invalid")
	ErrUnbalancedEntry  = errors.New("ledger entry is not balanced")
	ErrDuplicateEntryID = errors.New("ledger entry id already exists")
)

// Journal is append-only. Corrections must be recorded through compensating
// entries instead of mutating history.
type Journal struct {
	mu        sync.RWMutex
	entries   []model.Entry
	entrySeen map[string]struct{}
	store     JournalStore
}

func NewJournal() *Journal {
	return &Journal{
		entries:   make([]model.Entry, 0, 128),
		entrySeen: make(map[string]struct{}),
	}
}

func NewPersistentJournal(store JournalStore) *Journal {
	journal := NewJournal()
	journal.store = store
	return journal
}

func (j *Journal) Append(entry model.Entry) (model.Entry, error) {
	normalized, err := normalizeEntry(entry)
	if err != nil {
		return model.Entry{}, err
	}

	j.mu.Lock()
	defer j.mu.Unlock()

	if _, exists := j.entrySeen[normalized.EntryID]; exists {
		return model.Entry{}, ErrDuplicateEntryID
	}
	if j.store != nil {
		if err := j.store.AppendEntry(context.Background(), normalized); err != nil {
			return model.Entry{}, err
		}
	}

	j.entries = append(j.entries, normalized)
	j.entrySeen[normalized.EntryID] = struct{}{}
	return normalized, nil
}

func (j *Journal) Entries() []model.Entry {
	j.mu.RLock()
	defer j.mu.RUnlock()

	cloned := make([]model.Entry, len(j.entries))
	for i, entry := range j.entries {
		cloned[i] = cloneEntry(entry)
	}
	return cloned
}

func (j *Journal) BalanceOf(account, asset string) int64 {
	j.mu.RLock()
	defer j.mu.RUnlock()

	account = normalizeAccount(account)
	asset = normalizeAsset(asset)

	var balance int64
	for _, entry := range j.entries {
		for _, posting := range entry.Postings {
			if normalizeAccount(posting.Account) != account || normalizeAsset(posting.Asset) != asset {
				continue
			}
			switch posting.Direction {
			case model.DirectionCredit:
				balance += posting.Amount
			case model.DirectionDebit:
				balance -= posting.Amount
			}
		}
	}
	return balance
}

func normalizeEntry(entry model.Entry) (model.Entry, error) {
	if len(entry.Postings) == 0 {
		return model.Entry{}, ErrEmptyEntry
	}

	if strings.TrimSpace(entry.EntryID) == "" {
		entry.EntryID = sharedkafka.NewID("led")
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	if entry.Status == "" {
		entry.Status = model.EntryStatusConfirmed
	}

	debitByAsset := make(map[string]int64)
	creditByAsset := make(map[string]int64)
	postings := make([]model.Posting, 0, len(entry.Postings))

	for _, posting := range entry.Postings {
		normalized := model.Posting{
			Account:   normalizeAccount(posting.Account),
			Asset:     normalizeAsset(posting.Asset),
			Direction: posting.Direction,
			Amount:    posting.Amount,
		}
		if normalized.Account == "" || normalized.Asset == "" || normalized.Amount <= 0 {
			return model.Entry{}, ErrInvalidPosting
		}

		switch normalized.Direction {
		case model.DirectionDebit:
			debitByAsset[normalized.Asset] += normalized.Amount
		case model.DirectionCredit:
			creditByAsset[normalized.Asset] += normalized.Amount
		default:
			return model.Entry{}, ErrInvalidPosting
		}
		postings = append(postings, normalized)
	}

	for asset, debit := range debitByAsset {
		if creditByAsset[asset] != debit {
			return model.Entry{}, fmt.Errorf("%w: asset=%s debit=%d credit=%d", ErrUnbalancedEntry, asset, debit, creditByAsset[asset])
		}
	}
	for asset, credit := range creditByAsset {
		if debitByAsset[asset] != credit {
			return model.Entry{}, fmt.Errorf("%w: asset=%s debit=%d credit=%d", ErrUnbalancedEntry, asset, debitByAsset[asset], credit)
		}
	}

	entry.BizType = model.BizType(strings.ToUpper(strings.TrimSpace(string(entry.BizType))))
	entry.RefID = strings.TrimSpace(entry.RefID)
	entry.Postings = postings
	return entry, nil
}

func cloneEntry(entry model.Entry) model.Entry {
	cloned := entry
	cloned.Postings = append([]model.Posting(nil), entry.Postings...)
	return cloned
}

func normalizeAsset(asset string) string {
	return strings.ToUpper(strings.TrimSpace(asset))
}

func normalizeAccount(account string) string {
	return strings.ToLower(strings.TrimSpace(account))
}
