package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	accountclient "funnyoption/internal/account/client"
	"funnyoption/internal/api/dto"
	oracleservice "funnyoption/internal/oracle/service"
	"funnyoption/internal/shared/assets"
	sharedauth "funnyoption/internal/shared/auth"
	sharedkafka "funnyoption/internal/shared/kafka"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	Logger                *slog.Logger
	KafkaPublisher        sharedkafka.Publisher
	KafkaTopics           sharedkafka.Topics
	AccountClient         accountclient.AccountClient
	QueryStore            QueryStore
	OperatorWallets       []string
	DefaultOperatorUserID int64
	ExpectedChainID       int64
	ExpectedVaultAddress  string
}

type OrderHandler struct {
	logger              *slog.Logger
	publisher           sharedkafka.Publisher
	topics              sharedkafka.Topics
	account             accountclient.AccountClient
	store               QueryStore
	operatorWallets     map[string]struct{}
	operatorUserID      int64
	expectedChainID     int64
	expectedVaultAddr   string
	bootstrapReplayGate *bootstrapReplayGate
}

type collectionResponse[T any] struct {
	Items []T `json:"items"`
}

func NewOrderHandler(deps Dependencies) *OrderHandler {
	expectedChainID := deps.ExpectedChainID
	if expectedChainID <= 0 {
		expectedChainID = 97
	}
	return &OrderHandler{
		logger:              deps.Logger,
		publisher:           deps.KafkaPublisher,
		topics:              deps.KafkaTopics,
		account:             deps.AccountClient,
		store:               deps.QueryStore,
		operatorWallets:     normalizeOperatorWalletSet(deps.OperatorWallets),
		operatorUserID:      deps.DefaultOperatorUserID,
		expectedChainID:     expectedChainID,
		expectedVaultAddr:   sharedauth.NormalizeHex(deps.ExpectedVaultAddress),
		bootstrapReplayGate: newBootstrapReplayGate(),
	}
}

func normalizeCollectionItems[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}

func writeCollectionResponse[T any](ctx *gin.Context, statusCode int, items []T) {
	ctx.JSON(statusCode, collectionResponse[T]{Items: normalizeCollectionItems(items)})
}

func (h *OrderHandler) validateTradingKeyDomain(chainID int64, vaultAddress string) error {
	if chainID != h.expectedChainID {
		return fmt.Errorf("chain_id does not match target chain")
	}
	if sharedauth.NormalizeHex(vaultAddress) != h.expectedVaultAddr {
		return fmt.Errorf("vault_address does not match target vault")
	}
	return nil
}

func (h *OrderHandler) CreateMarket(ctx *gin.Context) {
	var req dto.CreateMarketRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req = dto.NormalizeCreateMarketRequest(req)
	operator, ok := h.requirePrivilegedOperator(ctx, req.Operator, req.OperatorMessage())
	if !ok {
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}
	req.CreatedBy = h.privilegedOperatorUserID()
	if strings.TrimSpace(req.Title) == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}
	if req.OpenAt > 0 && req.CloseAt > 0 && req.CloseAt < req.OpenAt {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "close_at must be greater than or equal to open_at"})
		return
	}
	if req.CloseAt > 0 && req.ResolveAt > 0 && req.ResolveAt < req.CloseAt {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "resolve_at must be greater than or equal to close_at"})
		return
	}
	normalizedOptions, err := dto.NormalizeMarketOptions(req.Options)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "options must contain at least two unique entries with key and label"})
		return
	}
	req.Options = normalizedOptions
	if normalizeMarketStatus(req.Status) == "OPEN" && !dto.IsBinaryTradingOptions(req.Options) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "only YES/NO binary options can enter OPEN status in the current engine"})
		return
	}
	categoryKey := dto.NormalizeMarketCategoryKey(req.CategoryKey, req.Metadata)
	contract, _, err := oracleservice.ParseContract(categoryKey, marketOptionKeys(req.Options), req.ResolveAt, req.Metadata)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if contract != nil {
		req.Metadata = oracleservice.CanonicalizeMetadata(req.Metadata, &contract.Metadata)
	}
	if req.MarketID <= 0 {
		req.MarketID = time.Now().UnixMilli()
	}
	req.Metadata = mergeCreateMarketMetadata(req.Metadata, req.CoverImageURL, req.CoverSourceURL, req.CoverSourceName, operator)

	market, err := h.store.CreateMarket(ctx, req)
	if err != nil {
		if errors.Is(err, ErrInvalidMarketCategory) || errors.Is(err, dto.ErrInvalidMarketOptions) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("create market failed", "market_id", req.MarketID, "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "create market failed"})
		return
	}
	ctx.JSON(http.StatusCreated, market)
}

func mergeCreateMarketMetadata(raw json.RawMessage, coverImageURL, coverSourceURL, coverSourceName string, operator *verifiedOperatorAction) json.RawMessage {
	metadata := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &metadata); err != nil {
			metadata = map[string]any{}
		}
	}
	if metadata == nil {
		metadata = map[string]any{}
	}

	if coverImageURL = strings.TrimSpace(coverImageURL); coverImageURL != "" {
		metadata["cover_image_url"] = coverImageURL
		metadata["coverImage"] = coverImageURL
		metadata["coverImageUrl"] = coverImageURL
	}
	if coverSourceURL = strings.TrimSpace(coverSourceURL); coverSourceURL != "" {
		metadata["cover_source_url"] = coverSourceURL
		metadata["sourceUrl"] = coverSourceURL
	}
	if coverSourceName = strings.TrimSpace(coverSourceName); coverSourceName != "" {
		metadata["cover_source_name"] = coverSourceName
		metadata["sourceName"] = coverSourceName
		metadata["coverSourceName"] = coverSourceName
	}
	if operator != nil {
		metadata["operatorWalletAddress"] = operator.WalletAddress
		metadata["operatorRequestedAt"] = operator.RequestedAt
		metadata["operatorService"] = "shared-api"
	}

	if len(metadata) == 0 {
		return nil
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return raw
	}
	return encoded
}

func (h *OrderHandler) ListMarkets(ctx *gin.Context) {
	var req dto.ListMarketsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}

	markets, err := h.store.ListMarkets(ctx, req)
	if err != nil {
		h.logger.Error("list markets failed", "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "list markets failed"})
		return
	}
	writeCollectionResponse(ctx, http.StatusOK, markets)
}

func (h *OrderHandler) GetMarket(ctx *gin.Context) {
	marketID, err := strconv.ParseInt(ctx.Param("market_id"), 10, 64)
	if err != nil || marketID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid market_id"})
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}

	market, err := h.store.GetMarket(ctx, marketID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "market not found"})
			return
		}
		h.logger.Error("get market failed", "market_id", marketID, "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "get market failed"})
		return
	}
	ctx.JSON(http.StatusOK, market)
}

func (h *OrderHandler) CreateFirstLiquidity(ctx *gin.Context) {
	marketID, err := strconv.ParseInt(ctx.Param("market_id"), 10, 64)
	if err != nil || marketID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid market_id"})
		return
	}

	var req dto.CreateFirstLiquidityRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	outcome, ok := dto.NormalizeBinaryOutcome(req.Outcome)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "operator bootstrap proof requires outcome to be YES or NO"})
		return
	}
	req.Outcome = outcome
	if req.Price <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "operator bootstrap proof requires price to be positive"})
		return
	}
	operator, ok := h.requirePrivilegedOperator(ctx, req.Operator, req.OperatorMessage(marketID))
	if !ok {
		return
	}
	if req.UserID <= 0 || req.Quantity <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id and quantity must be positive"})
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}
	if h.account == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "account client is not configured"})
		return
	}
	if h.publisher == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "kafka publisher is not configured"})
		return
	}

	unlockBootstrap := h.bootstrapReplayGate.Lock(req.BootstrapSemanticKey(marketID))
	defer unlockBootstrap()

	orderID := req.BootstrapOrderID(marketID)
	replayed, err := h.bootstrapOrderAlreadyAccepted(ctx, orderID)
	if err != nil {
		h.logger.Error("check bootstrap order uniqueness failed", "order_id", orderID, "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "check bootstrap order uniqueness failed"})
		return
	}
	if replayed {
		ctx.JSON(http.StatusConflict, gin.H{
			"error":    "operator bootstrap order already accepted",
			"order_id": orderID,
		})
		return
	}

	market, err := h.store.GetMarket(ctx, marketID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "market not found"})
			return
		}
		h.logger.Error("get market for first liquidity failed", "market_id", marketID, "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "get market failed"})
		return
	}
	if !isTradableMarket(market) {
		ctx.JSON(http.StatusConflict, gin.H{"error": "market is not tradable"})
		return
	}

	firstLiquidityID := sharedkafka.NewID("liq")
	commandID := sharedkafka.NewID("cmd")
	collateralAsset := assets.NormalizeAsset(market.CollateralAsset)
	collateralRef := firstLiquidityCollateralRef(firstLiquidityID)
	collateralDebit, err := assets.WinningPayoutAmount(req.Quantity)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	debitResult, err := h.account.DebitBalance(ctx, req.UserID, collateralAsset, collateralDebit, "FIRST_LIQUIDITY_COLLATERAL", collateralRef)
	if err != nil {
		status := http.StatusBadGateway
		if looksLikeInsufficientBalance(err) {
			status = http.StatusConflict
		}
		ctx.JSON(status, gin.H{"error": err.Error()})
		return
	}
	if !debitResult.Applied {
		ctx.JSON(http.StatusConflict, gin.H{"error": "first liquidity collateral debit was not applied"})
		return
	}

	issued := []issuedFirstLiquidityInventory{
		{
			Outcome:       "YES",
			PositionAsset: assets.PositionAsset(marketID, "YES"),
			Quantity:      req.Quantity,
		},
		{
			Outcome:       "NO",
			PositionAsset: assets.PositionAsset(marketID, "NO"),
			Quantity:      req.Quantity,
		},
	}

	nowMillis := time.Now().UnixMilli()
	for index := range issued {
		inventory := &issued[index]
		positionRef := firstLiquidityPositionRef(firstLiquidityID, inventory.Outcome)
		creditResult, err := h.account.CreditBalance(ctx, req.UserID, inventory.PositionAsset, inventory.Quantity, "FIRST_LIQUIDITY_POSITION", positionRef)
		if err != nil {
			h.rollbackFirstLiquidity(ctx, firstLiquidityID, req.UserID, collateralAsset, collateralDebit, issued)
			ctx.JSON(http.StatusBadGateway, gin.H{"error": "credit first-liquidity inventory failed"})
			return
		}
		if !creditResult.Applied {
			h.rollbackFirstLiquidity(ctx, firstLiquidityID, req.UserID, collateralAsset, collateralDebit, issued)
			ctx.JSON(http.StatusConflict, gin.H{"error": "first-liquidity inventory credit was not applied"})
			return
		}
		inventory.Credited = true

		event := sharedkafka.PositionChangedEvent{
			EventID:          sharedkafka.NewID("evt_pos"),
			TraceID:          firstLiquidityID,
			SourceTradeID:    firstLiquidityID,
			UserID:           req.UserID,
			MarketID:         marketID,
			Outcome:          inventory.Outcome,
			PositionAsset:    inventory.PositionAsset,
			DeltaQuantity:    inventory.Quantity,
			OccurredAtMillis: nowMillis,
		}
		if err := h.publisher.PublishJSON(ctx, h.topics.PositionChange, fmt.Sprintf("%d:%s", marketID, inventory.Outcome), event); err != nil {
			h.rollbackFirstLiquidity(ctx, firstLiquidityID, req.UserID, collateralAsset, collateralDebit, issued)
			h.logger.Error("publish first-liquidity position event failed", "market_id", marketID, "user_id", req.UserID, "outcome", inventory.Outcome, "err", err)
			ctx.JSON(http.StatusBadGateway, gin.H{"error": "publish first-liquidity position event failed"})
			return
		}
		inventory.EventPublished = true
	}

	freezeRecord, err := h.account.PreFreeze(ctx, accountclient.FreezeRequest{
		UserID:  req.UserID,
		Asset:   assets.PositionAsset(marketID, req.Outcome),
		RefType: "ORDER",
		RefID:   orderID,
		Amount:  req.Quantity,
	})
	if err != nil {
		h.rollbackFirstLiquidity(ctx, firstLiquidityID, req.UserID, collateralAsset, collateralDebit, issued)
		status := http.StatusBadGateway
		if looksLikeInsufficientBalance(err) {
			status = http.StatusConflict
		}
		ctx.JSON(status, gin.H{"error": err.Error()})
		return
	}

	command := sharedkafka.OrderCommand{
		CommandID:         commandID,
		TraceID:           firstLiquidityID,
		OrderID:           orderID,
		FreezeID:          freezeRecord.FreezeID,
		FreezeAsset:       freezeRecord.Asset,
		FreezeAmount:      freezeRecord.Amount,
		CollateralAsset:   collateralAsset,
		UserID:            req.UserID,
		MarketID:          marketID,
		Outcome:           req.Outcome,
		Side:              "SELL",
		Type:              "LIMIT",
		TimeInForce:       "GTC",
		Price:             req.Price,
		Quantity:          req.Quantity,
		RequestedAtMillis: operator.RequestedAt,
	}
	if err := h.publisher.PublishJSON(ctx, h.topics.OrderCommand, fmt.Sprintf("%d:%s", marketID, req.Outcome), command); err != nil {
		if releaseErr := h.account.ReleaseFreeze(ctx, freezeRecord.FreezeID); releaseErr != nil {
			h.logger.Error("release first-liquidity bootstrap freeze after publish failure failed", "freeze_id", freezeRecord.FreezeID, "err", releaseErr)
		}
		h.rollbackFirstLiquidity(ctx, firstLiquidityID, req.UserID, collateralAsset, collateralDebit, issued)
		h.logger.Error("publish first-liquidity bootstrap order failed", "command_id", commandID, "order_id", orderID, "market_id", marketID, "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "publish first-liquidity bootstrap order failed"})
		return
	}

	responseInventory := make([]dto.FirstLiquidityInventoryResponse, 0, len(issued))
	for _, inventory := range issued {
		responseInventory = append(responseInventory, dto.FirstLiquidityInventoryResponse{
			Outcome:       inventory.Outcome,
			PositionAsset: inventory.PositionAsset,
			Quantity:      inventory.Quantity,
		})
	}

	ctx.JSON(http.StatusAccepted, dto.CreateFirstLiquidityResponse{
		FirstLiquidityID: firstLiquidityID,
		MarketID:         marketID,
		UserID:           req.UserID,
		CollateralAsset:  collateralAsset,
		CollateralDebit:  collateralDebit,
		Inventory:        responseInventory,
		Status:           "ISSUED",
		OrderID:          orderID,
		OrderStatus:      "QUEUED",
	})
}

func (h *OrderHandler) ListOrders(ctx *gin.Context) {
	var req dto.ListOrdersRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}

	orders, err := h.store.ListOrders(ctx, req)
	if err != nil {
		h.logger.Error("list orders failed", "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "list orders failed"})
		return
	}
	writeCollectionResponse(ctx, http.StatusOK, orders)
}

func (h *OrderHandler) ListTrades(ctx *gin.Context) {
	var req dto.ListTradesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}

	trades, err := h.store.ListTrades(ctx, req)
	if err != nil {
		h.logger.Error("list trades failed", "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "list trades failed"})
		return
	}
	writeCollectionResponse(ctx, http.StatusOK, trades)
}

func (h *OrderHandler) ListBalances(ctx *gin.Context) {
	var req dto.ListBalancesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}

	items, err := h.store.ListBalances(ctx, req)
	if err != nil {
		h.logger.Error("list balances failed", "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "list balances failed"})
		return
	}
	writeCollectionResponse(ctx, http.StatusOK, items)
}

func (h *OrderHandler) ListPositions(ctx *gin.Context) {
	var req dto.ListPositionsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}

	items, err := h.store.ListPositions(ctx, req)
	if err != nil {
		h.logger.Error("list positions failed", "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "list positions failed"})
		return
	}
	writeCollectionResponse(ctx, http.StatusOK, items)
}

func (h *OrderHandler) ListPayouts(ctx *gin.Context) {
	var req dto.ListPayoutsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}

	items, err := h.store.ListPayouts(ctx, req)
	if err != nil {
		h.logger.Error("list payouts failed", "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "list payouts failed"})
		return
	}
	writeCollectionResponse(ctx, http.StatusOK, items)
}

func (h *OrderHandler) ListFreezes(ctx *gin.Context) {
	var req dto.ListFreezesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}

	items, err := h.store.ListFreezes(ctx, req)
	if err != nil {
		h.logger.Error("list freezes failed", "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "list freezes failed"})
		return
	}
	writeCollectionResponse(ctx, http.StatusOK, items)
}

func (h *OrderHandler) ListLedgerEntries(ctx *gin.Context) {
	var req dto.ListLedgerEntriesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}

	items, err := h.store.ListLedgerEntries(ctx, req)
	if err != nil {
		h.logger.Error("list ledger entries failed", "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "list ledger entries failed"})
		return
	}
	writeCollectionResponse(ctx, http.StatusOK, items)
}

func (h *OrderHandler) ListLedgerPostings(ctx *gin.Context) {
	entryID := strings.TrimSpace(ctx.Param("entry_id"))
	if entryID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "entry_id is required"})
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}

	items, err := h.store.ListLedgerPostings(ctx, entryID)
	if err != nil {
		h.logger.Error("list ledger postings failed", "entry_id", entryID, "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "list ledger postings failed"})
		return
	}
	writeCollectionResponse(ctx, http.StatusOK, items)
}

func (h *OrderHandler) GetLiabilityReport(ctx *gin.Context) {
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}

	items, err := h.store.BuildLiabilityReport(ctx)
	if err != nil {
		h.logger.Error("build liability report failed", "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "build liability report failed"})
		return
	}
	writeCollectionResponse(ctx, http.StatusOK, items)
}

func (h *OrderHandler) CreateTradingKeyChallenge(ctx *gin.Context) {
	var req dto.CreateTradingKeyChallengeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}
	if h.expectedChainID <= 0 || h.expectedVaultAddr == "" {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "trading key auth is not configured"})
		return
	}

	req.WalletAddress = sharedauth.NormalizeHex(req.WalletAddress)
	req.VaultAddress = sharedauth.NormalizeHex(req.VaultAddress)
	if !common.IsHexAddress(req.WalletAddress) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "wallet_address is invalid"})
		return
	}
	if err := h.validateTradingKeyDomain(req.ChainID, req.VaultAddress); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	challengeBytes := make([]byte, 32)
	if _, err := rand.Read(challengeBytes); err != nil {
		h.logger.Error("generate trading key challenge failed", "err", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "generate trading key challenge failed"})
		return
	}

	req.ChallengeID = sharedkafka.NewID("tkc")
	req.Challenge = "0x" + hex.EncodeToString(challengeBytes)
	req.ChallengeExpiresAt = time.Now().Add(5 * time.Minute).UnixMilli()

	item, err := h.store.CreateTradingKeyChallenge(ctx, req)
	if err != nil {
		h.logger.Error("create trading key challenge failed", "wallet_address", req.WalletAddress, "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "create trading key challenge failed"})
		return
	}

	ctx.JSON(http.StatusCreated, item)
}

func (h *OrderHandler) RegisterTradingKey(ctx *gin.Context) {
	var req dto.RegisterTradingKeyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}
	if h.expectedChainID <= 0 || h.expectedVaultAddr == "" {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "trading key auth is not configured"})
		return
	}
	if strings.ToUpper(strings.TrimSpace(req.WalletSignatureStandard)) != sharedauth.DefaultWalletSignatureStandard {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "wallet_signature_standard must be EIP712_V4"})
		return
	}

	authz := sharedauth.TradingKeyAuthorization{
		WalletAddress:            req.WalletAddress,
		TradingPublicKey:         req.TradingPublicKey,
		TradingKeyScheme:         req.TradingKeyScheme,
		Scope:                    req.Scope,
		Challenge:                req.Challenge,
		ChallengeExpiresAtMillis: req.ChallengeExpiresAtMillis,
		KeyExpiresAtMillis:       req.KeyExpiresAtMillis,
		ChainID:                  req.ChainID,
		VaultAddress:             req.VaultAddress,
	}
	if err := authz.Validate(time.Now()); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	normalized := authz.Normalize()
	if err := h.validateTradingKeyDomain(normalized.ChainID, normalized.VaultAddress); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	recoveredWallet, err := sharedauth.VerifyTradingKeyAuthorizationSignature(authz, req.WalletSignature)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.SessionID = authz.TradingKeyID()
	req.WalletAddress = normalized.WalletAddress
	req.VaultAddress = normalized.VaultAddress
	req.TradingPublicKey = normalized.TradingPublicKey
	req.TradingKeyScheme = normalized.TradingKeyScheme
	req.Scope = normalized.Scope
	req.Challenge = normalized.Challenge
	req.WalletSignatureStandard = sharedauth.DefaultWalletSignatureStandard

	session, err := h.store.RegisterTradingKey(ctx, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrTradingKeyChallengeConsumed):
			ctx.JSON(http.StatusConflict, gin.H{"error": "trading key challenge already used"})
			return
		case errors.Is(err, ErrTradingKeyChallengeExpired):
			ctx.JSON(http.StatusGone, gin.H{"error": "trading key challenge expired"})
			return
		case errors.Is(err, ErrNotFound):
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "trading key challenge is invalid"})
			return
		default:
			h.logger.Error("register trading key failed", "wallet_address", normalized.WalletAddress, "trading_key_id", req.SessionID, "err", err)
			ctx.JSON(http.StatusBadGateway, gin.H{"error": "register trading key failed"})
			return
		}
	}

	h.logger.Info("trading key registered", "session_id", session.SessionID, "user_id", session.UserID, "wallet_address", recoveredWallet)
	ctx.JSON(http.StatusCreated, session)
}

func (h *OrderHandler) CreateSession(ctx *gin.Context) {
	var req dto.CreateSessionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}
	if req.UserID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id must be positive"})
		return
	}

	grant := sharedauth.SessionGrant{
		WalletAddress:    req.WalletAddress,
		SessionPublicKey: req.SessionPublicKey,
		Scope:            req.Scope,
		ChainID:          req.ChainID,
		Nonce:            req.Nonce,
		IssuedAtMillis:   req.IssuedAtMillis,
		ExpiresAtMillis:  req.ExpiresAtMillis,
	}
	if err := grant.Validate(time.Now()); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	recoveredWallet, err := sharedauth.VerifyGrantSignature(grant, req.WalletSignature)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	normalized := grant.Normalize()
	req.SessionID = grant.SessionID()
	req.WalletAddress = normalized.WalletAddress
	req.SessionPublicKey = normalized.SessionPublicKey
	req.Scope = normalized.Scope
	req.Nonce = normalized.Nonce

	session, err := h.store.CreateSession(ctx, req)
	if err != nil {
		h.logger.Error("create session failed", "user_id", req.UserID, "wallet_address", normalized.WalletAddress, "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "create session failed"})
		return
	}

	h.logger.Info("wallet session created", "session_id", session.SessionID, "user_id", session.UserID, "wallet_address", recoveredWallet)
	ctx.JSON(http.StatusCreated, session)
}

func (h *OrderHandler) GetProfile(ctx *gin.Context) {
	var req dto.GetUserProfileRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}
	if req.UserID <= 0 && strings.TrimSpace(req.WalletAddress) == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id or wallet_address is required"})
		return
	}

	item, err := h.store.GetUserProfile(ctx, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
			return
		}
		h.logger.Error("get profile failed", "user_id", req.UserID, "wallet_address", req.WalletAddress, "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "get profile failed"})
		return
	}
	ctx.JSON(http.StatusOK, item)
}

func (h *OrderHandler) UpdateProfile(ctx *gin.Context) {
	var req dto.UpdateUserProfileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}
	preset, ok := dto.NormalizeAvatarPreset(req.AvatarPreset)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "avatar_preset is invalid"})
		return
	}

	session, err := h.store.GetSession(ctx, req.SessionID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "active session is required"})
			return
		}
		h.logger.Error("get session for profile update failed", "session_id", req.SessionID, "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "profile update failed"})
		return
	}
	if session.UserID != req.UserID || strings.ToUpper(session.Status) != "ACTIVE" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "active session is required"})
		return
	}
	if session.ExpiresAtMillis > 0 && session.ExpiresAtMillis < time.Now().UnixMilli() {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "session is expired"})
		return
	}

	req.AvatarPreset = preset
	req.DisplayName = dto.NormalizeUserDisplayName(req.DisplayName)
	item, err := h.store.UpsertUserProfile(ctx, req, session.WalletAddress)
	if err != nil {
		h.logger.Error("update profile failed", "user_id", req.UserID, "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "profile update failed"})
		return
	}
	ctx.JSON(http.StatusOK, item)
}

func (h *OrderHandler) ListSessions(ctx *gin.Context) {
	var req dto.ListSessionsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}

	items, err := h.store.ListSessions(ctx, req)
	if err != nil {
		h.logger.Error("list sessions failed", "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "list sessions failed"})
		return
	}
	writeCollectionResponse(ctx, http.StatusOK, items)
}

func (h *OrderHandler) ListTradingKeys(ctx *gin.Context) {
	h.ListSessions(ctx)
}

func (h *OrderHandler) RevokeSession(ctx *gin.Context) {
	sessionID := strings.TrimSpace(ctx.Param("session_id"))
	if sessionID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}

	item, err := h.store.RevokeSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		h.logger.Error("revoke session failed", "session_id", sessionID, "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "revoke session failed"})
		return
	}

	ctx.JSON(http.StatusOK, item)
}

func (h *OrderHandler) ListDeposits(ctx *gin.Context) {
	var req dto.ListDepositsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}

	items, err := h.store.ListDeposits(ctx, req)
	if err != nil {
		h.logger.Error("list deposits failed", "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "list deposits failed"})
		return
	}
	writeCollectionResponse(ctx, http.StatusOK, items)
}

func (h *OrderHandler) ListWithdrawals(ctx *gin.Context) {
	var req dto.ListWithdrawalsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}

	items, err := h.store.ListWithdrawals(ctx, req)
	if err != nil {
		h.logger.Error("list withdrawals failed", "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "list withdrawals failed"})
		return
	}
	writeCollectionResponse(ctx, http.StatusOK, items)
}

func (h *OrderHandler) CreateClaimPayout(ctx *gin.Context) {
	eventID := strings.TrimSpace(ctx.Param("event_id"))
	if eventID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "event_id is required"})
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}

	var req dto.CreateClaimPayoutRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.UserID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id must be positive"})
		return
	}
	normalizedReq, err := req.ValidateAddresses()
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.store.CreateClaimRequest(ctx, dto.ClaimPayoutRequest{
		EventID:          eventID,
		UserID:           req.UserID,
		WalletAddress:    normalizedReq.WalletAddress,
		RecipientAddress: normalizedReq.RecipientAddress,
	})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "payout not found"})
			return
		}
		h.logger.Error("create claim payout failed", "event_id", eventID, "user_id", req.UserID, "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "create claim payout failed"})
		return
	}
	ctx.JSON(http.StatusAccepted, item)
}

func (h *OrderHandler) ListChainTransactions(ctx *gin.Context) {
	var req dto.ListChainTransactionsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}

	items, err := h.store.ListChainTransactions(ctx, req)
	if err != nil {
		h.logger.Error("list chain transactions failed", "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "list chain transactions failed"})
		return
	}
	writeCollectionResponse(ctx, http.StatusOK, items)
}

func (h *OrderHandler) CreateOrder(ctx *gin.Context) {
	var req dto.CreateOrderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.MarketID <= 0 || req.Quantity <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "market_id and quantity must be positive"})
		return
	}

	orderID := sharedkafka.NewID("ord")
	commandID := sharedkafka.NewID("cmd")
	now := time.Now()
	requestedAt := req.RequestedAtMillis

	outcome := strings.ToUpper(strings.TrimSpace(req.Outcome))
	side := strings.ToUpper(strings.TrimSpace(req.Side))
	orderType := strings.ToUpper(strings.TrimSpace(req.Type))
	tif := strings.ToUpper(strings.TrimSpace(req.TimeInForce))
	collateralAsset := assets.DefaultCollateralAsset
	bookKey := fmt.Sprintf("%d:%s", req.MarketID, outcome)
	userID := req.UserID

	if strings.TrimSpace(req.SessionID) != "" {
		if requestedAt <= 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "requested_at is required for trading-key orders"})
			return
		}
		if h.store == nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
			return
		}

		session, err := h.store.GetSession(ctx, req.SessionID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				ctx.JSON(http.StatusNotFound, gin.H{"error": "trading key not found"})
				return
			}
			h.logger.Error("get session failed", "session_id", req.SessionID, "err", err)
			ctx.JSON(http.StatusBadGateway, gin.H{"error": "get trading key failed"})
			return
		}
		if session.Status != "ACTIVE" {
			ctx.JSON(http.StatusConflict, gin.H{"error": "trading key is not active"})
			return
		}
		if session.ExpiresAtMillis > 0 && now.UnixMilli() > session.ExpiresAtMillis {
			ctx.JSON(http.StatusConflict, gin.H{"error": "trading key expired"})
			return
		}
		if req.UserID > 0 && req.UserID != session.UserID {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id does not match trading key"})
			return
		}

		intent := sharedauth.OrderIntent{
			SessionID:         session.SessionID,
			WalletAddress:     session.WalletAddress,
			UserID:            session.UserID,
			MarketID:          req.MarketID,
			Outcome:           outcome,
			Side:              side,
			OrderType:         orderType,
			TimeInForce:       tif,
			Price:             req.Price,
			Quantity:          req.Quantity,
			ClientOrderID:     strings.TrimSpace(req.ClientOrderID),
			Nonce:             req.OrderNonce,
			RequestedAtMillis: requestedAt,
		}
		if err := intent.Validate(now); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := sharedauth.VerifyOrderIntentSignature(intent, session.SessionPublicKey, req.SessionSignature); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		nonceAdvanceReq := dto.AdvanceSessionNonceRequest{
			SessionID: session.SessionID,
			Nonce:     req.OrderNonce,
			AuthorizationWitness: func() *sharedauth.OrderAuthorizationWitness {
				witness := sharedauth.BuildOrderAuthorizationWitness(session.UserID, authorizedTradingKeyFromSession(session), intent, req.SessionSignature)
				return &witness
			}(),
		}
		if _, err := h.store.AdvanceSessionNonce(ctx, nonceAdvanceReq); err != nil {
			if errors.Is(err, ErrSessionNonceConflict) {
				ctx.JSON(http.StatusConflict, gin.H{"error": "trading key nonce conflict"})
				return
			}
			if errors.Is(err, ErrNotFound) {
				ctx.JSON(http.StatusNotFound, gin.H{"error": "trading key not found"})
				return
			}
			h.logger.Error("advance session nonce failed", "session_id", session.SessionID, "order_nonce", req.OrderNonce, "err", err)
			ctx.JSON(http.StatusBadGateway, gin.H{"error": "advance trading key nonce failed"})
			return
		}
		userID = session.UserID
	} else if req.Operator != nil {
		if req.UserID <= 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required for operator-authenticated bootstrap orders"})
			return
		}
		if side != "SELL" || orderType != "LIMIT" || tif != "GTC" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "operator-authenticated orders only support bootstrap sell limit GTC orders"})
			return
		}
		if requestedAt > 0 && requestedAt != req.Operator.RequestedAt {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "requested_at must match operator.requested_at"})
			return
		}
		operator, ok := h.requirePrivilegedOperator(ctx, req.Operator, req.BootstrapOperatorMessage())
		if !ok {
			return
		}
		if h.store == nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
			return
		}
		unlockBootstrap := h.bootstrapReplayGate.Lock(req.BootstrapSemanticKey())
		defer unlockBootstrap()

		orderID = req.BootstrapOrderID()
		replayed, err := h.bootstrapOrderAlreadyAccepted(ctx, orderID)
		if err != nil {
			h.logger.Error("check bootstrap order uniqueness failed", "order_id", orderID, "err", err)
			ctx.JSON(http.StatusBadGateway, gin.H{"error": "check bootstrap order uniqueness failed"})
			return
		}
		if replayed {
			ctx.JSON(http.StatusConflict, gin.H{
				"error":    "operator bootstrap order already accepted",
				"order_id": orderID,
			})
			return
		}
		requestedAt = operator.RequestedAt
	} else {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "trading-key-backed trade authorization or operator proof is required"})
		return
	}

	if userID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "user_id or trading key id is required"})
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}

	market, err := h.store.GetMarket(ctx, req.MarketID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "market not found"})
			return
		}
		h.logger.Error("get market for order failed", "market_id", req.MarketID, "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "get market failed"})
		return
	}
	if !isTradableMarket(market) {
		ctx.JSON(http.StatusConflict, gin.H{"error": "market is not tradable"})
		return
	}

	freezeAsset, freezeAmount, err := calculateFreeze(side, orderType, req.MarketID, outcome, req.Price, req.Quantity)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.account == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "account client is not configured"})
		return
	}

	freezeRecord, err := h.account.PreFreeze(ctx, accountclient.FreezeRequest{
		UserID:  userID,
		Asset:   freezeAsset,
		RefType: "ORDER",
		RefID:   orderID,
		Amount:  freezeAmount,
	})
	if err != nil {
		status := http.StatusBadRequest
		errText := strings.ToLower(err.Error())
		if strings.Contains(errText, "insufficient") {
			status = http.StatusConflict
		}
		ctx.JSON(status, gin.H{"error": err.Error()})
		return
	}

	command := sharedkafka.OrderCommand{
		CommandID:         commandID,
		TraceID:           strings.TrimSpace(req.TraceID),
		OrderID:           orderID,
		ClientOrderID:     strings.TrimSpace(req.ClientOrderID),
		FreezeID:          freezeRecord.FreezeID,
		FreezeAsset:       freezeRecord.Asset,
		FreezeAmount:      freezeRecord.Amount,
		CollateralAsset:   collateralAsset,
		UserID:            userID,
		MarketID:          req.MarketID,
		Outcome:           outcome,
		Side:              side,
		Type:              orderType,
		TimeInForce:       tif,
		Price:             req.Price,
		Quantity:          req.Quantity,
		RequestedAtMillis: requestedAt,
	}

	if err := h.publisher.PublishJSON(ctx, h.topics.OrderCommand, bookKey, command); err != nil {
		if releaseErr := h.account.ReleaseFreeze(ctx, freezeRecord.FreezeID); releaseErr != nil {
			h.logger.Error("release freeze after publish failure failed", "freeze_id", freezeRecord.FreezeID, "err", releaseErr)
		}
		h.logger.Error("publish order command failed", "command_id", commandID, "order_id", orderID, "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "publish order command failed"})
		return
	}

	h.logger.Info("order command published", "command_id", commandID, "order_id", orderID, "book_key", bookKey, "freeze_id", freezeRecord.FreezeID, "freeze_amount", freezeRecord.Amount)
	ctx.JSON(http.StatusAccepted, dto.CreateOrderResponse{
		CommandID: commandID,
		OrderID:   orderID,
		FreezeID:  freezeRecord.FreezeID,
		Asset:     freezeRecord.Asset,
		Amount:    freezeRecord.Amount,
		Topic:     h.topics.OrderCommand,
		Status:    "QUEUED",
	})
}

func calculateFreeze(side, orderType string, marketID int64, outcome string, price, quantity int64) (string, int64, error) {
	if quantity <= 0 {
		return "", 0, errors.New("quantity must be positive")
	}
	switch side {
	case "BUY":
		switch orderType {
		case "LIMIT":
			if price <= 0 {
				return "", 0, errors.New("limit order requires positive price")
			}
			if quantity > 0 && price > math.MaxInt64/quantity {
				return "", 0, errors.New("freeze amount overflow")
			}
			return assets.DefaultCollateralAsset, price * quantity, nil
		case "MARKET":
			return "", 0, errors.New("market order pre-freeze is not implemented yet")
		default:
			return "", 0, fmt.Errorf("unsupported order type: %s", orderType)
		}
	case "SELL":
		if strings.TrimSpace(outcome) == "" {
			return "", 0, errors.New("sell order requires outcome")
		}
		return assets.PositionAsset(marketID, outcome), quantity, nil
	default:
		return "", 0, fmt.Errorf("unsupported side: %s", side)
	}
}

func isTradableMarket(market dto.MarketResponse) bool {
	return marketIsOpenForTrading(market, time.Now().Unix())
}

type issuedFirstLiquidityInventory struct {
	Outcome        string
	PositionAsset  string
	Quantity       int64
	Credited       bool
	EventPublished bool
}

func (h *OrderHandler) rollbackFirstLiquidity(ctx context.Context, firstLiquidityID string, userID int64, collateralAsset string, collateralAmount int64, issued []issuedFirstLiquidityInventory) {
	for index := len(issued) - 1; index >= 0; index-- {
		inventory := issued[index]
		if inventory.Credited {
			if _, err := h.account.DebitBalance(
				ctx,
				userID,
				inventory.PositionAsset,
				inventory.Quantity,
				"FIRST_LIQUIDITY_POSITION_ROLLBACK",
				firstLiquidityPositionRollbackRef(firstLiquidityID, inventory.Outcome),
			); err != nil {
				h.logger.Error("rollback first-liquidity balance failed", "first_liquidity_id", firstLiquidityID, "user_id", userID, "asset", inventory.PositionAsset, "err", err)
			}
		}
		if inventory.EventPublished {
			rollbackEvent := sharedkafka.PositionChangedEvent{
				EventID:          sharedkafka.NewID("evt_pos"),
				TraceID:          firstLiquidityID,
				SourceTradeID:    firstLiquidityRollbackTradeRef(firstLiquidityID, inventory.Outcome),
				UserID:           userID,
				MarketID:         marketIDFromPositionAsset(inventory.PositionAsset),
				Outcome:          inventory.Outcome,
				PositionAsset:    inventory.PositionAsset,
				DeltaQuantity:    -inventory.Quantity,
				OccurredAtMillis: time.Now().UnixMilli(),
			}
			if err := h.publisher.PublishJSON(ctx, h.topics.PositionChange, fmt.Sprintf("%d:%s", marketIDFromPositionAsset(inventory.PositionAsset), inventory.Outcome), rollbackEvent); err != nil {
				h.logger.Error("rollback first-liquidity position event failed", "first_liquidity_id", firstLiquidityID, "user_id", userID, "asset", inventory.PositionAsset, "err", err)
			}
		}
	}

	if collateralAmount <= 0 {
		return
	}
	if _, err := h.account.CreditBalance(
		ctx,
		userID,
		collateralAsset,
		collateralAmount,
		"FIRST_LIQUIDITY_COLLATERAL_ROLLBACK",
		firstLiquidityCollateralRollbackRef(firstLiquidityID),
	); err != nil {
		h.logger.Error("rollback first-liquidity collateral failed", "first_liquidity_id", firstLiquidityID, "user_id", userID, "asset", collateralAsset, "err", err)
	}
}

func looksLikeInsufficientBalance(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "insufficient")
}

func firstLiquidityCollateralRef(firstLiquidityID string) string {
	return strings.TrimSpace(firstLiquidityID) + ":collateral"
}

func firstLiquidityCollateralRollbackRef(firstLiquidityID string) string {
	return strings.TrimSpace(firstLiquidityID) + ":collateral:rollback"
}

func firstLiquidityPositionRef(firstLiquidityID, outcome string) string {
	return strings.TrimSpace(firstLiquidityID) + ":" + strings.ToUpper(strings.TrimSpace(outcome))
}

func firstLiquidityPositionRollbackRef(firstLiquidityID, outcome string) string {
	return firstLiquidityPositionRef(firstLiquidityID, outcome) + ":rollback"
}

func firstLiquidityRollbackTradeRef(firstLiquidityID, outcome string) string {
	return strings.TrimSpace(firstLiquidityID) + ":rollback:" + strings.ToUpper(strings.TrimSpace(outcome))
}

func marketIDFromPositionAsset(asset string) int64 {
	parts := strings.Split(strings.ToUpper(strings.TrimSpace(asset)), ":")
	if len(parts) < 3 {
		return 0
	}
	marketID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0
	}
	return marketID
}

func (h *OrderHandler) ResolveMarket(ctx *gin.Context) {
	marketID, err := strconv.ParseInt(ctx.Param("market_id"), 10, 64)
	if err != nil || marketID <= 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid market_id"})
		return
	}

	var req dto.ResolveMarketRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	outcome, ok := dto.NormalizeBinaryOutcome(req.Outcome)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "outcome must be YES or NO"})
		return
	}
	req.Outcome = outcome
	if _, ok := h.requirePrivilegedOperator(ctx, req.Operator, req.OperatorMessage(marketID)); !ok {
		return
	}
	if h.store == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "query store is not configured"})
		return
	}
	market, err := h.store.GetMarket(ctx, marketID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "market not found"})
			return
		}
		h.logger.Error("load market before resolve failed", "market_id", marketID, "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "load market failed"})
		return
	}
	if effectiveMarketStatusAt(market.Status, market.CloseAt, time.Now().Unix()) == "RESOLVED" {
		ctx.JSON(http.StatusConflict, gin.H{"error": "market is already resolved"})
		return
	}
	if oracleservice.HasOracleResolutionMode(market.Metadata) {
		state, exists, err := h.store.GetMarketResolution(ctx, marketID)
		if err != nil {
			h.logger.Error("load market resolution before resolve failed", "market_id", marketID, "err", err)
			ctx.JSON(http.StatusBadGateway, gin.H{"error": "load market resolution failed"})
			return
		}
		if exists {
			switch normalizeUpper(state.Status) {
			case "OBSERVED", "RESOLVED":
				ctx.JSON(http.StatusConflict, gin.H{"error": "oracle market resolution is already observed or resolved"})
				return
			}
		}
	}

	event := sharedkafka.MarketEvent{
		EventID:          sharedkafka.NewID("evt_market"),
		TraceID:          strings.TrimSpace(ctx.GetHeader("X-Trace-Id")),
		MarketID:         marketID,
		Status:           "RESOLVED",
		ResolvedOutcome:  req.Outcome,
		OccurredAtMillis: time.Now().UnixMilli(),
	}

	key := fmt.Sprintf("%d", marketID)
	if err := h.publisher.PublishJSON(ctx, h.topics.MarketEvent, key, event); err != nil {
		h.logger.Error("publish market resolve event failed", "market_id", marketID, "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "publish market resolve event failed"})
		return
	}

	ctx.JSON(http.StatusAccepted, gin.H{
		"market_id":        marketID,
		"resolved_outcome": event.ResolvedOutcome,
		"status":           event.Status,
		"topic":            h.topics.MarketEvent,
	})
}

func marketOptionKeys(options []dto.MarketOption) []string {
	keys := make([]string, 0, len(options))
	for _, option := range options {
		keys = append(keys, option.Key)
	}
	return keys
}

func normalizeUpper(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}
