package service

import (
	"context"
	"encoding/json"
	"testing"

	sharedkafka "funnyoption/internal/shared/kafka"
)

type stubOrderStateStore struct {
	states  map[string]*OrderState
	mirrors map[string]OrderState
}

type stubFreezeReader struct {
	frozen bool
}

func (s stubFreezeReader) RollupFrozen(_ context.Context) (bool, error) {
	return s.frozen, nil
}

func (s *stubOrderStateStore) LoadOrderState(_ context.Context, orderID string) (*OrderState, error) {
	state, ok := s.states[orderID]
	if !ok {
		return nil, nil
	}
	cloned := *state
	return &cloned, nil
}

func (s *stubOrderStateStore) MirrorOrderState(_ context.Context, state OrderState) error {
	if s.mirrors == nil {
		s.mirrors = make(map[string]OrderState)
	}
	s.mirrors[state.OrderID] = state
	return nil
}

func TestEventProcessorOrderLifecycle(t *testing.T) {
	book := NewBalanceBook()
	book.SeedBalance(1001, "USDT", 10_000)

	processor := NewEventProcessor(book, NewOrderRegistry())

	orderEventPayload, err := json.Marshal(sharedkafka.OrderEvent{
		OrderID:           "ord_1",
		UserID:            1001,
		Side:              "BUY",
		Price:             60,
		FreezeID:          "frz_1",
		FreezeAsset:       "USDT",
		FreezeAmount:      3_000,
		Status:            "NEW",
		RemainingQuantity: 30,
	})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	if err := processor.HandleOrderEvent(context.Background(), sharedkafka.Message{Value: orderEventPayload}); err != nil {
		t.Fatalf("HandleOrderEvent returned error: %v", err)
	}

	balance := book.GetBalance(1001, "USDT")
	if balance.Available != 7_000 || balance.Frozen != 3_000 {
		t.Fatalf("unexpected balance after freeze: %+v", balance)
	}

	tradePayload, err := json.Marshal(sharedkafka.TradeMatchedEvent{
		EventID:         "evt_trade_1",
		CollateralAsset: "USDT",
		Price:           50,
		Quantity:        20,
		TakerOrderID:    "ord_1",
		MakerOrderID:    "ord_2",
		TakerUserID:     1001,
		MakerUserID:     1002,
		TakerSide:       "BUY",
		MakerSide:       "SELL",
	})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	if err := processor.HandleTradeMatched(context.Background(), sharedkafka.Message{Value: tradePayload}); err != nil {
		t.Fatalf("HandleTradeMatched returned error: %v", err)
	}

	balance = book.GetBalance(1001, "USDT")
	if balance.Available != 7_200 || balance.Frozen != 1_800 {
		t.Fatalf("unexpected balance after consume: %+v", balance)
	}

	finalOrderPayload, err := json.Marshal(sharedkafka.OrderEvent{
		OrderID:           "ord_1",
		UserID:            1001,
		Side:              "BUY",
		Price:             60,
		Status:            "CANCELLED",
		RemainingQuantity: 30,
	})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	if err := processor.HandleOrderEvent(context.Background(), sharedkafka.Message{Value: finalOrderPayload}); err != nil {
		t.Fatalf("HandleOrderEvent returned error: %v", err)
	}

	balance = book.GetBalance(1001, "USDT")
	if balance.Available != 9_000 || balance.Frozen != 0 {
		t.Fatalf("unexpected balance after release: %+v", balance)
	}
}

func TestEventProcessorFilledOrderKeepsConsumedFreeze(t *testing.T) {
	book := NewBalanceBook()
	book.SeedBalance(1001, "USDT", 10_000)

	processor := NewEventProcessor(book, NewOrderRegistry())

	newOrderPayload, err := json.Marshal(sharedkafka.OrderEvent{
		OrderID:           "ord_fill_1",
		UserID:            1001,
		Side:              "BUY",
		FreezeID:          "frz_fill_1",
		FreezeAsset:       "USDT",
		FreezeAmount:      5_800,
		Status:            "NEW",
		RemainingQuantity: 100,
	})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	if err := processor.HandleOrderEvent(context.Background(), sharedkafka.Message{Value: newOrderPayload}); err != nil {
		t.Fatalf("HandleOrderEvent returned error: %v", err)
	}

	filledOrderPayload, err := json.Marshal(sharedkafka.OrderEvent{
		OrderID:           "ord_fill_1",
		UserID:            1001,
		Side:              "BUY",
		Status:            "FILLED",
		RemainingQuantity: 0,
	})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	if err := processor.HandleOrderEvent(context.Background(), sharedkafka.Message{Value: filledOrderPayload}); err != nil {
		t.Fatalf("HandleOrderEvent returned error: %v", err)
	}

	tradePayload, err := json.Marshal(sharedkafka.TradeMatchedEvent{
		EventID:         "evt_trade_fill_1",
		CollateralAsset: "USDT",
		Price:           58,
		Quantity:        100,
		TakerOrderID:    "ord_fill_1",
		MakerOrderID:    "ord_fill_2",
		TakerUserID:     1001,
		MakerUserID:     1002,
		TakerSide:       "BUY",
		MakerSide:       "SELL",
		MarketID:        1001,
		Outcome:         "YES",
	})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	if err := processor.HandleTradeMatched(context.Background(), sharedkafka.Message{Value: tradePayload}); err != nil {
		t.Fatalf("HandleTradeMatched returned error: %v", err)
	}

	balance := book.GetBalance(1001, "USDT")
	if balance.Available != 4_200 || balance.Frozen != 0 {
		t.Fatalf("unexpected balance after filled trade settlement: %+v", balance)
	}

	if amount, ok := book.FreezeAmount("frz_fill_1"); !ok || amount != 0 {
		t.Fatalf("expected consumed freeze, got ok=%v amount=%d", ok, amount)
	}
}

func TestEventProcessorFilledBuyReleasesPriceImprovement(t *testing.T) {
	book := NewBalanceBook()
	book.SeedBalance(1001, "USDT", 10_000)

	processor := NewEventProcessor(book, NewOrderRegistry())

	newOrderPayload, err := json.Marshal(sharedkafka.OrderEvent{
		OrderID:           "ord_improve_1",
		UserID:            1001,
		Side:              "BUY",
		Price:             61,
		FreezeID:          "frz_improve_1",
		FreezeAsset:       "USDT",
		FreezeAmount:      4_270,
		Status:            "NEW",
		RemainingQuantity: 70,
	})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}
	if err := processor.HandleOrderEvent(context.Background(), sharedkafka.Message{Value: newOrderPayload}); err != nil {
		t.Fatalf("HandleOrderEvent returned error: %v", err)
	}

	filledOrderPayload, err := json.Marshal(sharedkafka.OrderEvent{
		OrderID:           "ord_improve_1",
		UserID:            1001,
		Side:              "BUY",
		Price:             61,
		Status:            "FILLED",
		RemainingQuantity: 0,
	})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}
	if err := processor.HandleOrderEvent(context.Background(), sharedkafka.Message{Value: filledOrderPayload}); err != nil {
		t.Fatalf("HandleOrderEvent returned error: %v", err)
	}

	tradePayload, err := json.Marshal(sharedkafka.TradeMatchedEvent{
		EventID:         "evt_trade_improve_1",
		CollateralAsset: "USDT",
		Price:           58,
		Quantity:        70,
		TakerOrderID:    "ord_improve_1",
		MakerOrderID:    "ord_improve_2",
		TakerUserID:     1001,
		MakerUserID:     1002,
		TakerSide:       "BUY",
		MakerSide:       "SELL",
		MarketID:        1001,
		Outcome:         "YES",
	})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}
	if err := processor.HandleTradeMatched(context.Background(), sharedkafka.Message{Value: tradePayload}); err != nil {
		t.Fatalf("HandleTradeMatched returned error: %v", err)
	}

	balance := book.GetBalance(1001, "USDT")
	if balance.Available != 5_940 || balance.Frozen != 0 {
		t.Fatalf("unexpected balance after releasing price improvement: %+v", balance)
	}
	if amount, ok := book.FreezeAmount("frz_improve_1"); !ok || amount != 0 {
		t.Fatalf("expected released freeze to report zero outstanding, got ok=%v amount=%d", ok, amount)
	}
}

func TestEventProcessorPassiveOrderUpdateKeepsRestoredFreeze(t *testing.T) {
	book := NewBalanceBook()
	book.SeedBalance(1003, "POSITION:1101:YES", 120)
	if _, err := book.PreFreeze(FreezeRequest{
		FreezeID: "frz_sell_restore",
		UserID:   1003,
		Asset:    "POSITION:1101:YES",
		RefType:  "ORDER",
		RefID:    "ord_sell_restore",
		Amount:   120,
	}); err != nil {
		t.Fatalf("PreFreeze returned error: %v", err)
	}

	store := &stubOrderStateStore{
		states: map[string]*OrderState{
			"ord_sell_restore": {
				OrderID:           "ord_sell_restore",
				UserID:            1003,
				Side:              "SELL",
				FreezeID:          "frz_sell_restore",
				FreezeAsset:       "POSITION:1101:YES",
				FreezeApplied:     true,
				Status:            "NEW",
				RemainingQuantity: 120,
			},
		},
	}
	processor := NewEventProcessor(book, NewPersistentOrderRegistry(store))

	passiveOrderPayload, err := json.Marshal(sharedkafka.OrderEvent{
		OrderID:           "ord_sell_restore",
		UserID:            1003,
		Side:              "SELL",
		CollateralAsset:   "USDT",
		Status:            "PARTIALLY_FILLED",
		RemainingQuantity: 110,
	})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}
	if err := processor.HandleOrderEvent(context.Background(), sharedkafka.Message{Value: passiveOrderPayload}); err != nil {
		t.Fatalf("HandleOrderEvent returned error: %v", err)
	}

	tradePayload, err := json.Marshal(sharedkafka.TradeMatchedEvent{
		EventID:         "evt_trade_restore_1",
		CollateralAsset: "USDT",
		MarketID:        1101,
		Outcome:         "YES",
		Price:           61,
		Quantity:        10,
		TakerOrderID:    "ord_buy_restore",
		MakerOrderID:    "ord_sell_restore",
		TakerUserID:     1001,
		MakerUserID:     1003,
		TakerSide:       "BUY",
		MakerSide:       "SELL",
	})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}
	if err := processor.HandleTradeMatched(context.Background(), sharedkafka.Message{Value: tradePayload}); err != nil {
		t.Fatalf("HandleTradeMatched returned error: %v", err)
	}

	balance := book.GetBalance(1003, "POSITION:1101:YES")
	if balance.Available != 0 || balance.Frozen != 110 {
		t.Fatalf("unexpected seller position balance after restored trade consume: %+v", balance)
	}
	if amount, ok := book.FreezeAmount("frz_sell_restore"); !ok || amount != 110 {
		t.Fatalf("expected active freeze with remaining 110, got ok=%v amount=%d", ok, amount)
	}
	if mirrored := store.mirrors["ord_sell_restore"]; mirrored.FreezeID != "frz_sell_restore" || mirrored.FreezeAsset != "POSITION:1101:YES" {
		t.Fatalf("expected restored freeze metadata to survive passive update, got %+v", mirrored)
	}
}

func TestEventProcessorSettlementCreditsFullWinningPayout(t *testing.T) {
	book := NewBalanceBook()
	book.SeedBalance(1001, "POSITION:1775124927529:YES", 10)

	processor := NewEventProcessor(book, NewOrderRegistry())

	settlementPayload, err := json.Marshal(sharedkafka.SettlementCompletedEvent{
		EventID:         "evt_settlement_1775124927529_1001_YES",
		MarketID:        1775124927529,
		UserID:          1001,
		WinningOutcome:  "YES",
		PositionAsset:   "POSITION:1775124927529:YES",
		SettledQuantity: 10,
		PayoutAsset:     "USDT",
		PayoutAmount:    1000,
	})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	if err := processor.HandleSettlementCompleted(context.Background(), sharedkafka.Message{Value: settlementPayload}); err != nil {
		t.Fatalf("HandleSettlementCompleted returned error: %v", err)
	}

	position := book.GetBalance(1001, "POSITION:1775124927529:YES")
	if position.Available != 0 || position.Frozen != 0 {
		t.Fatalf("unexpected settled position balance: %+v", position)
	}

	usdt := book.GetBalance(1001, "USDT")
	if usdt.Available != 1000 || usdt.Frozen != 0 {
		t.Fatalf("unexpected settlement payout balance: %+v", usdt)
	}
}

func TestEventProcessorSkipsOrderLifecycleWhileFrozen(t *testing.T) {
	book := NewBalanceBook()
	book.SeedBalance(1001, "USDT", 10_000)

	processor := NewEventProcessor(book, NewOrderRegistry(), stubFreezeReader{frozen: true})

	orderEventPayload, err := json.Marshal(sharedkafka.OrderEvent{
		OrderID:           "ord_frozen_1",
		UserID:            1001,
		Side:              "BUY",
		Price:             60,
		FreezeID:          "frz_frozen_1",
		FreezeAsset:       "USDT",
		FreezeAmount:      3_000,
		Status:            "NEW",
		RemainingQuantity: 30,
	})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	if err := processor.HandleOrderEvent(context.Background(), sharedkafka.Message{Value: orderEventPayload}); err != nil {
		t.Fatalf("HandleOrderEvent returned error: %v", err)
	}

	balance := book.GetBalance(1001, "USDT")
	if balance.Available != 10_000 || balance.Frozen != 0 {
		t.Fatalf("expected frozen account lane to skip order mutation, got %+v", balance)
	}
}

func TestEventProcessorSkipsSettlementWhileFrozen(t *testing.T) {
	book := NewBalanceBook()
	book.SeedBalance(1001, "POSITION:1775124927529:YES", 10)

	processor := NewEventProcessor(book, NewOrderRegistry(), stubFreezeReader{frozen: true})

	settlementPayload, err := json.Marshal(sharedkafka.SettlementCompletedEvent{
		EventID:         "evt_settlement_frozen",
		MarketID:        1775124927529,
		UserID:          1001,
		WinningOutcome:  "YES",
		PositionAsset:   "POSITION:1775124927529:YES",
		SettledQuantity: 10,
		PayoutAsset:     "USDT",
		PayoutAmount:    1000,
	})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	if err := processor.HandleSettlementCompleted(context.Background(), sharedkafka.Message{Value: settlementPayload}); err != nil {
		t.Fatalf("HandleSettlementCompleted returned error: %v", err)
	}

	position := book.GetBalance(1001, "POSITION:1775124927529:YES")
	if position.Available != 10 || position.Frozen != 0 {
		t.Fatalf("expected frozen account lane to skip settlement position mutation, got %+v", position)
	}

	usdt := book.GetBalance(1001, "USDT")
	if usdt.Available != 0 || usdt.Frozen != 0 {
		t.Fatalf("expected frozen account lane to skip payout mutation, got %+v", usdt)
	}
}
