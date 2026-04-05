package engine

import (
	"context"
	"fmt"
	"log/slog"
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
	books    map[string]*model.OrderBook
	sequence uint64
}

type AsyncEngine struct {
	engine   *Engine
	requests chan asyncRequest
}

func New(logger *slog.Logger) *Engine {
	return &Engine{
		logger: logger,
		books:  make(map[string]*model.OrderBook),
	}
}

func NewAsync(logger *slog.Logger, buffer int) *AsyncEngine {
	if buffer <= 0 {
		buffer = 1024
	}
	return &AsyncEngine{
		engine:   New(logger),
		requests: make(chan asyncRequest, buffer),
	}
}

func (e *AsyncEngine) Restore(sequence uint64, orders []*model.Order) error {
	atomic.StoreUint64(&e.engine.sequence, sequence)
	for _, order := range orders {
		if err := e.engine.RestoreOrder(order); err != nil {
			return err
		}
	}
	return nil
}

func (e *AsyncEngine) Start(ctx context.Context) {
	go func() {
		e.engine.logger.Info("matching event loop started", "buffer", cap(e.requests))
		for {
			select {
			case <-ctx.Done():
				e.engine.logger.Info("matching event loop stopped")
				return
			case req := <-e.requests:
				switch {
				case req.place != nil:
					result, err := e.engine.PlaceOrder(req.place.order)
					req.place.reply <- resultEnvelope{result: result, err: err}
				case req.cancel != nil:
					result, err := e.engine.CancelOrders(req.cancel.orders, req.cancel.reason)
					req.cancel.reply <- cancelEnvelope{result: result, err: err}
				}
			}
		}
	}()
}

func (e *AsyncEngine) Submit(ctx context.Context, order *model.Order) (Result, error) {
	reply := make(chan resultEnvelope, 1)
	request := asyncRequest{place: &placeOrderRequest{order: order, reply: reply}}

	select {
	case <-ctx.Done():
		return Result{}, ctx.Err()
	case e.requests <- request:
	}

	select {
	case <-ctx.Done():
		return Result{}, ctx.Err()
	case resp := <-reply:
		return resp.result, resp.err
	}
}

func (e *AsyncEngine) CancelOrders(ctx context.Context, orders []*model.Order, reason model.CancelReason) (CancelResult, error) {
	reply := make(chan cancelEnvelope, 1)
	request := asyncRequest{cancel: &cancelOrdersRequest{orders: orders, reason: reason, reply: reply}}

	select {
	case <-ctx.Done():
		return CancelResult{}, ctx.Err()
	case e.requests <- request:
	}

	select {
	case <-ctx.Done():
		return CancelResult{}, ctx.Err()
	case resp := <-reply:
		return resp.result, resp.err
	}
}

func (e *AsyncEngine) BookCount() int {
	return len(e.engine.books)
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
		return Result{Order: order}, fmt.Errorf("duplicate order id: %s", order.OrderID)
	}

	result := Result{Order: order}
	switch order.Type {
	case model.OrderTypeLimit:
		result.Trades, result.Affected = e.processLimitOrder(order, book)
	case model.OrderTypeMarket:
		result.Trades, result.Affected = e.processMarketOrder(order, book)
	default:
		order.Reject(model.CancelReasonValidationFailed)
		return Result{Order: order}, fmt.Errorf("unsupported order type: %s", order.Type)
	}
	result.Book = book.Snapshot(5)

	return result, nil
}

func (e *Engine) CancelOrders(orders []*model.Order, reason model.CancelReason) (CancelResult, error) {
	if len(orders) == 0 {
		return CancelResult{}, nil
	}
	if reason == "" {
		reason = model.CancelReasonMarketClosed
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
		existing, ok := book.OrderMap[candidate.OrderID]
		if !ok || existing.RemainingQuantity() <= 0 {
			continue
		}

		existing.Cancel(reason)
		existing.UpdatedAtMillis = nowMillis
		book.RemoveOrder(existing)
		cancelled = append(cancelled, cloneOrder(existing))

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
		if len(book.OrderMap) == 0 {
			delete(e.books, key)
		}
	}

	return CancelResult{
		Orders: cancelled,
		Books:  snapshots,
	}, nil
}

func (e *Engine) processLimitOrder(order *model.Order, book *model.OrderBook) ([]model.Trade, []*model.Order) {
	trades, affected := e.match(order, book)
	if order.RemainingQuantity() == 0 {
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

	book.AddOrder(order)
	if order.FilledQuantity > 0 {
		order.Status = model.OrderStatusPartiallyFilled
	} else {
		order.Status = model.OrderStatusNew
	}
	return trades, affected
}

func (e *Engine) processMarketOrder(order *model.Order, book *model.OrderBook) ([]model.Trade, []*model.Order) {
	trades, affected := e.match(order, book)
	if order.RemainingQuantity() > 0 {
		order.Cancel(model.CancelReasonMarketNoLiquidity)
	}
	return trades, affected
}

func (e *Engine) match(order *model.Order, book *model.OrderBook) ([]model.Trade, []*model.Order) {
	opposite := book.OppositeLevels(order)
	if len(opposite) == 0 {
		return nil, nil
	}

	trades := make([]model.Trade, 0)
	affected := make([]*model.Order, 0)
	for i := 0; i < len(opposite); i++ {
		level := opposite[i]
		if !book.IsCrossWithPrice(order, level.Price) {
			break
		}

		remaining := level.Orders[:0]
		for _, maker := range level.Orders {
			if order.RemainingQuantity() == 0 {
				remaining = append(remaining, maker)
				continue
			}
			tradeQty := min(order.RemainingQuantity(), maker.RemainingQuantity())
			if tradeQty <= 0 {
				remaining = append(remaining, maker)
				continue
			}

			order.ApplyFill(tradeQty)
			maker.ApplyFill(tradeQty)
			trade := model.Trade{
				Sequence:        atomic.AddUint64(&e.sequence, 1),
				MarketID:        order.MarketID,
				Outcome:         order.Outcome,
				BookKey:         order.BookKey(),
				Price:           maker.Price,
				Quantity:        tradeQty,
				TakerOrderID:    order.OrderID,
				MakerOrderID:    maker.OrderID,
				TakerUserID:     order.UserID,
				MakerUserID:     maker.UserID,
				TakerSide:       order.Side,
				MakerSide:       maker.Side,
				MatchedAtMillis: time.Now().UnixMilli(),
			}
			trades = append(trades, trade)

			if maker.RemainingQuantity() > 0 {
				remaining = append(remaining, maker)
			} else {
				book.RemoveFromMap(maker.OrderID)
			}
			affected = append(affected, cloneOrder(maker))
		}
		level.Orders = remaining
	}

	book.SetOppositeLevels(order, clearEmptyLevels(book.OppositeLevels(order)))
	return trades, affected
}

func (e *Engine) getOrCreateBook(key string) *model.OrderBook {
	book, ok := e.books[key]
	if ok {
		return book
	}
	book = model.NewOrderBook(key)
	e.books[key] = book
	return book
}

func clearEmptyLevels(levels []*model.DepthLevel) []*model.DepthLevel {
	result := levels[:0]
	for _, level := range levels {
		if !level.IsEmpty() {
			result = append(result, level)
		}
	}
	return result
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
