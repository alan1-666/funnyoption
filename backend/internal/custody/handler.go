package custody

import (
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	accountclient "funnyoption/internal/account/client"
	"funnyoption/internal/shared/assets"
	sharedkafka "funnyoption/internal/shared/kafka"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	logger           *slog.Logger
	store            *Store
	saas             *SaaSClient
	account          accountclient.AccountClient
	depositToken     string
	chain            string
	network          string
	coin             string
	chainDecimals    int
	accountingDigits int
}

type HandlerDeps struct {
	Logger            *slog.Logger
	Store             *Store
	SaaS              *SaaSClient
	Account           accountclient.AccountClient
	DepositToken      string
	Chain             string
	Network           string
	Coin              string
	ChainDecimals     int
	AccountingDigits  int
}

func NewHandler(d HandlerDeps) *Handler {
	chain := d.Chain
	if chain == "" {
		chain = "binance"
	}
	network := d.Network
	if network == "" {
		network = "testnet"
	}
	coin := d.Coin
	if coin == "" {
		coin = "USDT"
	}
	chainDecimals := d.ChainDecimals
	if chainDecimals <= 0 {
		chainDecimals = assets.DefaultCollateralChainDecimals
	}
	accountingDigits := d.AccountingDigits
	if accountingDigits <= 0 {
		accountingDigits = assets.DefaultCollateralDisplayDigits
	}
	return &Handler{
		logger:           d.Logger,
		store:            d.Store,
		saas:             d.SaaS,
		account:          d.Account,
		depositToken:     d.DepositToken,
		chain:            chain,
		network:          network,
		coin:             coin,
		chainDecimals:    chainDecimals,
		accountingDigits: accountingDigits,
	}
}

type DepositNotifyRequest struct {
	BizID   string `json:"biz_id"`
	ChainID int64  `json:"chain_id"`
	TxHash  string `json:"tx_hash"`
	TxIndex int    `json:"tx_index"`
	Address string `json:"address"`
	Asset   string `json:"asset"`
	Amount  string `json:"amount"`
}

type NotifyResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// DepositNotify handles the callback from Wallet SaaS when a deposit is confirmed.
// POST /internal/custody/deposit/notify
func (h *Handler) DepositNotify(ctx *gin.Context) {
	if h.depositToken != "" {
		token := ctx.GetHeader("x-deposit-token")
		if token != h.depositToken {
			ctx.JSON(http.StatusUnauthorized, NotifyResponse{Code: 401, Message: "unauthorized"})
			return
		}
	}

	var req DepositNotifyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, NotifyResponse{Code: 400, Message: err.Error()})
		return
	}
	if req.BizID == "" || req.Address == "" || req.Amount == "" {
		ctx.JSON(http.StatusBadRequest, NotifyResponse{Code: 400, Message: "biz_id, address, amount required"})
		return
	}

	processed, err := h.store.IsDepositProcessed(ctx, req.BizID)
	if err != nil {
		h.logger.Error("check deposit idempotency failed", "biz_id", req.BizID, "err", err)
		ctx.JSON(http.StatusInternalServerError, NotifyResponse{Code: 500, Message: "internal error"})
		return
	}
	if processed {
		ctx.JSON(http.StatusOK, NotifyResponse{Code: 0, Message: "already processed"})
		return
	}

	userID, err := h.store.LookupUserByAddress(ctx, req.Address, h.chain, h.network)
	if err != nil {
		h.logger.Error("lookup user by address failed", "address", req.Address, "err", err)
		ctx.JSON(http.StatusInternalServerError, NotifyResponse{Code: 500, Message: "internal error"})
		return
	}
	if userID == 0 {
		h.logger.Warn("deposit address not mapped to any user", "address", req.Address, "biz_id", req.BizID)
		ctx.JSON(http.StatusOK, NotifyResponse{Code: 10000, Message: "address not mapped"})
		return
	}

	creditAmount, err := h.parseChainAmount(req.Asset, req.Amount)
	if err != nil {
		h.logger.Error("parse chain amount failed", "amount", req.Amount, "asset", req.Asset, "err", err)
		ctx.JSON(http.StatusBadRequest, NotifyResponse{Code: 400, Message: "invalid amount"})
		return
	}
	if creditAmount <= 0 {
		ctx.JSON(http.StatusOK, NotifyResponse{Code: 0, Message: "dust deposit ignored"})
		return
	}

	depositID := sharedkafka.NewID("cdep")
	normalizedAsset := assets.NormalizeAsset(req.Asset)

	_, err = h.account.CreditBalance(ctx, userID, normalizedAsset, creditAmount, "CUSTODY_DEPOSIT", depositID)
	if err != nil {
		h.logger.Error("credit balance failed", "user_id", userID, "amount", creditAmount, "err", err)
		ctx.JSON(http.StatusInternalServerError, NotifyResponse{Code: 500, Message: "credit failed"})
		return
	}

	if err := h.store.InsertDeposit(ctx, DepositRecord{
		BizID:        req.BizID,
		UserID:       userID,
		Address:      req.Address,
		Asset:        normalizedAsset,
		ChainAmount:  req.Amount,
		CreditAmount: creditAmount,
		ChainID:      req.ChainID,
		TxHash:       req.TxHash,
		TxIndex:      req.TxIndex,
	}); err != nil {
		h.logger.Error("insert deposit record failed", "biz_id", req.BizID, "err", err)
	}

	h.logger.Info("custody deposit credited",
		"biz_id", req.BizID, "user_id", userID,
		"asset", normalizedAsset, "credit", creditAmount,
		"chain_amount", req.Amount, "tx_hash", req.TxHash)

	ctx.JSON(http.StatusOK, NotifyResponse{Code: 0, Message: "ok"})
}

// GetDepositAddress returns the user's custody deposit address, creating one via SaaS if needed.
// GET /api/v1/custody/deposit-address
func (h *Handler) GetDepositAddress(ctx *gin.Context) {
	userIDRaw, exists := ctx.Get("api.authenticated_user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDRaw.(int64)

	address, err := h.store.GetUserAddress(ctx, userID, h.chain, h.network, h.coin)
	if err != nil {
		h.logger.Error("get user address failed", "user_id", userID, "err", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	if address != "" {
		ctx.JSON(http.StatusOK, gin.H{
			"address": address,
			"chain":   h.chain,
			"network": h.network,
			"coin":    h.coin,
		})
		return
	}

	if h.saas == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "custody service not configured"})
		return
	}

	accountID := fmt.Sprintf("%d", userID)
	if err := h.saas.UpsertAccount(ctx, accountID); err != nil {
		h.logger.Error("saas upsert account failed", "user_id", userID, "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "custody account creation failed"})
		return
	}

	resp, err := h.saas.CreateAddress(ctx, accountID, h.chain, h.coin, h.network)
	if err != nil {
		h.logger.Error("saas create address failed", "user_id", userID, "err", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "address creation failed"})
		return
	}

	if err := h.store.SaveAddressMapping(ctx, AddressMapping{
		UserID:  userID,
		Chain:   h.chain,
		Network: h.network,
		Coin:    h.coin,
		Address: resp.Address,
		KeyID:   resp.KeyID,
	}); err != nil {
		h.logger.Error("save address mapping failed", "user_id", userID, "address", resp.Address, "err", err)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"address": resp.Address,
		"chain":   h.chain,
		"network": h.network,
		"coin":    h.coin,
	})
}

type WithdrawRequest struct {
	ToAddress string `json:"to_address" binding:"required"`
	Amount    int64  `json:"amount" binding:"required,gt=0"`
	Asset     string `json:"asset"`
}

// RequestWithdraw freezes user balance and submits a withdrawal to SaaS.
// POST /api/v1/custody/withdraw
func (h *Handler) RequestWithdraw(ctx *gin.Context) {
	userIDRaw, exists := ctx.Get("api.authenticated_user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDRaw.(int64)

	var req WithdrawRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	asset := assets.NormalizeAsset(req.Asset)
	if asset == "" {
		asset = assets.DefaultCollateralAsset
	}

	withdrawID := sharedkafka.NewID("cwdr")

	freeze, err := h.account.PreFreeze(ctx, accountclient.FreezeRequest{
		UserID:  userID,
		Asset:   asset,
		Amount:  req.Amount,
		RefType: "CUSTODY_WITHDRAW",
		RefID:   withdrawID,
	})
	if err != nil {
		h.logger.Error("freeze balance failed", "user_id", userID, "amount", req.Amount, "err", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "insufficient balance"})
		return
	}

	if h.saas == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "custody service not configured"})
		return
	}

	chainAmount, err := assets.AccountingToAssetChainAmount(asset, req.Amount)
	if err != nil {
		h.logger.Error("convert to chain amount failed", "amount", req.Amount, "err", err)
		_ = h.account.ReleaseFreeze(ctx, freeze.FreezeID)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "amount conversion failed"})
		return
	}

	_, keyID, _ := h.store.GetUserAddressWithKeyID(ctx, userID, h.chain, h.network, h.coin)

	saasResp, err := h.saas.SubmitWithdraw(ctx, CreateWithdrawRequest{
		AccountID: fmt.Sprintf("%d", userID),
		OrderID:   withdrawID,
		KeyID:     keyID,
		Chain:     h.chain,
		Network:   h.network,
		Coin:      h.coin,
		To:        req.ToAddress,
		Amount:    strconv.FormatInt(chainAmount, 10),
	})
	if err != nil {
		h.logger.Error("saas withdraw failed", "user_id", userID, "withdraw_id", withdrawID, "err", err)
		_ = h.account.ReleaseFreeze(ctx, freeze.FreezeID)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "withdrawal submission failed"})
		return
	}

	_ = h.account.ReleaseFreeze(ctx, freeze.FreezeID)
	_, _ = h.account.DebitBalance(ctx, userID, asset, req.Amount, "CUSTODY_WITHDRAW", withdrawID)

	h.logger.Info("custody withdrawal submitted",
		"user_id", userID, "withdraw_id", withdrawID,
		"to", req.ToAddress, "amount", req.Amount,
		"saas_status", saasResp.Status, "saas_tx_hash", saasResp.TxHash)

	ctx.JSON(http.StatusOK, gin.H{
		"withdraw_id": withdrawID,
		"status":      saasResp.Status,
		"tx_hash":     saasResp.TxHash,
	})
}

// parseChainAmount converts the SaaS amount string (in chain decimals, e.g. 18 for BSC USDT)
// to FunnyOption's internal accounting amount (e.g. 2 decimal places for USDT).
// BSC USDT is 18 decimals, but our collateral is 6 decimals by default. SaaS sends raw wei string.
func (h *Handler) parseChainAmount(assetName, rawAmount string) (int64, error) {
	rawAmount = strings.TrimSpace(rawAmount)
	if rawAmount == "" {
		return 0, fmt.Errorf("empty amount")
	}

	bigAmt, ok := new(big.Int).SetString(rawAmount, 10)
	if !ok {
		return 0, fmt.Errorf("invalid amount: %s", rawAmount)
	}
	if bigAmt.Sign() <= 0 {
		return 0, nil
	}

	chainDecimals := h.chainDecimals
	accountingDigits := h.accountingDigits
	diff := chainDecimals - accountingDigits
	if diff > 0 {
		divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(diff)), nil)
		bigAmt.Div(bigAmt, divisor)
	}

	if !bigAmt.IsInt64() {
		return 0, fmt.Errorf("amount overflows int64")
	}
	return bigAmt.Int64(), nil
}
