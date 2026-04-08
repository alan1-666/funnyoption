package engine

import (
	"fmt"
	"io"
	"log/slog"
	"testing"

	"funnyoption/internal/matching/model"
)

var benchLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func mkOrder(id string, uid, market int64, outcome string, side model.OrderSide, price, qty int64) *model.Order {
	return &model.Order{
		OrderID:     id,
		UserID:      uid,
		MarketID:    market,
		Outcome:     outcome,
		Side:        side,
		Type:        model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC,
		Price:       price,
		Quantity:    qty,
	}
}

// Each order placed into its own fresh book → measures pure insert path.
func BenchmarkPlaceOrder_EmptyBook(b *testing.B) {
	eng := New(benchLogger)
	orders := make([]*model.Order, b.N)
	for i := range orders {
		orders[i] = mkOrder(fmt.Sprintf("o-%d", i), 1, int64(i+1), "YES", model.OrderSideBuy, 50, 10)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eng.PlaceOrder(orders[i])
	}
}

// Single book with 1000 resting asks (qty=1B so it never drains).
// Each taker matches 1 lot against the best ask.
func BenchmarkPlaceOrder_DeepBook(b *testing.B) {
	eng := New(benchLogger)
	for j := 0; j < 1000; j++ {
		eng.PlaceOrder(mkOrder(fmt.Sprintf("m-%d", j), int64(j%500)+1, 1, "YES", model.OrderSideSell, int64(51+j%49), 1_000_000_000))
	}
	orders := make([]*model.Order, b.N)
	for i := range orders {
		orders[i] = mkOrder(fmt.Sprintf("t-%d", i), 999, 1, "YES", model.OrderSideBuy, 55, 1)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eng.PlaceOrder(orders[i])
	}
}

// 50 ask levels × 10 orders. Taker crosses entire spread matching 1 lot at best ask.
func BenchmarkMatch_CrossSpread(b *testing.B) {
	eng := New(benchLogger)
	for p := int64(1); p <= 50; p++ {
		for j := 0; j < 10; j++ {
			eng.PlaceOrder(mkOrder(fmt.Sprintf("a-%d-%d", p, j), int64(j+1), 1, "YES", model.OrderSideSell, p, 1_000_000_000))
		}
	}
	orders := make([]*model.Order, b.N)
	for i := range orders {
		orders[i] = mkOrder(fmt.Sprintf("c-%d", i), 999, 1, "YES", model.OrderSideBuy, 50, 1)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eng.PlaceOrder(orders[i])
	}
}

// DeterministicTradeID generation overhead.
func BenchmarkDeterministicTradeID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = model.DeterministicTradeID("1:YES", uint64(i))
	}
}

// Match with epoch/tradeID (Phase 5). Same as CrossSpread — measures
// additional overhead from localSeq increment + DeterministicTradeID.
func BenchmarkMatch_CrossSpread_WithEpoch(b *testing.B) {
	eng := New(benchLogger)
	for p := int64(1); p <= 50; p++ {
		for j := 0; j < 10; j++ {
			eng.PlaceOrder(mkOrder(fmt.Sprintf("a-%d-%d", p, j), int64(j+1), 1, "YES", model.OrderSideSell, p, 1_000_000_000))
		}
	}
	orders := make([]*model.Order, b.N)
	for i := range orders {
		orders[i] = mkOrder(fmt.Sprintf("c-%d", i), 999, 1, "YES", model.OrderSideBuy, 50, 1)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eng.PlaceOrder(orders[i])
	}
}

// Pure AddOrder into a book (no matching). Each call inserts into a fresh book.
func BenchmarkAddOrder_Fresh(b *testing.B) {
	orders := make([]*model.Order, b.N)
	for i := range orders {
		orders[i] = mkOrder(fmt.Sprintf("n-%d", i), 1, int64(i+1), "YES", model.OrderSideBuy, 50, 10)
	}
	books := make([]*model.OrderBook, b.N)
	for i := range books {
		books[i] = model.NewOrderBook(fmt.Sprintf("%d:YES", i+1))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		books[i].AddOrder(orders[i])
	}
}
