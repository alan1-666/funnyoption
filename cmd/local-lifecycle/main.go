package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	accountclient "funnyoption/internal/account/client"
	"funnyoption/internal/api/dto"
	chainservice "funnyoption/internal/chain/service"
	sharedauth "funnyoption/internal/shared/auth"
	"funnyoption/internal/shared/config"
	shareddb "funnyoption/internal/shared/db"
	sharedkafka "funnyoption/internal/shared/kafka"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

type walletIdentity struct {
	Label      string
	UserID     int64
	PrivateKey *ecdsa.PrivateKey
	Address    string
}

type sessionContext struct {
	UserID        int64
	WalletAddress string
	SessionID     string
	SessionPubKey string
	SessionPriv   ed25519.PrivateKey
	LastNonce     uint64
	ChainID       int64
	VaultAddress  string
}

type apiClient struct {
	baseURL      string
	client       *http.Client
	sessionAuths map[int64]string
}

type collectionResponse[T any] struct {
	Items []T `json:"items"`
}

type marketResponse struct {
	MarketID        int64  `json:"market_id"`
	Status          string `json:"status"`
	ResolvedOutcome string `json:"resolved_outcome"`
	Runtime         struct {
		TradeCount       int64 `json:"trade_count"`
		MatchedNotional  int64 `json:"matched_notional"`
		ActiveOrderCount int64 `json:"active_order_count"`
		PayoutCount      int64 `json:"payout_count"`
	} `json:"runtime"`
}

type balanceResponse struct {
	UserID    int64  `json:"user_id"`
	Asset     string `json:"asset"`
	Available int64  `json:"available"`
	Frozen    int64  `json:"frozen"`
}

type positionResponse struct {
	MarketID        int64  `json:"market_id"`
	UserID          int64  `json:"user_id"`
	Outcome         string `json:"outcome"`
	Quantity        int64  `json:"quantity"`
	SettledQuantity int64  `json:"settled_quantity"`
}

type depositResponse struct {
	DepositID     string `json:"deposit_id"`
	UserID        int64  `json:"user_id"`
	WalletAddress string `json:"wallet_address"`
	VaultAddress  string `json:"vault_address"`
	Asset         string `json:"asset"`
	ChainName     string `json:"chain_name"`
	NetworkName   string `json:"network_name"`
	TxHash        string `json:"tx_hash"`
	LogIndex      int64  `json:"log_index"`
	BlockNumber   int64  `json:"block_number"`
	Amount        int64  `json:"amount"`
	Status        string `json:"status"`
	CreditedAt    int64  `json:"credited_at"`
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt     int64  `json:"updated_at"`
}

type payoutResponse struct {
	EventID         string `json:"event_id"`
	MarketID        int64  `json:"market_id"`
	UserID          int64  `json:"user_id"`
	WinningOutcome  string `json:"winning_outcome"`
	SettledQuantity int64  `json:"settled_quantity"`
	PayoutAmount    int64  `json:"payout_amount"`
	Status          string `json:"status"`
}

type rollupForcedWithdrawalResponse struct {
	RequestID          int64  `json:"request_id"`
	WalletAddress      string `json:"wallet_address"`
	RecipientAddress   string `json:"recipient_address"`
	Amount             int64  `json:"amount"`
	RequestedAt        int64  `json:"requested_at"`
	DeadlineAt         int64  `json:"deadline_at"`
	SatisfiedClaimID   string `json:"satisfied_claim_id"`
	SatisfiedAt        int64  `json:"satisfied_at"`
	FrozenAt           int64  `json:"frozen_at"`
	Status             string `json:"status"`
	SatisfactionStatus string `json:"satisfaction_status"`
}

type rollupEscapeCollateralClaimResponse struct {
	BatchID           int64    `json:"batch_id"`
	AccountID         int64    `json:"account_id"`
	StateRoot         string   `json:"state_root"`
	CollateralAsset   string   `json:"collateral_asset"`
	MerkleRoot        string   `json:"merkle_root"`
	LeafCount         int64    `json:"leaf_count"`
	TotalAmount       int64    `json:"total_amount"`
	WalletAddress     string   `json:"wallet_address"`
	ClaimAmount       int64    `json:"claim_amount"`
	LeafIndex         int64    `json:"leaf_index"`
	LeafHash          string   `json:"leaf_hash"`
	ProofHashes       []string `json:"proof_hashes"`
	ClaimID           string   `json:"claim_id"`
	ClaimStatus       string   `json:"claim_status"`
	ClaimTxHash       string   `json:"claim_tx_hash"`
	ClaimSubmittedAt  int64    `json:"claim_submitted_at"`
	ClaimedAt         int64    `json:"claimed_at"`
	AnchorStatus      string   `json:"anchor_status"`
	AnchorTxHash      string   `json:"anchor_tx_hash"`
	AnchorSubmittedAt int64    `json:"anchor_submitted_at"`
	AnchoredAt        int64    `json:"anchored_at"`
	LastError         string   `json:"last_error"`
	LastErrorAt       int64    `json:"last_error_at"`
	CreatedAt         int64    `json:"created_at"`
	UpdatedAt         int64    `json:"updated_at"`
}

type rollupFreezeStateResponse struct {
	Frozen    bool  `json:"frozen"`
	FrozenAt  int64 `json:"frozen_at"`
	RequestID int64 `json:"request_id"`
	UpdatedAt int64 `json:"updated_at"`
}

type tradeResponse struct {
	TradeID     string `json:"trade_id"`
	MarketID    int64  `json:"market_id"`
	Outcome     string `json:"outcome"`
	Price       int64  `json:"price"`
	Quantity    int64  `json:"quantity"`
	TakerUserID int64  `json:"taker_user_id"`
	MakerUserID int64  `json:"maker_user_id"`
	TakerSide   string `json:"taker_side"`
	MakerSide   string `json:"maker_side"`
	OccurredAt  int64  `json:"occurred_at"`
}

type orderResponse struct {
	OrderID           string `json:"order_id"`
	UserID            int64  `json:"user_id"`
	MarketID          int64  `json:"market_id"`
	Outcome           string `json:"outcome"`
	Side              string `json:"side"`
	Status            string `json:"status"`
	Price             int64  `json:"price"`
	Quantity          int64  `json:"quantity"`
	FilledQuantity    int64  `json:"filled_quantity"`
	RemainingQuantity int64  `json:"remaining_quantity"`
}

type remoteSession struct {
	SessionID        string `json:"session_id"`
	UserID           int64  `json:"user_id"`
	WalletAddress    string `json:"wallet_address"`
	SessionPublicKey string `json:"session_public_key"`
	Scope            string `json:"scope"`
	ChainID          int64  `json:"chain_id"`
	VaultAddress     string `json:"vault_address"`
	LastOrderNonce   uint64 `json:"last_order_nonce"`
	Status           string `json:"status"`
	ExpiresAtMillis  int64  `json:"expires_at"`
}

type createOrderResult struct {
	CommandID string `json:"command_id"`
	OrderID   string `json:"order_id"`
	FreezeID  string `json:"freeze_id"`
	Status    string `json:"status"`
}

type createFirstLiquidityResult struct {
	FirstLiquidityID string `json:"first_liquidity_id"`
	Status           string `json:"status"`
	OrderID          string `json:"order_id"`
	OrderStatus      string `json:"order_status"`
}

type lifecycleSummary struct {
	ProofEnvironment proofEnvironmentSummary `json:"proof_environment"`
	MarketID         int64                   `json:"market_id"`
	TradeID          string                  `json:"trade_id"`
	Buyer            struct {
		UserID          int64  `json:"user_id"`
		WalletAddress   string `json:"wallet_address"`
		InitialUSDT     int64  `json:"initial_usdt"`
		PostDepositUSDT int64  `json:"post_deposit_usdt"`
		FinalUSDT       int64  `json:"final_usdt"`
		PayoutAmount    int64  `json:"payout_amount"`
	} `json:"buyer"`
	Maker struct {
		UserID           int64  `json:"user_id"`
		WalletAddress    string `json:"wallet_address"`
		FinalUSDT        int64  `json:"final_usdt"`
		FirstLiquidityID string `json:"first_liquidity_id"`
	} `json:"maker"`
	DepositID            string `json:"deposit_id"`
	DepositTxHash        string `json:"deposit_tx_hash"`
	DepositLogIndex      int64  `json:"deposit_log_index"`
	DepositBlockNumber   int64  `json:"deposit_block_number"`
	DepositVault         string `json:"deposit_vault_address"`
	DepositStatus        string `json:"deposit_status"`
	BootstrapOrderID     string `json:"bootstrap_order_id"`
	BootstrapOrderStatus string `json:"bootstrap_order_status"`
	BuyOrderID           string `json:"buy_order_id"`
	MarketStatus         string `json:"market_status"`
	ResolvedOutcome      string `json:"resolved_outcome"`
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := config.Load("chain")
	apiCfg := config.Load("api")

	flow := flag.String("flow", "trading-key-oracle", "lifecycle flow to run: trading-key-oracle (default) or legacy")
	baseURLFlag := flag.String("base-url", httpBaseURL(apiCfg.HTTPAddr), "API base URL")
	depositAmount := flag.Int64("deposit-amount", 50000000, "listener-driven deposit amount in chain base units (for example 50000000 = 50.00 USDT)")
	price := flag.Int64("price", 58, "limit price in cents")
	quantity := flag.Int64("quantity", 40, "trade quantity")
	timeout := flag.Duration("timeout", 5*time.Minute, "overall lifecycle timeout")
	flag.Parse()

	if *depositAmount <= 0 || *price <= 0 || *quantity <= 0 {
		log.Fatal("deposit-amount, price, and quantity must be positive")
	}

	opts := lifecycleOptions{
		BaseURL:       strings.TrimRight(*baseURLFlag, "/"),
		DepositAmount: *depositAmount,
		Price:         *price,
		Quantity:      *quantity,
		Timeout:       *timeout,
	}

	buyer := mustWalletFromEnv("buyer", 1001, "FUNNYOPTION_LOCAL_CHAIN_BUYER_PRIVATE_KEY", "59c6995e998f97a5a004497e5daef0d4f7dcd0cfd5401397dbeed52b21965b1d")
	maker := mustWalletFromEnv("maker", 1002, "FUNNYOPTION_LOCAL_CHAIN_MAKER_PRIVATE_KEY", "8b3a350cf5c34c9194ca85829f093d784c2f2c6c3a0eb1f3f3f94a639a6a39d1")
	operator := mustOperatorWallet(cfg)

	if strings.EqualFold(strings.TrimSpace(*flow), "trading-key-oracle") {
		if err := runTradingKeyOracleLifecycle(opts, logger, cfg, apiCfg, buyer, maker, operator); err != nil {
			log.Fatal(err)
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	depositEnv, err := buildDepositEnvironment(ctx, cfg, buyer)
	if err != nil {
		log.Fatalf("setup deposit environment: %v", err)
	}
	defer depositEnv.Close()

	client := &apiClient{
		baseURL: opts.BaseURL,
		client:  &http.Client{Timeout: 5 * time.Second},
	}

	if err := client.ping(ctx); err != nil {
		log.Fatalf("local API is not reachable: %v", err)
	}

	now := time.Now()
	market, err := client.createMarket(ctx, operator, dto.CreateMarketRequest{
		Title:           fmt.Sprintf("Local lifecycle proof %d", now.Unix()),
		Description:     "Admin-created market used by cmd/local-lifecycle to verify the local off-chain path.",
		CollateralAsset: "USDT",
		Status:          "OPEN",
		OpenAt:          now.Add(-5 * time.Minute).Unix(),
		CloseAt:         now.Add(30 * time.Minute).Unix(),
		ResolveAt:       now.Add(35 * time.Minute).Unix(),
		CreatedBy:       9001,
		Metadata: mustJSONRaw(map[string]any{
			"category":   "Local QA",
			"yesOdds":    0.58,
			"noOdds":     0.42,
			"sourceKind": "local-lifecycle",
		}),
	})
	if err != nil {
		log.Fatalf("create market: %v", err)
	}
	log.Printf("created market #%d", market.MarketID)

	buyerSession, err := client.createSession(ctx, buyer, depositEnv.depositChainID())
	if err != nil {
		log.Fatalf("create buyer session: %v", err)
	}
	makerSession, err := client.createSession(ctx, maker, depositEnv.depositChainID())
	if err != nil {
		log.Fatalf("create maker session: %v", err)
	}
	log.Printf("created sessions buyer=%s maker=%s", buyerSession.SessionID, makerSession.SessionID)

	if proofEnv, ok := depositEnv.(*listenerProofEnvironment); ok {
		dbConn, err := shareddb.OpenPostgres(ctx, cfg.PostgresDSN)
		if err != nil {
			log.Fatalf("open postgres: %v", err)
		}
		defer dbConn.Close()

		accountRPC, err := accountclient.NewGRPCClient(cfg.AccountGRPCAddr)
		if err != nil {
			log.Fatalf("open account grpc: %v", err)
		}
		defer accountRPC.Close()

		publisher := sharedkafka.NewJSONPublisher(logger, cfg.KafkaBrokers)
		defer publisher.Close()

		store := chainservice.NewSQLStore(dbConn)
		processor := chainservice.NewProcessor(logger, store, accountRPC, publisher, cfg.KafkaTopics)

		listenerCfg := proofEnv.listenerConfig(cfg)
		listener, err := chainservice.NewDepositListenerWithReader(logger, listenerCfg, store, processor, proofEnv.logReader())
		if err != nil {
			log.Fatalf("bootstrap deposit listener proof: %v", err)
		}
		listenerCtx, stopListener := context.WithCancel(ctx)
		defer stopListener()
		go listener.Start(listenerCtx)
	}

	initialBuyerUSDT, err := client.fetchUSDTBalance(ctx, buyer.UserID)
	if err != nil {
		log.Fatalf("fetch initial buyer balance: %v", err)
	}

	depositTxHash, err := depositEnv.submitDeposit(ctx, buyer, *depositAmount)
	if err != nil {
		log.Fatalf("submit wallet deposit: %v", err)
	}
	log.Printf(
		"submitted wallet deposit tx=%s vault=%s chain_id=%d",
		depositTxHash,
		depositEnv.summary().VaultAddress,
		depositEnv.depositChainID(),
	)
	var creditedDeposit depositResponse
	if err := waitFor(ctx, 500*time.Millisecond, func() (bool, error) {
		items, err := client.listDeposits(ctx, buyer.UserID)
		if err != nil {
			return false, err
		}
		for _, item := range items {
			if strings.EqualFold(item.TxHash, normalizeLifecycleTxHash(depositTxHash)) && item.CreditedAt > 0 {
				creditedDeposit = item
				return true, nil
			}
		}
		return false, nil
	}); err != nil {
		log.Fatalf("wait for deposit read: %v", err)
	}

	postDepositBuyerUSDT, err := client.fetchUSDTBalance(ctx, buyer.UserID)
	if err != nil {
		log.Fatalf("fetch buyer balance after deposit: %v", err)
	}
	log.Printf(
		"buyer USDT %d -> %d after listener-driven deposit %s",
		initialBuyerUSDT,
		postDepositBuyerUSDT,
		creditedDeposit.DepositID,
	)

	firstLiquidity, err := client.createFirstLiquidity(ctx, operator, market.MarketID, maker.UserID, *quantity, "YES", *price)
	if err != nil {
		log.Fatalf("issue first-liquidity inventory: %v", err)
	}
	if err := waitFor(ctx, 500*time.Millisecond, func() (bool, error) {
		items, err := client.listPositions(ctx, maker.UserID, market.MarketID)
		if err != nil {
			return false, err
		}
		var yesReady bool
		var noReady bool
		for _, item := range items {
			if item.MarketID != market.MarketID {
				continue
			}
			switch item.Outcome {
			case "YES":
				yesReady = item.Quantity >= *quantity
			case "NO":
				noReady = item.Quantity >= *quantity
			}
		}
		return yesReady && noReady, nil
	}); err != nil {
		log.Fatalf("wait for explicit first-liquidity inventory: %v", err)
	}
	if strings.TrimSpace(firstLiquidity.OrderID) == "" {
		log.Fatalf("issue first-liquidity inventory: missing bootstrap order id in response")
	}
	log.Printf(
		"issued first-liquidity inventory %s and queued bootstrap sell %s for maker=%d",
		firstLiquidity.FirstLiquidityID,
		firstLiquidity.OrderID,
		maker.UserID,
	)

	var bootstrapOrder orderResponse
	if err := waitFor(ctx, 500*time.Millisecond, func() (bool, error) {
		items, err := client.listOrders(ctx, maker.UserID, market.MarketID)
		if err != nil {
			return false, err
		}
		for _, item := range items {
			if item.OrderID != firstLiquidity.OrderID {
				continue
			}
			bootstrapOrder = item
			return item.Side == "SELL" && item.RemainingQuantity >= *quantity, nil
		}
		return false, nil
	}); err != nil {
		log.Fatalf("wait for bootstrap sell order visibility: %v", err)
	}
	log.Printf("bootstrap sell order %s is visible with status=%s", bootstrapOrder.OrderID, bootstrapOrder.Status)

	buyResult, err := client.createSignedOrder(ctx, &buyerSession, market.MarketID, "YES", "BUY", *price, *quantity)
	if err != nil {
		log.Fatalf("create buy order: %v", err)
	}
	log.Printf("queued buy order %s", buyResult.OrderID)

	var matchedTrade tradeResponse
	if err := waitFor(ctx, 500*time.Millisecond, func() (bool, error) {
		items, err := client.listTrades(ctx, market.MarketID)
		if err != nil {
			return false, err
		}
		for _, item := range items {
			if item.MarketID == market.MarketID && item.Quantity == *quantity && item.Price == *price {
				matchedTrade = item
				return true, nil
			}
		}
		return false, nil
	}); err != nil {
		log.Fatalf("wait for matched trade: %v", err)
	}
	log.Printf("matched trade %s quantity=%d price=%d", matchedTrade.TradeID, matchedTrade.Quantity, matchedTrade.Price)

	if err := client.resolveMarket(ctx, operator, market.MarketID, "YES"); err != nil {
		log.Fatalf("resolve market: %v", err)
	}
	log.Printf("queued resolution for market #%d", market.MarketID)

	var finalMarket marketResponse
	if err := waitFor(ctx, 500*time.Millisecond, func() (bool, error) {
		item, err := client.getMarket(ctx, market.MarketID)
		if err != nil {
			return false, err
		}
		finalMarket = item
		return item.Status == "RESOLVED" && item.ResolvedOutcome == "YES", nil
	}); err != nil {
		log.Fatalf("wait for resolved market: %v", err)
	}

	var buyerPayout payoutResponse
	if err := waitFor(ctx, 500*time.Millisecond, func() (bool, error) {
		items, err := client.listPayouts(ctx, buyer.UserID, market.MarketID)
		if err != nil {
			return false, err
		}
		for _, item := range items {
			if item.MarketID == market.MarketID && item.PayoutAmount > 0 {
				buyerPayout = item
				return true, nil
			}
		}
		return false, nil
	}); err != nil {
		log.Fatalf("wait for buyer payout: %v", err)
	}

	finalBuyerUSDT, err := client.fetchUSDTBalance(ctx, buyer.UserID)
	if err != nil {
		log.Fatalf("fetch buyer final balance: %v", err)
	}
	finalMakerUSDT, err := client.fetchUSDTBalance(ctx, maker.UserID)
	if err != nil {
		log.Fatalf("fetch maker final balance: %v", err)
	}

	summary := lifecycleSummary{
		ProofEnvironment:     depositEnv.summary(),
		MarketID:             market.MarketID,
		TradeID:              matchedTrade.TradeID,
		DepositID:            creditedDeposit.DepositID,
		DepositTxHash:        depositTxHash,
		DepositLogIndex:      creditedDeposit.LogIndex,
		DepositBlockNumber:   creditedDeposit.BlockNumber,
		DepositVault:         creditedDeposit.VaultAddress,
		DepositStatus:        creditedDeposit.Status,
		BootstrapOrderID:     bootstrapOrder.OrderID,
		BootstrapOrderStatus: bootstrapOrder.Status,
		BuyOrderID:           buyResult.OrderID,
		MarketStatus:         finalMarket.Status,
		ResolvedOutcome:      finalMarket.ResolvedOutcome,
	}
	summary.Buyer.UserID = buyer.UserID
	summary.Buyer.WalletAddress = buyer.Address
	summary.Buyer.InitialUSDT = initialBuyerUSDT
	summary.Buyer.PostDepositUSDT = postDepositBuyerUSDT
	summary.Buyer.FinalUSDT = finalBuyerUSDT
	summary.Buyer.PayoutAmount = buyerPayout.PayoutAmount
	summary.Maker.UserID = maker.UserID
	summary.Maker.WalletAddress = maker.Address
	summary.Maker.FinalUSDT = finalMakerUSDT
	summary.Maker.FirstLiquidityID = firstLiquidity.FirstLiquidityID

	out, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		log.Fatalf("marshal summary: %v", err)
	}
	fmt.Println(string(out))
}

func mustWallet(label string, userID int64, privateKeyHex string) walletIdentity {
	key, err := crypto.HexToECDSA(strings.TrimPrefix(strings.TrimSpace(privateKeyHex), "0x"))
	if err != nil {
		log.Fatalf("invalid %s private key: %v", label, err)
	}
	return walletIdentity{
		Label:      label,
		UserID:     userID,
		PrivateKey: key,
		Address:    strings.ToLower(crypto.PubkeyToAddress(key.PublicKey).Hex()),
	}
}

func mustWalletFromEnv(label string, userID int64, envKey, fallbackPrivateKeyHex string) walletIdentity {
	privateKeyHex := strings.TrimSpace(os.Getenv(envKey))
	if privateKeyHex == "" {
		privateKeyHex = fallbackPrivateKeyHex
	}
	return mustWallet(label, userID, privateKeyHex)
}

func mustOperatorWallet(cfg config.ServiceConfig) walletIdentity {
	privateKeyHex := strings.TrimSpace(cfg.ChainOperatorPrivateKey)
	if privateKeyHex == "" {
		log.Fatal("FUNNYOPTION_CHAIN_OPERATOR_PRIVATE_KEY is required for cmd/local-lifecycle")
	}
	return mustWallet("operator", 0, privateKeyHex)
}

func mustJSONRaw(value any) json.RawMessage {
	payload, err := json.Marshal(value)
	if err != nil {
		log.Fatalf("marshal lifecycle metadata: %v", err)
	}
	return payload
}

func httpBaseURL(addr string) string {
	trimmed := strings.TrimSpace(addr)
	if trimmed == "" {
		return "http://127.0.0.1:8080"
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return trimmed
	}
	if strings.HasPrefix(trimmed, ":") {
		return "http://127.0.0.1" + trimmed
	}
	if strings.Contains(trimmed, "://") {
		return trimmed
	}
	return "http://" + trimmed
}

func normalizeLifecycleTxHash(txHash string) string {
	trimmed := strings.ToLower(strings.TrimSpace(txHash))
	return strings.TrimPrefix(trimmed, "0x")
}

func waitFor(ctx context.Context, interval time.Duration, fn func() (bool, error)) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		ok, err := fn()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (c *apiClient) ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/healthz", nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected healthz status %d", resp.StatusCode)
	}
	return nil
}

func (c *apiClient) createMarket(ctx context.Context, operator walletIdentity, req dto.CreateMarketRequest) (marketResponse, error) {
	requestedAt := time.Now().UnixMilli()
	req.Operator = &dto.OperatorAction{
		WalletAddress: operator.Address,
		RequestedAt:   requestedAt,
	}
	signature, err := signPersonalMessage(req.OperatorMessage(), operator.PrivateKey)
	if err != nil {
		return marketResponse{}, err
	}
	req.Operator.Signature = signature

	var result marketResponse
	return result, c.doJSON(ctx, http.MethodPost, "/api/v1/markets", req, &result)
}

func (c *apiClient) createSession(ctx context.Context, wallet walletIdentity, chainID int64) (sessionContext, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return sessionContext{}, err
	}
	now := time.Now()
	grant := sharedauth.SessionGrant{
		WalletAddress:    wallet.Address,
		SessionPublicKey: hexutil.Encode(pub),
		Scope:            sharedauth.DefaultSessionScope,
		ChainID:          chainID,
		Nonce:            fmt.Sprintf("sess_%d", now.UnixNano()),
		IssuedAtMillis:   now.UnixMilli(),
		ExpiresAtMillis:  now.Add(24 * time.Hour).UnixMilli(),
	}
	signature, err := signPersonalMessage(grant.Message(), wallet.PrivateKey)
	if err != nil {
		return sessionContext{}, err
	}

	var remote remoteSession
	err = c.doJSON(ctx, http.MethodPost, "/api/v1/sessions", map[string]any{
		"user_id":            wallet.UserID,
		"wallet_address":     wallet.Address,
		"session_public_key": grant.SessionPublicKey,
		"scope":              grant.Scope,
		"chain_id":           grant.ChainID,
		"nonce":              grant.Nonce,
		"issued_at":          grant.IssuedAtMillis,
		"expires_at":         grant.ExpiresAtMillis,
		"wallet_signature":   signature,
	}, &remote)
	if err != nil {
		return sessionContext{}, err
	}
	c.rememberSession(remote.UserID, remote.SessionID)

	return sessionContext{
		UserID:        remote.UserID,
		WalletAddress: strings.ToLower(remote.WalletAddress),
		SessionID:     remote.SessionID,
		SessionPubKey: remote.SessionPublicKey,
		SessionPriv:   priv,
		LastNonce:     remote.LastOrderNonce,
		ChainID:       remote.ChainID,
		VaultAddress:  strings.ToLower(strings.TrimSpace(remote.VaultAddress)),
	}, nil
}

func (c *apiClient) createSignedOrder(ctx context.Context, session *sessionContext, marketID int64, outcome, side string, price, quantity int64) (createOrderResult, error) {
	session.LastNonce++
	intent := sharedauth.OrderIntent{
		SessionID:         session.SessionID,
		WalletAddress:     session.WalletAddress,
		UserID:            session.UserID,
		MarketID:          marketID,
		Outcome:           strings.ToUpper(strings.TrimSpace(outcome)),
		Side:              strings.ToUpper(strings.TrimSpace(side)),
		OrderType:         "LIMIT",
		TimeInForce:       "GTC",
		Price:             price,
		Quantity:          quantity,
		ClientOrderID:     fmt.Sprintf("%s_%d", strings.ToLower(side), time.Now().UnixNano()),
		Nonce:             session.LastNonce,
		RequestedAtMillis: time.Now().UnixMilli(),
	}
	signature := hexutil.Encode(ed25519.Sign(session.SessionPriv, []byte(intent.Message())))

	var result createOrderResult
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/orders", map[string]any{
		"user_id":           session.UserID,
		"market_id":         marketID,
		"outcome":           strings.ToLower(intent.Outcome),
		"side":              strings.ToLower(intent.Side),
		"type":              "limit",
		"time_in_force":     "gtc",
		"price":             price,
		"quantity":          quantity,
		"client_order_id":   intent.ClientOrderID,
		"session_id":        session.SessionID,
		"session_signature": signature,
		"order_nonce":       session.LastNonce,
		"requested_at":      intent.RequestedAtMillis,
	}, &result)
	if err != nil {
		return createOrderResult{}, err
	}
	return result, nil
}

func (c *apiClient) createFirstLiquidity(ctx context.Context, operator walletIdentity, marketID, userID, quantity int64, outcome string, price int64) (createFirstLiquidityResult, error) {
	req := dto.CreateFirstLiquidityRequest{
		UserID:   userID,
		Quantity: quantity,
		Outcome:  outcome,
		Price:    price,
		Operator: &dto.OperatorAction{
			WalletAddress: operator.Address,
			RequestedAt:   time.Now().UnixMilli(),
		},
	}
	signature, err := signPersonalMessage(req.OperatorMessage(marketID), operator.PrivateKey)
	if err != nil {
		return createFirstLiquidityResult{}, err
	}
	req.Operator.Signature = signature

	var result createFirstLiquidityResult
	err = c.doJSON(ctx, http.MethodPost, fmt.Sprintf("/api/v1/admin/markets/%d/first-liquidity", marketID), req, &result)
	if err != nil {
		return createFirstLiquidityResult{}, err
	}
	return result, nil
}

func (c *apiClient) resolveMarket(ctx context.Context, operator walletIdentity, marketID int64, outcome string) error {
	req := dto.ResolveMarketRequest{
		Outcome: outcome,
		Operator: &dto.OperatorAction{
			WalletAddress: operator.Address,
			RequestedAt:   time.Now().UnixMilli(),
		},
	}
	signature, err := signPersonalMessage(req.OperatorMessage(marketID), operator.PrivateKey)
	if err != nil {
		return err
	}
	req.Operator.Signature = signature
	return c.doJSON(ctx, http.MethodPost, fmt.Sprintf("/api/v1/markets/%d/resolve", marketID), req, &map[string]any{})
}

func (c *apiClient) getMarket(ctx context.Context, marketID int64) (marketResponse, error) {
	var result marketResponse
	return result, c.doJSON(ctx, http.MethodGet, fmt.Sprintf("/api/v1/markets/%d", marketID), nil, &result)
}

func (c *apiClient) listTrades(ctx context.Context, marketID int64) ([]tradeResponse, error) {
	var result collectionResponse[tradeResponse]
	err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("/api/v1/trades?market_id=%d&limit=20", marketID), nil, &result)
	return result.Items, err
}

func (c *apiClient) listOrders(ctx context.Context, userID, marketID int64) ([]orderResponse, error) {
	var result collectionResponse[orderResponse]
	err := c.doJSONForUser(ctx, userID, http.MethodGet, fmt.Sprintf("/api/v1/orders?user_id=%d&market_id=%d&limit=20", userID, marketID), nil, &result)
	return result.Items, err
}

func (c *apiClient) listBalances(ctx context.Context, userID int64) ([]balanceResponse, error) {
	var result collectionResponse[balanceResponse]
	err := c.doJSONForUser(ctx, userID, http.MethodGet, fmt.Sprintf("/api/v1/balances?user_id=%d&limit=20", userID), nil, &result)
	return result.Items, err
}

func (c *apiClient) fetchUSDTBalance(ctx context.Context, userID int64) (int64, error) {
	items, err := c.listBalances(ctx, userID)
	if err != nil {
		return 0, err
	}
	for _, item := range items {
		if item.Asset == "USDT" {
			return item.Available, nil
		}
	}
	return 0, nil
}

func (c *apiClient) listPositions(ctx context.Context, userID, marketID int64) ([]positionResponse, error) {
	var result collectionResponse[positionResponse]
	err := c.doJSONForUser(ctx, userID, http.MethodGet, fmt.Sprintf("/api/v1/positions?user_id=%d&market_id=%d&limit=20", userID, marketID), nil, &result)
	return result.Items, err
}

func (c *apiClient) listDeposits(ctx context.Context, userID int64) ([]depositResponse, error) {
	var result collectionResponse[depositResponse]
	err := c.doJSONForUser(ctx, userID, http.MethodGet, fmt.Sprintf("/api/v1/deposits?user_id=%d&limit=20", userID), nil, &result)
	return result.Items, err
}

func (c *apiClient) listPayouts(ctx context.Context, userID, marketID int64) ([]payoutResponse, error) {
	var result collectionResponse[payoutResponse]
	err := c.doJSONForUser(ctx, userID, http.MethodGet, fmt.Sprintf("/api/v1/payouts?user_id=%d&market_id=%d&limit=20", userID, marketID), nil, &result)
	return result.Items, err
}

func (c *apiClient) listRollupForcedWithdrawals(ctx context.Context, walletAddress, status string, limit int) ([]rollupForcedWithdrawalResponse, error) {
	query := url.Values{}
	if strings.TrimSpace(walletAddress) != "" {
		query.Set("wallet_address", strings.ToLower(strings.TrimSpace(walletAddress)))
	}
	if strings.TrimSpace(status) != "" {
		query.Set("status", strings.ToUpper(strings.TrimSpace(status)))
	}
	if limit <= 0 {
		limit = 20
	}
	query.Set("limit", fmt.Sprintf("%d", limit))

	var result collectionResponse[rollupForcedWithdrawalResponse]
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/rollup/forced-withdrawals?"+query.Encode(), nil, &result)
	return result.Items, err
}

func (c *apiClient) listRollupEscapeCollateralClaims(ctx context.Context, userID int64, walletAddress, status string, limit int) ([]rollupEscapeCollateralClaimResponse, error) {
	query := url.Values{}
	if userID > 0 {
		query.Set("user_id", fmt.Sprintf("%d", userID))
	}
	if strings.TrimSpace(walletAddress) != "" {
		query.Set("wallet_address", strings.ToLower(strings.TrimSpace(walletAddress)))
	}
	if strings.TrimSpace(status) != "" {
		query.Set("status", strings.ToUpper(strings.TrimSpace(status)))
	}
	if limit <= 0 {
		limit = 20
	}
	query.Set("limit", fmt.Sprintf("%d", limit))

	var result collectionResponse[rollupEscapeCollateralClaimResponse]
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/rollup/escape-collateral?"+query.Encode(), nil, &result)
	return result.Items, err
}

func (c *apiClient) getRollupFreezeState(ctx context.Context) (rollupFreezeStateResponse, error) {
	var result rollupFreezeStateResponse
	return result, c.doJSON(ctx, http.MethodGet, "/api/v1/rollup/freeze-state", nil, &result)
}

func (c *apiClient) rememberSession(userID int64, sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if userID <= 0 || sessionID == "" {
		return
	}
	if c.sessionAuths == nil {
		c.sessionAuths = make(map[int64]string)
	}
	c.sessionAuths[userID] = sessionID
}

func (c *apiClient) doJSONForUser(ctx context.Context, userID int64, method, path string, body any, out any) error {
	sessionID := strings.TrimSpace(c.sessionAuths[userID])
	if sessionID == "" {
		return fmt.Errorf("%s %s: missing session authorization for user %d", method, path, userID)
	}
	return c.doJSONAuthorized(ctx, sessionID, method, path, body, out)
}

func (c *apiClient) doJSON(ctx context.Context, method, path string, body any, out any) error {
	return c.doJSONAuthorized(ctx, "", method, path, body, out)
}

func (c *apiClient) doJSONAuthorized(ctx context.Context, sessionID, method, path string, body any, out any) error {
	var reader *strings.Reader
	if body == nil {
		reader = strings.NewReader("")
	} else {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = strings.NewReader(string(payload))
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	if strings.TrimSpace(sessionID) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(sessionID))
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && strings.TrimSpace(apiErr.Error) != "" {
			return fmt.Errorf("%s %s: %s", method, path, apiErr.Error)
		}
		return fmt.Errorf("%s %s: http %d", method, path, resp.StatusCode)
	}

	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func signPersonalMessage(message string, key *ecdsa.PrivateKey) (string, error) {
	signature, err := crypto.Sign(accounts.TextHash([]byte(message)), key)
	if err != nil {
		return "", err
	}
	return hexutil.Encode(signature), nil
}
