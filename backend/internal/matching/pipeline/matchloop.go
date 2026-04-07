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

// MatchLoop is the single-threaded matching core.
// It ONLY reads from InputRB and writes to OutputRB — zero IO, zero allocation on hot path.
type MatchLoop struct {
	logger   *slog.Logger
	eng      *engine.Engine
	inputRB  *ringbuffer.RingBuffer[MatchCommand]
	outputRB *ringbuffer.RingBuffer[MatchResult]
	idle     *ringbuffer.IdleStrategy

	matched  atomic.Uint64
	batches  atomic.Uint64
	outStall atomic.Uint64
}

const matchDrainBatch = 64

func NewMatchLoop(
	logger *slog.Logger,
	eng *engine.Engine,
	inputRB *ringbuffer.RingBuffer[MatchCommand],
	outputRB *ringbuffer.RingBuffer[MatchResult],
) *MatchLoop {
	return &MatchLoop{
		logger:   logger,
		eng:      eng,
		inputRB:  inputRB,
		outputRB: outputRB,
		idle:     ringbuffer.DefaultIdleStrategy(),
	}
}

func (m *MatchLoop) Run(ctx context.Context) {
	m.logger.Info("match loop started")
	defer m.logger.Info("match loop stopped")

	buf := make([]MatchCommand, matchDrainBatch)

	for {
		n := m.inputRB.DrainTo(buf, matchDrainBatch)
		if n == 0 {
			m.idle.Idle()
			if ctx.Err() != nil {
				return
			}
			continue
		}
		m.idle.Reset()
		m.batches.Add(1)

		nowMillis := time.Now().UnixMilli()

		for i := 0; i < n; i++ {
			cmd := &buf[i]

			switch cmd.Action {
			case ActionPlace:
				m.handlePlace(ctx, cmd, nowMillis)
			case ActionCancel:
				m.handleCancel(ctx, cmd)
			}
		}
	}
}

func (m *MatchLoop) handlePlace(ctx context.Context, cmd *MatchCommand, nowMillis int64) {
	order := cmd.ToOrder(nowMillis)

	result, err := m.eng.PlaceOrder(order)
	mr := MatchResult{
		Command:  *cmd,
		Result:   result,
		Rejected: err != nil,
	}

	m.publishResult(ctx, mr)
	m.matched.Add(1)
}

func (m *MatchLoop) handleCancel(ctx context.Context, cmd *MatchCommand) {
	order := &model.Order{
		OrderID:  cmd.OrderID,
		MarketID: cmd.MarketID,
		Outcome:  cmd.Outcome,
		Side:     cmd.Side.ToModel(),
		Price:    cmd.Price,
	}
	cancelResult, _ := m.eng.CancelOrders([]*model.Order{order}, cmd.CancelReason.ToModel())

	for _, cancelled := range cancelResult.Orders {
		mr := MatchResult{
			Command: *cmd,
			Result: engine.Result{
				Order:    cancelled,
				Affected: []*model.Order{cancelled},
			},
		}
		m.publishResult(ctx, mr)
	}
	for _, book := range cancelResult.Books {
		mr := MatchResult{
			Command: *cmd,
			Result:  engine.Result{Book: book},
		}
		m.publishResult(ctx, mr)
	}
	m.matched.Add(1)
}

func (m *MatchLoop) publishResult(ctx context.Context, mr MatchResult) {
	for !m.outputRB.TryPublish(mr) {
		m.outStall.Add(1)
		m.idle.Idle()
		if ctx.Err() != nil {
			return
		}
	}
	m.idle.Reset()
}

// RestoreOrder delegates to the underlying engine for startup recovery.
func (m *MatchLoop) RestoreOrder(order *model.Order) error {
	return m.eng.RestoreOrder(order)
}

func (m *MatchLoop) Stats() (matched, batches, outStall uint64) {
	return m.matched.Load(), m.batches.Load(), m.outStall.Load()
}
