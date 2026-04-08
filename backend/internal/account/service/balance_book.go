package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"funnyoption/internal/account/model"
	sharedkafka "funnyoption/internal/shared/kafka"
)

var (
	ErrInvalidAsset        = errors.New("invalid asset")
	ErrInvalidAmount       = errors.New("invalid amount")
	ErrInsufficientBalance = errors.New("insufficient available balance")
	ErrFreezeNotFound      = errors.New("freeze record not found")
	ErrFreezeAlreadyClosed = errors.New("freeze record already closed")
	ErrFreezeAlreadyExists = errors.New("freeze record already exists")
)

type FreezeRequest struct {
	FreezeID string
	UserID   int64
	Asset    string
	RefType  string
	RefID    string
	Amount   int64
}

type CreditRequest struct {
	UserID  int64
	Asset   string
	Amount  int64
	RefType string
	RefID   string
}

type DebitRequest struct {
	UserID  int64
	Asset   string
	Amount  int64
	RefType string
	RefID   string
}

type BalanceStore interface {
	LoadBalances(ctx context.Context) ([]model.Balance, error)
	LoadFreezes(ctx context.Context) ([]model.FreezeRecord, error)
	UpsertBalance(ctx context.Context, balance model.Balance) error
	UpsertFreeze(ctx context.Context, record model.FreezeRecord) error
	UpsertBalanceAndFreeze(ctx context.Context, balance model.Balance, freeze model.FreezeRecord) error
	ApplyCreditEvent(ctx context.Context, req CreditRequest) (model.Balance, bool, error)
	ApplyDebitEvent(ctx context.Context, req DebitRequest) (model.Balance, bool, error)
}

// BalanceBook maintains mutable trading balances. It is intentionally separate
// from the append-only ledger journal, which remains the source of truth for
// replay and reconciliation.
type BalanceBook struct {
	mu            sync.RWMutex
	balances      map[string]*model.Balance
	freezes       map[string]*model.FreezeRecord
	processedRefs map[string]struct{}
	store         BalanceStore
}

func NewBalanceBook() *BalanceBook {
	return &BalanceBook{
		balances:      make(map[string]*model.Balance),
		freezes:       make(map[string]*model.FreezeRecord),
		processedRefs: make(map[string]struct{}),
	}
}

func NewBalanceBookWithStore(store BalanceStore) *BalanceBook {
	book := NewBalanceBook()
	book.store = store
	return book
}

func (b *BalanceBook) SeedBalance(userID int64, asset string, available int64) {
	b.mu.Lock()
	key := balanceKey(userID, asset)
	balance := &model.Balance{
		UserID:    userID,
		Asset:     normalizeAsset(asset),
		Available: available,
	}
	b.balances[key] = balance
	b.mu.Unlock()

	b.persistBalance(*balance)
}

func (b *BalanceBook) PreFreeze(req FreezeRequest) (*model.FreezeRecord, error) {
	if req.UserID <= 0 {
		return nil, ErrInvalidAmount
	}
	if req.Amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if normalizeAsset(req.Asset) == "" {
		return nil, ErrInvalidAsset
	}

	b.mu.Lock()
	key := balanceKey(req.UserID, req.Asset)
	balance, ok := b.balances[key]
	if !ok {
		balance = &model.Balance{
			UserID: req.UserID,
			Asset:  normalizeAsset(req.Asset),
		}
		b.balances[key] = balance
	}
	if balance.Available < req.Amount {
		b.mu.Unlock()
		return nil, ErrInsufficientBalance
	}

	freezeID := strings.TrimSpace(req.FreezeID)
	if freezeID == "" {
		freezeID = sharedkafka.NewID("frz")
	}
	if _, exists := b.freezes[freezeID]; exists {
		b.mu.Unlock()
		return nil, ErrFreezeAlreadyExists
	}
	balance.Available -= req.Amount
	balance.Frozen += req.Amount

	record := &model.FreezeRecord{
		FreezeID:       freezeID,
		UserID:         req.UserID,
		Asset:          normalizeAsset(req.Asset),
		RefType:        req.RefType,
		RefID:          req.RefID,
		OriginalAmount: req.Amount,
		Amount:         req.Amount,
	}
	b.freezes[freezeID] = record
	snapshotBalance := *balance
	snapshotFreeze := *record
	b.mu.Unlock()

	if err := b.persistBalanceAndFreeze(snapshotBalance, snapshotFreeze); err != nil {
		b.mu.Lock()
		balance.Available += req.Amount
		balance.Frozen -= req.Amount
		delete(b.freezes, freezeID)
		b.mu.Unlock()
		return nil, err
	}
	return &snapshotFreeze, nil
}

func (b *BalanceBook) ApplyExternalFreeze(req FreezeRequest) (*model.FreezeRecord, error) {
	if strings.TrimSpace(req.FreezeID) == "" {
		return nil, ErrFreezeNotFound
	}
	return b.PreFreeze(req)
}

func (b *BalanceBook) ReleaseFreeze(freezeID string) error {
	b.mu.Lock()
	record, balance, err := b.getActiveFreezeLocked(freezeID)
	if err != nil {
		b.mu.Unlock()
		return err
	}
	oldBalance := *balance
	oldRecord := *record
	balance.Frozen -= record.Amount
	balance.Available += record.Amount
	record.Amount = 0
	record.Released = true
	snapshotBalance := *balance
	snapshotFreeze := *record
	b.mu.Unlock()

	if err := b.persistBalanceAndFreeze(snapshotBalance, snapshotFreeze); err != nil {
		b.mu.Lock()
		*balance = oldBalance
		*record = oldRecord
		b.mu.Unlock()
		return err
	}
	return nil
}

func (b *BalanceBook) ReleaseFreezeAmount(freezeID string, amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	b.mu.Lock()
	record, balance, err := b.getActiveFreezeLocked(freezeID)
	if err != nil {
		b.mu.Unlock()
		return err
	}
	if amount > record.Amount {
		b.mu.Unlock()
		return fmt.Errorf("release amount exceeds frozen amount")
	}
	oldBalance := *balance
	oldRecord := *record
	balance.Frozen -= amount
	balance.Available += amount
	record.Amount -= amount
	if record.Amount == 0 {
		record.Released = true
	}
	snapshotBalance := *balance
	snapshotFreeze := *record
	b.mu.Unlock()

	if err := b.persistBalanceAndFreeze(snapshotBalance, snapshotFreeze); err != nil {
		b.mu.Lock()
		*balance = oldBalance
		*record = oldRecord
		b.mu.Unlock()
		return err
	}
	return nil
}

func (b *BalanceBook) ConsumeFreeze(freezeID string, amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	b.mu.Lock()
	record, balance, err := b.getActiveFreezeLocked(freezeID)
	if err != nil {
		b.mu.Unlock()
		return err
	}
	if amount > record.Amount {
		b.mu.Unlock()
		return fmt.Errorf("consume amount exceeds frozen amount")
	}
	oldBalance := *balance
	oldRecord := *record
	balance.Frozen -= amount
	record.Amount -= amount
	if record.Amount == 0 {
		record.Consumed = true
	}
	snapshotBalance := *balance
	snapshotFreeze := *record
	b.mu.Unlock()

	if err := b.persistBalanceAndFreeze(snapshotBalance, snapshotFreeze); err != nil {
		b.mu.Lock()
		*balance = oldBalance
		*record = oldRecord
		b.mu.Unlock()
		return err
	}
	return nil
}

func (b *BalanceBook) GetBalance(userID int64, asset string) model.Balance {
	b.mu.RLock()
	defer b.mu.RUnlock()

	key := balanceKey(userID, asset)
	balance, ok := b.balances[key]
	if !ok {
		return model.Balance{
			UserID: userID,
			Asset:  normalizeAsset(asset),
		}
	}
	return *balance
}

func (b *BalanceBook) CreditAvailable(userID int64, asset string, amount int64) error {
	if userID <= 0 || amount <= 0 {
		return ErrInvalidAmount
	}
	if normalizeAsset(asset) == "" {
		return ErrInvalidAsset
	}

	b.mu.Lock()
	key := balanceKey(userID, asset)
	balance, ok := b.balances[key]
	if !ok {
		balance = &model.Balance{
			UserID: userID,
			Asset:  normalizeAsset(asset),
		}
		b.balances[key] = balance
	}
	balance.Available += amount
	snapshotBalance := *balance
	b.mu.Unlock()

	if err := b.persistBalance(snapshotBalance); err != nil {
		b.mu.Lock()
		balance.Available -= amount
		b.mu.Unlock()
		return err
	}
	return nil
}

func (b *BalanceBook) CreditAvailableWithRef(req CreditRequest) (model.Balance, bool, error) {
	if req.UserID <= 0 || req.Amount <= 0 {
		return model.Balance{}, false, ErrInvalidAmount
	}
	if normalizeAsset(req.Asset) == "" {
		return model.Balance{}, false, ErrInvalidAsset
	}
	if strings.TrimSpace(req.RefType) == "" || strings.TrimSpace(req.RefID) == "" {
		return model.Balance{}, false, ErrInvalidAmount
	}

	if b.store != nil {
		balance, applied, err := b.store.ApplyCreditEvent(context.Background(), CreditRequest{
			UserID:  req.UserID,
			Asset:   normalizeAsset(req.Asset),
			Amount:  req.Amount,
			RefType: strings.ToUpper(strings.TrimSpace(req.RefType)),
			RefID:   strings.TrimSpace(req.RefID),
		})
		if err != nil {
			return model.Balance{}, false, err
		}
		b.mu.Lock()
		snapshot := balance
		b.balances[balanceKey(snapshot.UserID, snapshot.Asset)] = &snapshot
		b.mu.Unlock()
		return balance, applied, nil
	}

	refKey := strings.ToUpper(strings.TrimSpace(req.RefType)) + ":" + strings.TrimSpace(req.RefID)
	b.mu.RLock()
	_, seen := b.processedRefs[refKey]
	b.mu.RUnlock()
	if seen {
		return b.GetBalance(req.UserID, req.Asset), false, nil
	}

	if err := b.CreditAvailable(req.UserID, req.Asset, req.Amount); err != nil {
		return model.Balance{}, false, err
	}
	b.mu.Lock()
	b.processedRefs[refKey] = struct{}{}
	b.mu.Unlock()
	return b.GetBalance(req.UserID, req.Asset), true, nil
}

func (b *BalanceBook) DebitAvailable(userID int64, asset string, amount int64) error {
	if userID <= 0 || amount <= 0 {
		return ErrInvalidAmount
	}
	if normalizeAsset(asset) == "" {
		return ErrInvalidAsset
	}

	b.mu.Lock()
	key := balanceKey(userID, asset)
	balance, ok := b.balances[key]
	if !ok || balance.Available < amount {
		b.mu.Unlock()
		return ErrInsufficientBalance
	}
	balance.Available -= amount
	snapshotBalance := *balance
	b.mu.Unlock()

	if err := b.persistBalance(snapshotBalance); err != nil {
		b.mu.Lock()
		balance.Available += amount
		b.mu.Unlock()
		return err
	}
	return nil
}

func (b *BalanceBook) DebitAvailableWithRef(req DebitRequest) (model.Balance, bool, error) {
	if req.UserID <= 0 || req.Amount <= 0 {
		return model.Balance{}, false, ErrInvalidAmount
	}
	if normalizeAsset(req.Asset) == "" {
		return model.Balance{}, false, ErrInvalidAsset
	}
	if strings.TrimSpace(req.RefType) == "" || strings.TrimSpace(req.RefID) == "" {
		return model.Balance{}, false, ErrInvalidAmount
	}

	if b.store != nil {
		balance, applied, err := b.store.ApplyDebitEvent(context.Background(), DebitRequest{
			UserID:  req.UserID,
			Asset:   normalizeAsset(req.Asset),
			Amount:  req.Amount,
			RefType: strings.ToUpper(strings.TrimSpace(req.RefType)),
			RefID:   strings.TrimSpace(req.RefID),
		})
		if err != nil {
			return model.Balance{}, false, err
		}
		b.mu.Lock()
		snapshot := balance
		b.balances[balanceKey(snapshot.UserID, snapshot.Asset)] = &snapshot
		b.mu.Unlock()
		return balance, applied, nil
	}

	refKey := strings.ToUpper(strings.TrimSpace(req.RefType)) + ":" + strings.TrimSpace(req.RefID)
	b.mu.RLock()
	_, seen := b.processedRefs[refKey]
	b.mu.RUnlock()
	if seen {
		return b.GetBalance(req.UserID, req.Asset), false, nil
	}

	if err := b.DebitAvailable(req.UserID, req.Asset, req.Amount); err != nil {
		return model.Balance{}, false, err
	}
	b.mu.Lock()
	b.processedRefs[refKey] = struct{}{}
	b.mu.Unlock()
	return b.GetBalance(req.UserID, req.Asset), true, nil
}

func (b *BalanceBook) OutstandingFrozen(asset string) int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var total int64
	normalized := normalizeAsset(asset)
	for _, record := range b.freezes {
		if record.Asset != normalized {
			continue
		}
		if record.Released || record.Consumed {
			continue
		}
		total += record.Amount
	}
	return total
}

func (b *BalanceBook) FreezeAmount(freezeID string) (int64, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	record, ok := b.freezes[freezeID]
	if !ok {
		return 0, false
	}
	if record.Released || record.Consumed {
		return 0, true
	}
	return record.Amount, true
}

func (b *BalanceBook) getActiveFreezeLocked(freezeID string) (*model.FreezeRecord, *model.Balance, error) {
	record, ok := b.freezes[freezeID]
	if !ok {
		return nil, nil, ErrFreezeNotFound
	}
	if record.Released || record.Consumed {
		return nil, nil, ErrFreezeAlreadyClosed
	}

	key := balanceKey(record.UserID, record.Asset)
	balance, ok := b.balances[key]
	if !ok {
		return nil, nil, ErrFreezeNotFound
	}
	return record, balance, nil
}

func (b *BalanceBook) Hydrate(balances []model.Balance, freezes []model.FreezeRecord) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.balances = make(map[string]*model.Balance, len(balances))
	for _, item := range balances {
		balance := item
		balance.Asset = normalizeAsset(balance.Asset)
		b.balances[balanceKey(balance.UserID, balance.Asset)] = &balance
	}

	b.freezes = make(map[string]*model.FreezeRecord, len(freezes))
	for _, item := range freezes {
		record := item
		record.Asset = normalizeAsset(record.Asset)
		b.freezes[record.FreezeID] = &record
	}
}

func (b *BalanceBook) BalanceCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.balances)
}

func (b *BalanceBook) persistBalance(balance model.Balance) error {
	if b.store == nil {
		return nil
	}
	return b.store.UpsertBalance(context.Background(), balance)
}

func (b *BalanceBook) persistFreeze(record model.FreezeRecord) error {
	if b.store == nil {
		return nil
	}
	return b.store.UpsertFreeze(context.Background(), record)
}

func (b *BalanceBook) persistBalanceAndFreeze(balance model.Balance, freeze model.FreezeRecord) error {
	if b.store == nil {
		return nil
	}
	return b.store.UpsertBalanceAndFreeze(context.Background(), balance, freeze)
}

func balanceKey(userID int64, asset string) string {
	return fmt.Sprintf("%d:%s", userID, normalizeAsset(asset))
}

func normalizeAsset(asset string) string {
	return strings.ToUpper(strings.TrimSpace(asset))
}
