package service

import "testing"

func TestBalanceBookPreFreezeAndRelease(t *testing.T) {
	book := NewBalanceBook()
	book.SeedBalance(1001, "USDT", 10_000)

	record, err := book.PreFreeze(FreezeRequest{
		UserID:  1001,
		Asset:   "USDT",
		RefType: "ORDER",
		RefID:   "ord_1",
		Amount:  3_000,
	})
	if err != nil {
		t.Fatalf("PreFreeze returned error: %v", err)
	}

	balance := book.GetBalance(1001, "USDT")
	if balance.Available != 7_000 || balance.Frozen != 3_000 {
		t.Fatalf("unexpected balance after pre-freeze: %+v", balance)
	}

	if err := book.ReleaseFreeze(record.FreezeID); err != nil {
		t.Fatalf("ReleaseFreeze returned error: %v", err)
	}

	balance = book.GetBalance(1001, "USDT")
	if balance.Available != 10_000 || balance.Frozen != 0 {
		t.Fatalf("unexpected balance after release: %+v", balance)
	}
	if amount, ok := book.FreezeAmount(record.FreezeID); !ok || amount != 0 {
		t.Fatalf("expected released freeze amount 0, got ok=%v amount=%d", ok, amount)
	}
}

func TestBalanceBookConsumeFreeze(t *testing.T) {
	book := NewBalanceBook()
	book.SeedBalance(1002, "USDT", 20_000)

	record, err := book.PreFreeze(FreezeRequest{
		UserID:  1002,
		Asset:   "USDT",
		RefType: "ORDER",
		RefID:   "ord_2",
		Amount:  8_000,
	})
	if err != nil {
		t.Fatalf("PreFreeze returned error: %v", err)
	}

	if err := book.ConsumeFreeze(record.FreezeID, 5_000); err != nil {
		t.Fatalf("ConsumeFreeze returned error: %v", err)
	}

	balance := book.GetBalance(1002, "USDT")
	if balance.Available != 12_000 || balance.Frozen != 3_000 {
		t.Fatalf("unexpected balance after consume: %+v", balance)
	}

	if outstanding := book.OutstandingFrozen("USDT"); outstanding != 3_000 {
		t.Fatalf("unexpected outstanding frozen amount: %d", outstanding)
	}
}

func TestBalanceBookCreditAvailableWithRef(t *testing.T) {
	book := NewBalanceBook()

	balance, applied, err := book.CreditAvailableWithRef(CreditRequest{
		UserID:  1001,
		Asset:   "USDT",
		Amount:  500,
		RefType: "DEPOSIT",
		RefID:   "dep_1",
	})
	if err != nil {
		t.Fatalf("CreditAvailableWithRef returned error: %v", err)
	}
	if !applied {
		t.Fatalf("expected applied=true")
	}
	if balance.Available != 500 {
		t.Fatalf("unexpected balance: %+v", balance)
	}
}

func TestBalanceBookDebitAvailableWithRef(t *testing.T) {
	book := NewBalanceBook()
	book.SeedBalance(1001, "USDT", 2_000)

	balance, applied, err := book.DebitAvailableWithRef(DebitRequest{
		UserID:  1001,
		Asset:   "USDT",
		Amount:  750,
		RefType: "WITHDRAWAL",
		RefID:   "wdq_1",
	})
	if err != nil {
		t.Fatalf("DebitAvailableWithRef returned error: %v", err)
	}
	if !applied {
		t.Fatalf("expected applied=true")
	}
	if balance.Available != 1_250 {
		t.Fatalf("unexpected balance: %+v", balance)
	}
}
