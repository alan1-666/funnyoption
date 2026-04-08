package order

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	accountclient "funnyoption/internal/account/client"
	"funnyoption/internal/api/dto"
	"funnyoption/internal/shared/assets"
	sharedkafka "funnyoption/internal/shared/kafka"
)

// MarketStore is the minimal read interface the order service needs for market
// state and rollup freeze checks.
type MarketStore interface {
	GetMarket(ctx context.Context, marketID int64) (dto.MarketResponse, error)
	GetRollupFreezeState(ctx context.Context) (dto.RollupFreezeStateResponse, error)
}

// OrderExistenceChecker is an optional interface for upstream duplicate-order
// detection. When set, the OrderService rejects client-supplied OrderIDs that
// already exist before publishing to Kafka.
type OrderExistenceChecker interface {
	OrderExists(ctx context.Context, orderID string) (bool, error)
}

// Dependencies bundles everything the OrderService requires.
type Dependencies struct {
	Logger     *slog.Logger
	Account    accountclient.AccountClient
	Publisher  sharedkafka.Publisher
	Topics     sharedkafka.Topics
	Store      MarketStore
	DedupCheck OrderExistenceChecker // optional; nil skips dedup
}

// Service encapsulates the core order-submission workflow independent of any
// transport layer (HTTP, gRPC, FIX, etc.).
type Service struct {
	logger     *slog.Logger
	account    accountclient.AccountClient
	publisher  sharedkafka.Publisher
	topics     sharedkafka.Topics
	store      MarketStore
	dedupCheck OrderExistenceChecker
}

func NewService(deps Dependencies) *Service {
	return &Service{
		logger:     deps.Logger,
		account:    deps.Account,
		publisher:  deps.Publisher,
		topics:     deps.Topics,
		store:      deps.Store,
		dedupCheck: deps.DedupCheck,
	}
}

// SubmitOrder validates, freezes, and publishes an order command to Kafka.
// It is transport-agnostic: the caller is responsible for authentication and
// HTTP response formatting.
func (s *Service) SubmitOrder(ctx context.Context, req SubmitRequest) (*SubmitResult, error) {
	outcome := strings.ToUpper(strings.TrimSpace(req.Outcome))
	side := strings.ToUpper(strings.TrimSpace(req.Side))
	orderType := strings.ToUpper(strings.TrimSpace(req.Type))
	tif := strings.ToUpper(strings.TrimSpace(req.TimeInForce))

	if req.MarketID <= 0 || req.Quantity <= 0 || req.UserID <= 0 {
		return nil, fmt.Errorf("%w: market_id, quantity, and user_id must be positive", ErrValidationFailed)
	}
	if err := ValidateOrderFields(outcome, side, orderType, tif, req.Price, req.Quantity); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidationFailed, err)
	}

	if err := s.requireRollupNotFrozen(ctx); err != nil {
		return nil, err
	}

	market, err := s.store.GetMarket(ctx, req.MarketID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMarketNotFound, err)
	}
	if !marketIsOpenForTrading(market, time.Now().Unix()) {
		return nil, ErrMarketNotTradable
	}

	freezeAsset, freezeAmount, err := CalculateFreeze(side, orderType, req.MarketID, outcome, req.Price, req.Quantity)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidationFailed, err)
	}

	orderID := req.OrderID
	clientSuppliedID := orderID != ""
	if orderID == "" {
		orderID = sharedkafka.NewID("ord")
	}
	commandID := sharedkafka.NewID("cmd")
	bookKey := fmt.Sprintf("%d:%s", req.MarketID, outcome)

	if clientSuppliedID && s.dedupCheck != nil {
		exists, err := s.dedupCheck.OrderExists(ctx, orderID)
		if err != nil {
			s.logger.Warn("dedup check failed, proceeding", "order_id", orderID, "err", err)
		} else if exists {
			return nil, fmt.Errorf("%w: duplicate order_id %s", ErrValidationFailed, orderID)
		}
	}

	freezeRecord, err := s.account.PreFreeze(ctx, accountclient.FreezeRequest{
		UserID:  req.UserID,
		Asset:   freezeAsset,
		RefType: "ORDER",
		RefID:   orderID,
		Amount:  freezeAmount,
	})
	if err != nil {
		if looksLikeInsufficientBalance(err) {
			return nil, fmt.Errorf("%w: %v", ErrInsufficientFunds, err)
		}
		return nil, err
	}

	commandType := orderType
	commandTIF := tif
	commandPrice := req.Price
	if orderType == "MARKET" {
		commandType = "LIMIT"
		commandTIF = "IOC"
		if side == "BUY" {
			commandPrice = 99
		} else {
			commandPrice = 1
		}
	}

	command := sharedkafka.OrderCommand{
		CommandID:         commandID,
		TraceID:           strings.TrimSpace(req.TraceID),
		OrderID:           orderID,
		ClientOrderID:     strings.TrimSpace(req.ClientOrderID),
		FreezeID:          freezeRecord.FreezeID,
		FreezeAsset:       freezeRecord.Asset,
		FreezeAmount:      freezeRecord.Amount,
		CollateralAsset:   assets.DefaultCollateralAsset,
		UserID:            req.UserID,
		MarketID:          req.MarketID,
		Outcome:           outcome,
		BookKey:           bookKey,
		Side:              side,
		Type:              commandType,
		TimeInForce:       commandTIF,
		Price:             commandPrice,
		Quantity:          req.Quantity,
		RequestedAtMillis: req.RequestedAt,
	}

	if err := s.publisher.PublishJSON(ctx, s.topics.OrderCommand, bookKey, command); err != nil {
		if releaseErr := s.account.ReleaseFreeze(ctx, freezeRecord.FreezeID); releaseErr != nil {
			s.logger.Error("release freeze after publish failure", "freeze_id", freezeRecord.FreezeID, "err", releaseErr)
		}
		return nil, fmt.Errorf("%w: %v", ErrPublishFailed, err)
	}

	s.logger.Info("order submitted",
		"command_id", commandID,
		"order_id", orderID,
		"book_key", bookKey,
		"freeze_id", freezeRecord.FreezeID,
		"freeze_amount", freezeRecord.Amount,
	)

	return &SubmitResult{
		CommandID: commandID,
		OrderID:   orderID,
		FreezeID:  freezeRecord.FreezeID,
		Asset:     freezeRecord.Asset,
		Amount:    freezeRecord.Amount,
	}, nil
}

// SeedLiquidity mints a complete set (YES + NO positions) for a market and
// places a bootstrap SELL LIMIT GTC order on the specified outcome.
func (s *Service) SeedLiquidity(ctx context.Context, req SeedLiquidityRequest) (*SeedLiquidityResult, error) {
	if req.MarketID <= 0 || req.Quantity <= 0 || req.Price <= 0 || req.OperatorUserID <= 0 {
		return nil, fmt.Errorf("%w: all fields must be positive", ErrValidationFailed)
	}
	outcome := strings.ToUpper(strings.TrimSpace(req.Outcome))
	if outcome != "YES" && outcome != "NO" {
		return nil, fmt.Errorf("%w: outcome must be YES or NO", ErrValidationFailed)
	}

	if err := s.requireRollupNotFrozen(ctx); err != nil {
		return nil, err
	}
	market, err := s.store.GetMarket(ctx, req.MarketID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMarketNotFound, err)
	}
	if !marketIsOpenForTrading(market, time.Now().Unix()) {
		return nil, ErrMarketNotTradable
	}

	firstLiquidityID := sharedkafka.NewID("liq")
	collateralAsset := assets.NormalizeAsset(market.CollateralAsset)
	collateralDebit, err := assets.WinningPayoutAmount(req.Quantity)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidationFailed, err)
	}

	collateralRef := firstLiquidityID + ":collateral"
	debitResult, err := s.account.DebitBalance(ctx, req.OperatorUserID, collateralAsset, collateralDebit, "FIRST_LIQUIDITY_COLLATERAL", collateralRef)
	if err != nil {
		if looksLikeInsufficientBalance(err) {
			return nil, fmt.Errorf("%w: %v", ErrInsufficientFunds, err)
		}
		return nil, err
	}
	if !debitResult.Applied {
		return nil, fmt.Errorf("%w: collateral debit was not applied", ErrInsufficientFunds)
	}

	inventory := []InventoryItem{
		{Outcome: "YES", PositionAsset: assets.PositionAsset(req.MarketID, "YES"), Quantity: req.Quantity},
		{Outcome: "NO", PositionAsset: assets.PositionAsset(req.MarketID, "NO"), Quantity: req.Quantity},
	}

	nowMillis := time.Now().UnixMilli()
	for _, item := range inventory {
		positionRef := firstLiquidityID + ":" + item.Outcome
		creditResult, err := s.account.CreditBalance(ctx, req.OperatorUserID, item.PositionAsset, item.Quantity, "FIRST_LIQUIDITY_POSITION", positionRef)
		if err != nil || !creditResult.Applied {
			s.logger.Error("first-liquidity credit failed, manual reconciliation needed",
				"first_liquidity_id", firstLiquidityID,
				"outcome", item.Outcome,
				"err", err,
			)
			return nil, fmt.Errorf("credit first-liquidity inventory failed")
		}

		event := sharedkafka.PositionChangedEvent{
			EventID:          sharedkafka.NewID("evt_pos"),
			TraceID:          firstLiquidityID,
			SourceTradeID:    firstLiquidityID,
			UserID:           req.OperatorUserID,
			MarketID:         req.MarketID,
			Outcome:          item.Outcome,
			PositionAsset:    item.PositionAsset,
			DeltaQuantity:    item.Quantity,
			OccurredAtMillis: nowMillis,
		}
		if err := s.publisher.PublishJSON(ctx, s.topics.PositionChange, fmt.Sprintf("%d:%s", req.MarketID, item.Outcome), event); err != nil {
			s.logger.Error("first-liquidity position event publish failed", "outcome", item.Outcome, "err", err)
		}
	}

	// Place a bootstrap SELL order for the specified outcome.
	submitResult, err := s.SubmitOrder(ctx, SubmitRequest{
		UserID:      req.OperatorUserID,
		MarketID:    req.MarketID,
		Outcome:     outcome,
		Side:        "SELL",
		Type:        "LIMIT",
		TimeInForce: "GTC",
		Price:       req.Price,
		Quantity:    req.Quantity,
		TraceID:     firstLiquidityID,
		RequestedAt: nowMillis,
	})
	if err != nil {
		s.logger.Error("first-liquidity bootstrap order failed", "first_liquidity_id", firstLiquidityID, "err", err)
		return nil, err
	}

	return &SeedLiquidityResult{
		FirstLiquidityID: firstLiquidityID,
		OrderID:          submitResult.OrderID,
		CollateralDebit:  collateralDebit,
		Inventory:        inventory,
	}, nil
}

func (s *Service) requireRollupNotFrozen(ctx context.Context) error {
	if s.store == nil {
		return nil
	}
	item, err := s.store.GetRollupFreezeState(ctx)
	if err != nil {
		return nil
	}
	if item.Frozen {
		return ErrRollupFrozen
	}
	return nil
}

func marketIsOpenForTrading(market dto.MarketResponse, nowUnix int64) bool {
	status := strings.ToUpper(strings.TrimSpace(market.Status))
	if status != "" && status != "OPEN" {
		return false
	}
	if market.CloseAt > 0 && nowUnix >= market.CloseAt {
		return false
	}
	return true
}

func looksLikeInsufficientBalance(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "insufficient")
}

// MarketMetadataUsesOracle checks if market metadata indicates oracle resolution.
func MarketMetadataUsesOracle(metadata json.RawMessage) bool {
	if len(metadata) == 0 {
		return false
	}
	var parsed map[string]any
	if err := json.Unmarshal(metadata, &parsed); err != nil {
		return false
	}
	resolution, ok := parsed["resolution"].(map[string]any)
	if !ok {
		return false
	}
	mode, _ := resolution["mode"].(string)
	return strings.EqualFold(mode, "ORACLE")
}
