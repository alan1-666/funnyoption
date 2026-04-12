package engine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"funnyoption/internal/matching/model"
)

type Result struct {
	Order    *model.Order
	Affected []*model.Order
	Trades   []model.Trade
	Book     model.BookSnapshot
}

type CancelResult struct {
	Orders []*model.Order
	Books  []model.BookSnapshot
}

type placeOrderRequest struct {
	order *model.Order
	reply chan resultEnvelope
}

type cancelOrdersRequest struct {
	orders []*model.Order
	reason model.CancelReason
	reply  chan cancelEnvelope
}

type asyncRequest struct {
	place  *placeOrderRequest
	cancel *cancelOrdersRequest
}

type resultEnvelope struct {
	result Result
	err    error
}

type cancelEnvelope struct {
	result CancelResult
	err    error
}

type Engine struct {
	logger   *slog.Logger
	books    map[string]*model.OrderBookDirect
	sequence *uint64
	localSeq uint64

	// Reusable buffers to avoid per-match allocations.
	tradesBuf   []model.Trade
	affectedBuf []*model.Order
	removeBuf   []*model.DirectOrder
}

type bookWorker struct {
	bookKey string
	engine  *Engine
	ch      chan asyncRequest
}

type AsyncEngine struct {
	logger   *slog.Logger
	sequence uint64
	mu       sync.RWMutex
	workers  map[string]*bookWorker
	buffer   int
	ctx      context.Context
}

func New(logger *slog.Logger) *Engine {
	seq := uint64(0)
	return &Engine{
		logger:   logger,
		books:    make(map[string]*model.OrderBookDirect),
		sequence: &seq,
	}
}

func (e *Engine) SetSequence(seq uint64) {
	atomic.StoreUint64(e.sequence, seq)
}

func (e *Engine) LocalSeq() uint64 {
	return e.localSeq
}

func (e *Engine) SetLocalSeq(seq uint64) {
	e.localSeq = seq
}

func (e *Engine) BookCount() int {
	count := 0
	for _, book := range e.books {
		if book.OrderCount() > 0 {
			count++
		}
	}
	return count
}

func NewWithSequence(logger *slog.Logger, sequence *uint64) *Engine {
	return &Engine{
		logger:   logger,
		books:    make(map[string]*model.OrderBookDirect),
		sequence: sequence,
	}
}

func NewAsync(logger *slog.Logger, buffer int) *AsyncEngine {
	if buffer <= 0 {
		buffer = 1024
	}
	return &AsyncEngine{
		logger:  logger,
		workers: make(map[string]*bookWorker),
		buffer:  buffer,
	}
}

func (e *AsyncEngine) Restore(sequence uint64, orders []*model.Order) error {
	atomic.StoreUint64(&e.sequence, sequence)
	for _, order := range orders {
		if order == nil {
			continue
		}
		w := e.getOrCreateWorkerLocked(order.BookKey())
		if err := w.engine.RestoreOrder(order); err != nil {
			return err
		}
	}
	return nil
}

func (e *AsyncEngine) Start(ctx context.Context) {
	e.ctx = ctx
	e.mu.RLock()
	for _, w := range e.workers {
		startWorker(ctx, w)
	}
	e.mu.RUnlock()
	e.logger.Info("matching sharded engine started", "workers", len(e.workers), "buffer_per_book", e.buffer)
}

func startWorker(ctx context.Context, w *bookWorker) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case req := <-w.ch:
				switch {
				case req.place != nil:
					result, err := w.engine.PlaceOrder(req.place.order)
					req.place.reply <- resultEnvelope{result: result, err: err}
				case req.cancel != nil:
					result, err := w.engine.CancelOrders(req.cancel.orders, req.cancel.reason)
					req.cancel.reply <- cancelEnvelope{result: result, err: err}
				}
			}
		}
	}()
}

func (e *AsyncEngine) getOrCreateWorkerLocked(bookKey string) *bookWorker {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.getOrCreateWorkerUnsafe(bookKey)
}

func (e *AsyncEngine) getOrCreateWorkerUnsafe(bookKey string) *bookWorker {
	if w, ok := e.workers[bookKey]; ok {
		return w
	}
	eng := NewWithSequence(e.logger, &e.sequence)
	w := &bookWorker{
		bookKey: bookKey,
		engine:  eng,
		ch:      make(chan asyncRequest, e.buffer),
	}
	e.workers[bookKey] = w
	if e.ctx != nil {
		startWorker(e.ctx, w)
	}
	return w
}

func (e *AsyncEngine) Submit(ctx context.Context, order *model.Order) (Result, error) {
	if order == nil {
		return Result{}, fmt.Errorf("order is nil")
	}
	bookKey := order.BookKey()

	e.mu.Lock()
	w := e.getOrCreateWorkerUnsafe(bookKey)
	e.mu.Unlock()

	reply := make(chan resultEnvelope, 1)
	request := asyncRequest{place: &placeOrderRequest{order: order, reply: reply}}

	select {
	case <-ctx.Done():
		return Result{}, ctx.Err()
	case w.ch <- request:
	}

	select {
	case <-ctx.Done():
		return Result{}, ctx.Err()
	case resp := <-reply:
		return resp.result, resp.err
	}
}

func (e *AsyncEngine) CancelOrders(ctx context.Context, orders []*model.Order, reason model.CancelReason) (CancelResult, error) {
	grouped := make(map[string][]*model.Order)
	for _, o := range orders {
		if o == nil {
			continue
		}
		key := o.BookKey()
		grouped[key] = append(grouped[key], o)
	}
	if len(grouped) == 0 {
		return CancelResult{}, nil
	}

	type fanoutResult struct {
		result CancelResult
		err    error
	}

	e.mu.RLock()
	results := make(chan fanoutResult, len(grouped))
	sent := 0
	for bookKey, bookOrders := range grouped {
		w, ok := e.workers[bookKey]
		if !ok {
			continue
		}
		sent++
		reply := make(chan cancelEnvelope, 1)
		request := asyncRequest{cancel: &cancelOrdersRequest{orders: bookOrders, reason: reason, reply: reply}}

		go func(w *bookWorker, reply chan cancelEnvelope) {
			select {
			case <-ctx.Done():
				results <- fanoutResult{err: ctx.Err()}
			case w.ch <- request:
				select {
				case <-ctx.Done():
					results <- fanoutResult{err: ctx.Err()}
				case resp := <-reply:
					results <- fanoutResult{result: resp.result, err: resp.err}
				}
			}
		}(w, reply)
	}
	e.mu.RUnlock()

	merged := CancelResult{}
	for i := 0; i < sent; i++ {
		fr := <-results
		if fr.err != nil {
			return CancelResult{}, fr.err
		}
		merged.Orders = append(merged.Orders, fr.result.Orders...)
		merged.Books = append(merged.Books, fr.result.Books...)
	}
	return merged, nil
}

func (e *AsyncEngine) BookCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	count := 0
	for _, w := range e.workers {
		for _, book := range w.engine.books {
			if book.OrderCount() > 0 {
				count++
			}
		}
	}
	return count
}

func (e *Engine) RestoreOrder(order *model.Order) error {
	if order == nil {
		return nil
	}
	if err := order.Validate(); err != nil {
		return fmt.Errorf("restore order %s: %w", order.OrderID, err)
	}
	if order.RemainingQuantity() <= 0 {
		return nil
	}

	book := e.getOrCreateBook(order.BookKey())
	if book.HasOrder(order.OrderID) {
		return nil
	}

	restored := cloneOrder(order)
	if restored.Status == "" {
		if restored.FilledQuantity > 0 {
			restored.Status = model.OrderStatusPartiallyFilled
		} else {
			restored.Status = model.OrderStatusNew
		}
	}
	book.AddOrder(restored)
	return nil
}

func (e *Engine) PlaceOrder(order *model.Order) (Result, error) {
	if order == nil {
		return Result{}, fmt.Errorf("order is nil")
	}
	if err := order.Validate(); err != nil {
		order.Reject(model.CancelReasonValidationFailed)
		return Result{Order: order}, err
	}
	if order.Status == "" {
		order.Status = model.OrderStatusNew
	}

	book := e.getOrCreateBook(order.BookKey())
	if book.HasOrder(order.OrderID) {
		order.Reject(model.CancelReasonValidationFailed)
		return Result{Order: order}, fmt.Errorf("assert: duplicate order id: %s", order.OrderID)
	}

	result := Result{Order: order}
	switch order.Type {
	case model.OrderTypeLimit:
		result.Trades, result.Affected = e.processLimitOrder(order, book)
	default:
		order.Reject(model.CancelReasonValidationFailed)
		return Result{Order: order}, fmt.Errorf("assert: unsupported order type: %s (MARKET should be converted upstream to LIMIT IOC)", order.Type)
	}
	result.Book = book.Snapshot(5)

	return result, nil
}

func (e *Engine) CancelOrders(orders []*model.Order, reason model.CancelReason) (CancelResult, error) {
	if len(orders) == 0 {
		return CancelResult{}, nil
	}
	if reason == "" {
		return CancelResult{}, fmt.Errorf("assert: cancel reason is required")
	}

	nowMillis := time.Now().UnixMilli()
	cancelled := make([]*model.Order, 0, len(orders))
	bookKeys := make([]string, 0, len(orders))
	seenBooks := make(map[string]struct{}, len(orders))

	for _, candidate := range orders {
		if candidate == nil {
			continue
		}
		book, ok := e.books[candidate.BookKey()]
		if !ok {
			continue
		}
		existing, ok := book.GetDirectOrder(candidate.OrderID)
		if !ok || existing.RemainingQuantity() <= 0 {
			continue
		}

		existing.Cancel(reason)
		existing.UpdatedAtMillis = nowMillis
		result := existing.ToOrder()
		book.RemoveDirectOrder(existing)
		cancelled = append(cancelled, result)

		if _, already := seenBooks[book.Key]; !already {
			seenBooks[book.Key] = struct{}{}
			bookKeys = append(bookKeys, book.Key)
		}
	}

	snapshots := make([]model.BookSnapshot, 0, len(bookKeys))
	for _, key := range bookKeys {
		book, ok := e.books[key]
		if !ok {
			continue
		}
		snapshots = append(snapshots, book.Snapshot(5))
		if book.OrderCount() == 0 {
			delete(e.books, key)
		}
	}

	return CancelResult{
		Orders: cancelled,
		Books:  snapshots,
	}, nil
}

func (e *Engine) processLimitOrder(order *model.Order, book *model.OrderBookDirect) ([]model.Trade, []*model.Order) {
	switch order.TimeInForce {
	case model.TimeInForcePostOnly:
		return e.processPostOnly(order, book)
	case model.TimeInForceFOK:
		return e.processFOK(order, book)
	default:
		// GTC and IOC both attempt immediate matching.
		return e.processGTCOrIOC(order, book)
	}
}

// processPostOnly adds the order to the book only if it would NOT cross.
func (e *Engine) processPostOnly(order *model.Order, book *model.OrderBookDirect) ([]model.Trade, []*model.Order) {
	if book.IsCross(order) {
		order.Cancel(model.CancelReasonPostOnlyCross)
		return nil, nil
	}
	book.AddOrder(order)
	order.Status = model.OrderStatusNew
	return nil, nil
}

// processFOK implements Fill-or-Kill with a two-phase approach:
// Phase 1 — simulate matching (read-only) to check if the full quantity can fill.
// Phase 2 — if yes, execute the real match.
func (e *Engine) processFOK(order *model.Order, book *model.OrderBookDirect) ([]model.Trade, []*model.Order) {
	if !book.IsCross(order) {
		order.Cancel(model.CancelReasonFOKNotFilled)
		return nil, nil
	}
	if !e.fokCanFill(order, book) {
		order.Cancel(model.CancelReasonFOKNotFilled)
		return nil, nil
	}
	// Full fill guaranteed — execute real match.
	trades, affected := e.match(order, book)
	return trades, affected
}

// fokCanFill simulates matching without modifying any state, returning true
// only if the order's full quantity can be filled against the current book.
func (e *Engine) fokCanFill(order *model.Order, book *model.OrderBookDirect) bool {
	remaining := order.RemainingQuantity()

	type bucketIter struct {
		first func() *model.Bucket
		next  func(int64) *model.Bucket
		cross func(int64) bool
	}
	var it bucketIter
	if order.IsBuy() {
		it = bucketIter{
			first: book.FirstAskBucket,
			next:  book.NextAskBucket,
			cross: func(p int64) bool { return order.Price >= p },
		}
	} else {
		it = bucketIter{
			first: book.FirstBidBucket,
			next:  book.NextBidBucket,
			cross: func(p int64) bool { return order.Price <= p },
		}
	}

	for bucket := it.first(); bucket != nil && remaining > 0; bucket = it.next(bucket.Price) {
		if !it.cross(bucket.Price) {
			break
		}
		for maker := bucket.Head; maker != nil && remaining > 0; maker = maker.Next() {
			if e.stpWouldCancelTaker(order, maker) {
				return false
			}
			if e.stpWouldSkipMaker(order, maker) {
				continue
			}
			qty := min(remaining, maker.RemainingQuantity())
			if qty > 0 {
				remaining -= qty
			}
		}
	}
	return remaining == 0
}

func (e *Engine) processGTCOrIOC(order *model.Order, book *model.OrderBookDirect) ([]model.Trade, []*model.Order) {
	trades, affected := e.match(order, book)
	if order.RemainingQuantity() == 0 {
		return trades, affected
	}
	// STP may have cancelled the taker inside match() — don't rest or overwrite.
	if order.Status == model.OrderStatusCancelled {
		return trades, affected
	}

	if order.TimeInForce == model.TimeInForceIOC {
		if len(trades) == 0 {
			order.Cancel(model.CancelReasonIOCNoLiquidity)
		} else {
			order.Cancel(model.CancelReasonIOCPartialFill)
		}
		return trades, affected
	}

	// GTC: rest remainder on the book.
	book.AddOrder(order)
	if order.FilledQuantity > 0 {
		order.Status = model.OrderStatusPartiallyFilled
	} else {
		order.Status = model.OrderStatusNew
	}
	return trades, affected
}

// AmendOrder cancels an existing resting order and places a new one with
// updated price/quantity. The new order loses time priority (cancel + relist).
func (e *Engine) AmendOrder(original *model.Order, newPrice, newQty int64) (Result, error) {
	if original == nil {
		return Result{}, fmt.Errorf("original order is nil")
	}

	book, ok := e.books[original.BookKey()]
	if !ok {
		return Result{}, fmt.Errorf("book not found: %s", original.BookKey())
	}
	existing, ok := book.GetDirectOrder(original.OrderID)
	if !ok || existing.RemainingQuantity() <= 0 {
		return Result{}, fmt.Errorf("order not found or already filled: %s", original.OrderID)
	}

	// Cancel existing.
	existing.Cancel(model.CancelReasonAmended)
	existing.UpdatedAtMillis = time.Now().UnixMilli()
	cancelled := existing.ToOrder()
	book.RemoveDirectOrder(existing)

	// Build replacement order.
	amended := cloneOrder(cancelled)
	amended.Status = model.OrderStatusNew
	amended.CancelReason = ""
	amended.FilledQuantity = 0
	if newPrice > 0 {
		amended.Price = newPrice
	}
	if newQty > 0 {
		amended.Quantity = newQty
	}

	// Place the amended order through normal flow.
	result, err := e.PlaceOrder(amended)
	// Attach the cancelled original as the first affected order.
	result.Affected = append([]*model.Order{cancelled}, result.Affected...)
	return result, err
}

// ---- STP helpers ----

// stpWouldCancelTaker returns true if the taker's STP strategy means the
// taker itself should be cancelled upon encountering this maker.
func (e *Engine) stpWouldCancelTaker(taker *model.Order, maker *model.DirectOrder) bool {
	if taker.UserID != maker.UserID || taker.STPStrategy == model.STPNone {
		return false
	}
	return taker.STPStrategy == model.STPCancelTaker || taker.STPStrategy == model.STPCancelBoth
}

// stpWouldSkipMaker returns true if the taker's STP strategy means this
// maker should be skipped (and later cancelled).
func (e *Engine) stpWouldSkipMaker(taker *model.Order, maker *model.DirectOrder) bool {
	if taker.UserID != maker.UserID || taker.STPStrategy == model.STPNone {
		return false
	}
	return taker.STPStrategy == model.STPCancelMaker || taker.STPStrategy == model.STPCancelBoth
}

func (e *Engine) match(order *model.Order, book *model.OrderBookDirect) ([]model.Trade, []*model.Order) {
	if !book.IsCross(order) {
		return nil, nil
	}

	// Reuse pre-allocated buffers (reset length, keep backing array).
	trades := e.tradesBuf[:0]
	affected := e.affectedBuf[:0]
	toRemove := e.removeBuf[:0]

	bookKey := order.BookKey()
	stpStrategy := order.STPStrategy

	type bucketIter struct {
		first func() *model.Bucket
		next  func(int64) *model.Bucket
		cross func(int64) bool
	}
	var it bucketIter
	if order.IsBuy() {
		it = bucketIter{
			first: book.FirstAskBucket,
			next:  book.NextAskBucket,
			cross: func(p int64) bool { return order.Price >= p },
		}
	} else {
		it = bucketIter{
			first: book.FirstBidBucket,
			next:  book.NextBidBucket,
			cross: func(p int64) bool { return order.Price <= p },
		}
	}

	takerDone := false
	for bucket := it.first(); bucket != nil && order.RemainingQuantity() > 0 && !takerDone; bucket = it.next(bucket.Price) {
		if !it.cross(bucket.Price) {
			break
		}
		for maker := bucket.Head; maker != nil && order.RemainingQuantity() > 0 && !takerDone; {
			nextMaker := maker.Next()

			// Self-trade prevention.
			if order.UserID == maker.UserID && stpStrategy != model.STPNone {
				switch stpStrategy {
				case model.STPCancelTaker:
					order.Cancel(model.CancelReasonSTPTaker)
					takerDone = true
				case model.STPCancelMaker:
					maker.Cancel(model.CancelReasonSTPMaker)
					affected = append(affected, maker.ToOrder())
					toRemove = append(toRemove, maker)
				case model.STPCancelBoth:
					order.Cancel(model.CancelReasonSTPBoth)
					maker.Cancel(model.CancelReasonSTPBoth)
					affected = append(affected, maker.ToOrder())
					toRemove = append(toRemove, maker)
					takerDone = true
				}
				maker = nextMaker
				continue
			}
			// Legacy: skip same-user when no STP strategy set (backward compat).
			if order.UserID == maker.UserID {
				maker = nextMaker
				continue
			}

			tradeQty := min(order.RemainingQuantity(), maker.RemainingQuantity())
			if tradeQty <= 0 {
				maker = nextMaker
				continue
			}
			order.ApplyFill(tradeQty)
			maker.ApplyFill(tradeQty)
			e.localSeq++
			seq := atomic.AddUint64(e.sequence, 1)
			trades = append(trades, model.Trade{
				Sequence:        seq,
				TradeID:         model.DeterministicTradeID(bookKey, e.localSeq),
				MarketID:        order.MarketID,
				Outcome:         order.Outcome,
				BookKey:         bookKey,
				Price:           maker.Price,
				Quantity:        tradeQty,
				TakerOrderID:    order.OrderID,
				MakerOrderID:    maker.OrderID,
				TakerUserID:     order.UserID,
				MakerUserID:     maker.UserID,
				TakerSide:       order.Side,
				MakerSide:       maker.Side,
				MatchedAtMillis: time.Now().UnixMilli(),
			})
			affected = append(affected, maker.ToOrder())
			if maker.RemainingQuantity() == 0 {
				toRemove = append(toRemove, maker)
			}
			maker = nextMaker
		}
	}

	for _, do := range toRemove {
		book.RemoveDirectOrder(do)
	}

	// Save buffers back for next reuse.
	e.tradesBuf = trades
	e.affectedBuf = affected
	e.removeBuf = toRemove

	// Detach returned slices from the reusable backing arrays.
	out := make([]model.Trade, len(trades))
	copy(out, trades)
	outAff := make([]*model.Order, len(affected))
	copy(outAff, affected)
	return out, outAff
}

// ExportBooks returns the internal book map for snapshot export.
// Only safe to call from the owning goroutine (single-writer guarantee).
func (e *Engine) ExportBooks() map[string]*model.OrderBookDirect {
	return e.books
}

func (e *Engine) getOrCreateBook(key string) *model.OrderBookDirect {
	book, ok := e.books[key]
	if ok {
		return book
	}
	book = model.NewOrderBookDirect(key)
	e.books[key] = book
	return book
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func cloneOrder(order *model.Order) *model.Order {
	if order == nil {
		return nil
	}
	cloned := *order
	return &cloned
}
