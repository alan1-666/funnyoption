package pipeline

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"

	"funnyoption/internal/matching/model"
	"funnyoption/internal/matching/ringbuffer"
)

const outputChSize = 8192

// BookSupervisor manages the lifecycle of per-book BookEngine instances.
// It routes commands by BookKey, lazily creating BookEngines on first contact.
type BookSupervisor struct {
	logger   *slog.Logger
	sequence uint64
	mu       sync.RWMutex
	books    map[string]*BookEngine
	outputCh chan MatchResult
	ctx      context.Context
	idle     *ringbuffer.IdleStrategy
}

func NewBookSupervisor(logger *slog.Logger) *BookSupervisor {
	return &BookSupervisor{
		logger:   logger,
		books:    make(map[string]*BookEngine, 64),
		outputCh: make(chan MatchResult, outputChSize),
	}
}

// Route sends a MatchCommand to the appropriate BookEngine, creating one if needed.
func (s *BookSupervisor) Route(cmd MatchCommand) bool {
	bookKey := cmd.BookKey

	s.mu.RLock()
	be, ok := s.books[bookKey]
	s.mu.RUnlock()

	if !ok {
		be = s.getOrCreate(bookKey)
	}
	return be.TryPublish(cmd)
}

// SubmitCancel injects a cancel command into the correct BookEngine.
func (s *BookSupervisor) SubmitCancel(cmd MatchCommand) bool {
	cmd.Action = ActionCancel
	return s.Route(cmd)
}

// OutputCh returns the shared fan-in channel that all BookEngines write to.
func (s *BookSupervisor) OutputCh() <-chan MatchResult {
	return s.outputCh
}

// Restore loads resting orders into the appropriate BookEngines (pre-Start).
func (s *BookSupervisor) Restore(sequence uint64, orders []*model.Order) error {
	atomic.StoreUint64(&s.sequence, sequence)
	grouped := make(map[string][]*model.Order, len(orders))
	for _, o := range orders {
		if o == nil {
			continue
		}
		key := o.BookKey()
		grouped[key] = append(grouped[key], o)
	}
	for bookKey, bookOrders := range grouped {
		be := s.getOrCreate(bookKey)
		for _, o := range bookOrders {
			if err := be.RestoreOrder(o); err != nil {
				return err
			}
		}
	}
	return nil
}

// Start launches all existing BookEngines' goroutines.
func (s *BookSupervisor) Start(ctx context.Context) {
	s.ctx = ctx
	s.mu.RLock()
	for _, be := range s.books {
		go be.Run(ctx)
	}
	count := len(s.books)
	s.mu.RUnlock()
	s.logger.Info("book supervisor started", "initial_books", count)
}

// Close shuts down all BookEngines by cancelling the context.
func (s *BookSupervisor) Close() {
	close(s.outputCh)
}

func (s *BookSupervisor) getOrCreate(bookKey string) *BookEngine {
	s.mu.Lock()
	defer s.mu.Unlock()

	if be, ok := s.books[bookKey]; ok {
		return be
	}
	be := NewBookEngine(bookKey, s.logger, &s.sequence, s.outputCh)
	s.books[bookKey] = be

	if s.ctx != nil {
		go be.Run(s.ctx)
	}
	return be
}

// BookCount returns the number of active book engines.
func (s *BookSupervisor) BookCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.books)
}

// Stats aggregates stats across all BookEngines.
func (s *BookSupervisor) Stats() (totalMatched, totalBatches uint64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, be := range s.books {
		m, b, _ := be.Stats()
		totalMatched += m
		totalBatches += b
	}
	return
}
