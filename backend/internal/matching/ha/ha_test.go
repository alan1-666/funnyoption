package ha

import (
	"funnyoption/internal/matching/model"
	"testing"
)

func TestEpochManagerAdvance(t *testing.T) {
	em := NewEpochManager(0)
	if got := em.Current(); got != 0 {
		t.Fatalf("expected epoch 0, got %d", got)
	}

	e1 := em.Advance()
	if e1 != 1 {
		t.Fatalf("expected epoch 1 after first advance, got %d", e1)
	}
	e2 := em.Advance()
	if e2 != 2 {
		t.Fatalf("expected epoch 2, got %d", e2)
	}
	if got := em.Current(); got != 2 {
		t.Fatalf("expected current epoch 2, got %d", got)
	}
}

func TestEpochManagerSet(t *testing.T) {
	em := NewEpochManager(0)
	em.Set(42)
	if got := em.Current(); got != 42 {
		t.Fatalf("expected epoch 42, got %d", got)
	}
}

func TestRoleManagerTransition(t *testing.T) {
	rm := NewRoleManager(RolePrimary)
	if !rm.IsPrimary() {
		t.Fatal("expected PRIMARY")
	}

	transitionCount := 0
	rm.OnTransition(func(old, newRole Role) {
		transitionCount++
		if old != RolePrimary || newRole != RoleStandby {
			t.Fatalf("unexpected transition: %s -> %s", old, newRole)
		}
	})

	// Same role should not fire listener
	rm.Transition(RolePrimary)
	if transitionCount != 0 {
		t.Fatal("listener should not fire on same-role transition")
	}

	rm.Transition(RoleStandby)
	if transitionCount != 1 {
		t.Fatalf("expected 1 transition, got %d", transitionCount)
	}
	if rm.IsPrimary() {
		t.Fatal("expected STANDBY")
	}
}

func TestRoleString(t *testing.T) {
	if RolePrimary.String() != "PRIMARY" {
		t.Fatal("unexpected string for PRIMARY")
	}
	if RoleStandby.String() != "STANDBY" {
		t.Fatal("unexpected string for STANDBY")
	}
}

func TestDeterministicTradeID(t *testing.T) {
	id1 := model.DeterministicTradeID("1:YES", 1)
	id2 := model.DeterministicTradeID("1:YES", 1)
	if id1 != id2 {
		t.Fatalf("expected deterministic IDs to be equal: %s != %s", id1, id2)
	}

	id3 := model.DeterministicTradeID("1:YES", 2)
	if id1 == id3 {
		t.Fatal("different sequences should produce different IDs")
	}

	id4 := model.DeterministicTradeID("2:NO", 1)
	if id1 == id4 {
		t.Fatal("different book keys should produce different IDs")
	}

	expected := "1:YES:00000001"
	if id1 != expected {
		t.Fatalf("expected %s, got %s", expected, id1)
	}
}

func TestComputeDepthDiff(t *testing.T) {
	old := model.BookSnapshot{
		Key: "1:YES",
		Bids: []model.BookLevel{
			{Price: 50, Quantity: 100},
			{Price: 49, Quantity: 200},
		},
		Asks: []model.BookLevel{
			{Price: 51, Quantity: 150},
		},
		BestBid: 50,
		BestAsk: 51,
	}

	newSnap := model.BookSnapshot{
		Key: "1:YES",
		Bids: []model.BookLevel{
			{Price: 50, Quantity: 80},
			{Price: 48, Quantity: 50},
		},
		Asks: []model.BookLevel{
			{Price: 51, Quantity: 150},
			{Price: 52, Quantity: 75},
		},
		BestBid: 50,
		BestAsk: 51,
	}

	diff := ComputeDepthDiff(old, newSnap, 42)

	if diff.BookKey != "1:YES" {
		t.Fatalf("unexpected book key: %s", diff.BookKey)
	}
	if diff.SequenceID != 42 {
		t.Fatalf("unexpected sequence: %d", diff.SequenceID)
	}

	// Bid deltas: price 50 changed 100->80, price 49 removed, price 48 added
	bidMap := make(map[int64]int64)
	for _, d := range diff.BidDeltas {
		bidMap[d.Price] = d.Quantity
	}
	if bidMap[50] != 80 {
		t.Fatalf("expected bid 50 qty 80, got %d", bidMap[50])
	}
	if bidMap[49] != 0 {
		t.Fatalf("expected bid 49 removed (qty 0), got %d", bidMap[49])
	}
	if bidMap[48] != 50 {
		t.Fatalf("expected bid 48 added qty 50, got %d", bidMap[48])
	}

	// Ask deltas: price 51 unchanged, price 52 added
	askMap := make(map[int64]int64)
	for _, d := range diff.AskDeltas {
		askMap[d.Price] = d.Quantity
	}
	if _, exists := askMap[51]; exists {
		t.Fatal("price 51 should not appear in diff (unchanged)")
	}
	if askMap[52] != 75 {
		t.Fatalf("expected ask 52 added qty 75, got %d", askMap[52])
	}
}

func TestComputeDepthDiffEmpty(t *testing.T) {
	empty := model.BookSnapshot{Key: "1:YES"}
	diff := ComputeDepthDiff(empty, empty, 1)
	if len(diff.BidDeltas) != 0 || len(diff.AskDeltas) != 0 {
		t.Fatal("expected no deltas for identical empty snapshots")
	}
}

func TestFullSnapshot(t *testing.T) {
	snap := FullSnapshot{
		EpochID:        5,
		GlobalSequence: 100,
		Books: []BookSnapshotData{
			{
				BookKey:  "1:YES",
				LocalSeq: 42,
				Orders: []*model.Order{
					{OrderID: "order-1", MarketID: 1, Outcome: "YES"},
				},
			},
		},
	}

	if snap.EpochID != 5 {
		t.Fatalf("expected epoch 5, got %d", snap.EpochID)
	}
	if len(snap.Books) != 1 {
		t.Fatalf("expected 1 book, got %d", len(snap.Books))
	}
	if snap.Books[0].LocalSeq != 42 {
		t.Fatalf("expected local seq 42, got %d", snap.Books[0].LocalSeq)
	}
}
