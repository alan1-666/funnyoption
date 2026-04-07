package marketmaker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	sharedkafka "funnyoption/internal/shared/kafka"
)

// Service is the main market maker orchestration loop.
type Service struct {
	logger   *slog.Logger
	cfg      Config
	api      *OperatorAPIClient
	strategy *SpreadStrategy
	state    *StateBook
}

func NewService(logger *slog.Logger, cfg Config) *Service {
	return &Service{
		logger:   logger,
		cfg:      cfg,
		api:      NewOperatorAPIClient(cfg),
		strategy: NewSpreadStrategy(cfg),
		state:    NewStateBook(),
	}
}

// Run starts the main loop: discover markets, seed liquidity, maintain quotes.
func (s *Service) Run(ctx context.Context) error {
	if err := s.discoverAndSeed(ctx); err != nil {
		s.logger.Error("initial market discovery failed", "err", err)
	}

	ticker := time.NewTicker(s.cfg.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("market maker shutting down")
			return ctx.Err()
		case <-ticker.C:
			if err := s.refreshQuotes(ctx); err != nil {
				s.logger.Warn("refresh quotes failed", "err", err)
			}
		}
	}
}

// HandleMarketEvent is the Kafka consumer handler for market events.
func (s *Service) HandleMarketEvent(ctx context.Context, msg sharedkafka.Message) error {
	var event sharedkafka.MarketEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return err
	}

	switch event.Status {
	case "OPEN":
		if !s.cfg.SeedOnNewMarket {
			return nil
		}
		s.logger.Info("new market detected, seeding liquidity", "market_id", event.MarketID)
		if err := s.seedMarket(ctx, event.MarketID); err != nil {
			s.logger.Warn("seed new market failed", "market_id", event.MarketID, "err", err)
		}
	case "RESOLVED", "CLOSED":
		s.state.Remove(event.MarketID)
		s.logger.Info("market closed/resolved, removing state", "market_id", event.MarketID)
	}
	return nil
}

// HandleTradeMatched is the Kafka consumer handler for trade events.
// The bot updates inventory when its own orders are filled.
func (s *Service) HandleTradeMatched(ctx context.Context, msg sharedkafka.Message) error {
	var event sharedkafka.TradeMatchedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return err
	}

	botUserID := s.cfg.BotUserID

	// Only react to trades where the bot is the maker (passive fill).
	if event.MakerUserID != botUserID {
		return nil
	}

	state, ok := s.state.Get(event.MarketID)
	if !ok {
		return nil
	}

	state.ApplyTradeFill(event.Outcome, event.Quantity)
	state.RemoveOrder(event.MakerOrderID)

	s.logger.Info("bot order filled",
		"market_id", event.MarketID,
		"outcome", event.Outcome,
		"price", event.Price,
		"quantity", event.Quantity,
		"yes_inventory", state.YesInventory,
		"no_inventory", state.NoInventory,
	)

	return nil
}

// HandleOrderEvent tracks status changes for the bot's own orders.
func (s *Service) HandleOrderEvent(ctx context.Context, msg sharedkafka.Message) error {
	var event sharedkafka.OrderEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return err
	}
	if event.UserID != s.cfg.BotUserID {
		return nil
	}

	state, ok := s.state.Get(event.MarketID)
	if !ok {
		return nil
	}

	switch event.Status {
	case "FILLED":
		state.RemoveOrder(event.OrderID)
	case "CANCELLED", "REJECTED":
		state.RemoveOrder(event.OrderID)
	case "PARTIALLY_FILLED":
		if ord, exists := state.ActiveOrders[event.OrderID]; exists {
			ord.Quantity = event.RemainingQuantity
			state.ActiveOrders[event.OrderID] = ord
		}
	}
	return nil
}

func (s *Service) discoverAndSeed(ctx context.Context) error {
	markets, err := s.api.ListOpenMarkets(ctx)
	if err != nil {
		return fmt.Errorf("list markets: %w", err)
	}

	s.logger.Info("discovered markets", "count", len(markets))
	for _, market := range markets {
		state := s.state.GetOrCreate(market.MarketID)
		state.Title = market.Title
		if !state.Seeded && s.cfg.SeedOnNewMarket {
			if err := s.seedMarket(ctx, market.MarketID); err != nil {
				s.logger.Warn("seed market failed", "market_id", market.MarketID, "err", err)
			}
		}
	}
	return nil
}

func (s *Service) seedMarket(ctx context.Context, marketID int64) error {
	state := s.state.GetOrCreate(marketID)
	if state.Seeded {
		return nil
	}

	qty := s.cfg.DefaultQuantity
	mid := s.cfg.DefaultMidPrice
	halfSpread := s.cfg.DefaultSpread / 2
	if halfSpread < 1 {
		halfSpread = 1
	}
	yesPrice := clampPrice(mid + halfSpread)
	noPrice := clampPrice((100 - mid) + halfSpread)

	// Seed YES side: mint complete set + SELL YES
	_, err := s.api.SeedFirstLiquidity(ctx, marketID, qty, yesPrice, "YES")
	if err != nil && !errors.Is(err, ErrAlreadySeeded) {
		return fmt.Errorf("seed YES: %w", err)
	}

	// Seed NO side: use bootstrap order to SELL NO
	_, err = s.api.PlaceBootstrapOrder(ctx, marketID, "NO", noPrice, qty)
	if err != nil && !errors.Is(err, ErrOrderAlreadyExists) {
		return fmt.Errorf("seed NO: %w", err)
	}

	state.Seeded = true
	state.AddInventory(qty)
	s.logger.Info("seeded market",
		"market_id", marketID,
		"yes_price", yesPrice,
		"no_price", noPrice,
		"quantity", qty,
	)
	return nil
}

func (s *Service) refreshQuotes(ctx context.Context) error {
	for _, marketID := range s.state.AllMarketIDs() {
		state, ok := s.state.Get(marketID)
		if !ok || !state.Seeded {
			continue
		}

		desired := s.strategy.ComputeQuotes(marketID, state.MidPrice, state.YesInventory, state.NoInventory)

		for _, q := range desired.Quotes {
			if s.hasMatchingOrder(state, q) {
				continue
			}

			resp, err := s.api.PlaceBootstrapOrder(ctx, marketID, q.Outcome, q.Price, q.Quantity)
			if err != nil {
				if errors.Is(err, ErrOrderAlreadyExists) {
					continue
				}
				s.logger.Warn("place quote failed",
					"market_id", marketID,
					"outcome", q.Outcome,
					"price", q.Price,
					"err", err,
				)
				continue
			}
			state.TrackOrder(resp.OrderID, q.Outcome, q.Price, q.Quantity)
		}
	}
	return nil
}

func (s *Service) hasMatchingOrder(state *MarketState, q Quote) bool {
	for _, o := range state.ActiveOrders {
		if o.Outcome == q.Outcome && o.Price == q.Price {
			return true
		}
	}
	return false
}
