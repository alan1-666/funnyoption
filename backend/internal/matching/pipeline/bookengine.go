package pipeline

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"funnyoption/internal/matching/engine"
	"funnyoption/internal/matching/model"
	"funnyoption/internal/matching/ringbuffer"
)

const (
	bookRBSize     = 1024
	matchDrainBatch = 64
)

// BookEngine is a fully isolated matching unit for a single BookKey.
// It owns its own InputRB, Engine (single book), and MatchLoop goroutine.
// Results are forwarded to the shared OutputCh (fan-in channel).
type BookEngine struct {
	bookKey  string
	eng      *engine.Engine
	inputRB  *ringbuffer.RingBuffer[MatchCommand]
	outputCh chan<- MatchResult
	logger   *slog.Logger
	idle     *ringbuffer.IdleStrategy

	matched  atomic.Uint64
	batches  atomic.Uint64
	outStall atomic.Uint64
}

func NewBookEngine(bookKey string, logger *slog.Logger, sequence *uint64, outputCh chan<- MatchResult) *BookEngine {
	eng := engine.NewWithSequence(logger, sequence)
	return &BookEngine{
		bookKey:  bookKey,
		eng:      eng,
		inputRB:  ringbuffer.New[MatchCommand](bookRBSize),
		outputCh: outputCh,
		logger:   logger,
		idle:     ringbuffer.DefaultIdleStrategy(),
	}
}

func (be *BookEngine) Run(ctx context.Context) {
	be.logger.Info("book engine started", "book_key", be.bookKey)
	defer be.logger.Info("book engine stopped", "book_key", be.bookKey)

	buf := make([]MatchCommand, matchDrainBatch)

	for {
		n := be.inputRB.DrainTo(buf, matchDrainBatch)
		if n == 0 {
			be.idle.Idle()
			if ctx.Err() != nil {
				return
			}
			continue
		}
		be.idle.Reset()
		be.batches.Add(1)

		nowMillis := time.Now().UnixMilli()

		for i := 0; i < n; i++ {
			cmd := &buf[i]
			switch cmd.Action {
			case ActionPlace:
				be.handlePlace(ctx, cmd, nowMillis)
			case ActionCancel:
				be.handleCancel(ctx, cmd)
			}
		}
	}
}

func (be *BookEngine) handlePlace(ctx context.Context, cmd *MatchCommand, nowMillis int64) {
	order := cmd.ToOrder(nowMillis)
	result, err := be.eng.PlaceOrder(order)
	mr := MatchResult{
		Command:  *cmd,
		Result:   result,
		Rejected: err != nil,
	}
	be.sendResult(ctx, mr)
	be.matched.Add(1)
}

func (be *BookEngine) handleCancel(ctx context.Context, cmd *MatchCommand) {
	order := &model.Order{
		OrderID:  cmd.OrderID,
		MarketID: cmd.MarketID,
		Outcome:  cmd.Outcome,
		Side:     cmd.Side.ToModel(),
		Price:    cmd.Price,
	}
	cancelResult, _ := be.eng.CancelOrders([]*model.Order{order}, cmd.CancelReason.ToModel())

	for _, cancelled := range cancelResult.Orders {
		mr := MatchResult{
			Command: *cmd,
			Result: engine.Result{
				Order:    cancelled,
				Affected: []*model.Order{cancelled},
			},
		}
		be.sendResult(ctx, mr)
	}
	for _, book := range cancelResult.Books {
		mr := MatchResult{
			Command: *cmd,
			Result:  engine.Result{Book: book},
		}
		be.sendResult(ctx, mr)
	}
	be.matched.Add(1)
}

func (be *BookEngine) sendResult(ctx context.Context, mr MatchResult) {
	select {
	case be.outputCh <- mr:
	case <-ctx.Done():
	}
}

func (be *BookEngine) RestoreOrder(order *model.Order) error {
	return be.eng.RestoreOrder(order)
}

func (be *BookEngine) TryPublish(cmd MatchCommand) bool {
	return be.inputRB.TryPublish(cmd)
}

func (be *BookEngine) Stats() (matched, batches, outStall uint64) {
	return be.matched.Load(), be.batches.Load(), be.outStall.Load()
}
