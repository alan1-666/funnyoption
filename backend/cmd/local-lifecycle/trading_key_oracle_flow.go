package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"time"

	"funnyoption/internal/api/dto"
	chainservice "funnyoption/internal/chain/service"
	oracleservice "funnyoption/internal/oracle/service"
	"funnyoption/internal/shared/assets"
	sharedauth "funnyoption/internal/shared/auth"
	"funnyoption/internal/shared/config"
	shareddb "funnyoption/internal/shared/db"
	sharedkafka "funnyoption/internal/shared/kafka"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

const (
	tradingKeyOracleFlowName   = "trading-key-oracle"
	localOracleSymbol          = "BTCUSDT"
	localOracleObservedPrice   = "85000.25000000"
	localOracleThresholdPrice  = "84000.00000000"
	localOracleResolveDelay    = 20 * time.Second
	localOracleWorkerPoll      = 250 * time.Millisecond
	defaultSessionListReadback = 200
)

type lifecycleOptions struct {
	BaseURL       string
	DepositAmount int64
	Price         int64
	Quantity      int64
	Timeout       time.Duration
}

type tradingKeyChallengeResponse struct {
	ChallengeID        string `json:"challenge_id"`
	Challenge          string `json:"challenge"`
	ChallengeExpiresAt int64  `json:"challenge_expires_at"`
}

type flowStepResult struct {
	Step          string `json:"step"`
	Status        string `json:"status"`
	SignatureMode string `json:"signature_mode"`
	Notes         string `json:"notes"`
}

type localFullFlowSummary struct {
	Flow             string                  `json:"flow"`
	RunCommand       string                  `json:"run_command"`
	ProofEnvironment proofEnvironmentSummary `json:"proof_environment"`
	PassFailMatrix   []flowStepResult        `json:"pass_fail_matrix"`
	IDs              struct {
		BuyerChallengeID    string `json:"buyer_challenge_id"`
		BuyerTradingKeyID   string `json:"buyer_trading_key_id"`
		MakerChallengeID    string `json:"maker_challenge_id"`
		MakerTradingKeyID   string `json:"maker_trading_key_id"`
		DepositID           string `json:"deposit_id"`
		DepositTxHash       string `json:"deposit_tx_hash"`
		FirstLiquidityID    string `json:"first_liquidity_id"`
		BootstrapOrderID    string `json:"bootstrap_order_id"`
		BuyOrderID          string `json:"buy_order_id"`
		TradeID             string `json:"trade_id"`
		MarketID            int64  `json:"market_id"`
		PayoutEventID       string `json:"payout_event_id"`
		EscapeClaimID       string `json:"escape_claim_id"`
		ResolutionResolver  string `json:"resolution_resolver_ref"`
		OracleObservationID string `json:"oracle_observation_id"`
	} `json:"ids"`
	Buyer struct {
		UserID           int64  `json:"user_id"`
		WalletAddress    string `json:"wallet_address"`
		TradingPublicKey string `json:"trading_public_key"`
		InitialUSDT      int64  `json:"initial_usdt"`
		PostDepositUSDT  int64  `json:"post_deposit_usdt"`
		FinalUSDT        int64  `json:"final_usdt"`
		EscapedUSDT      int64  `json:"escaped_usdt"`
		PayoutAmount     int64  `json:"payout_amount"`
		SettledYesQty    int64  `json:"settled_yes_quantity"`
		RestoreSessionID string `json:"restore_session_id"`
		RestoreUserID    int64  `json:"restore_user_id"`
	} `json:"buyer"`
	Maker struct {
		UserID           int64  `json:"user_id"`
		WalletAddress    string `json:"wallet_address"`
		TradingPublicKey string `json:"trading_public_key"`
		FinalUSDT        int64  `json:"final_usdt"`
		BootstrapStatus  string `json:"bootstrap_order_status"`
		RestoreSessionID string `json:"restore_session_id"`
		RestoreUserID    int64  `json:"restore_user_id"`
	} `json:"maker"`
	Deposit struct {
		Status       string `json:"status"`
		VaultAddress string `json:"vault_address"`
		BlockNumber  int64  `json:"block_number"`
		LogIndex     int64  `json:"log_index"`
	} `json:"deposit"`
	Market struct {
		Status                 string `json:"status"`
		ResolvedOutcome        string `json:"resolved_outcome"`
		TradeCount             int64  `json:"trade_count"`
		ActiveOrderCount       int64  `json:"active_order_count"`
		ResolutionStatus       string `json:"resolution_status"`
		ResolutionResolverType string `json:"resolution_resolver_type"`
		OracleDispatchStatus   string `json:"oracle_dispatch_status"`
		OracleDispatchAttempts int    `json:"oracle_dispatch_attempts"`
	} `json:"market"`
	Oracle struct {
		ProviderKey    string `json:"provider_key"`
		Symbol         string `json:"symbol"`
		ThresholdPrice string `json:"threshold_price"`
		ObservedPrice  string `json:"observed_price"`
		FixtureBaseURL string `json:"fixture_base_url"`
	} `json:"oracle"`
	Rollup struct {
		BaselineAcceptedBatchID int64    `json:"baseline_accepted_batch_id"`
		AcceptedBatchCount      int64    `json:"accepted_batch_count"`
		LatestAcceptedBatchID   int64    `json:"latest_accepted_batch_id"`
		LatestAcceptedStateRoot string   `json:"latest_accepted_state_root"`
		LatestSubmissionStatus  string   `json:"latest_submission_status"`
		LatestEscapeBatchID     int64    `json:"latest_escape_batch_id"`
		LatestEscapeMerkleRoot  string   `json:"latest_escape_merkle_root"`
		ForcedRequestID         int64    `json:"forced_request_id"`
		ForcedRequestStatus     string   `json:"forced_request_status"`
		Frozen                  bool     `json:"frozen"`
		FrozenAt                int64    `json:"frozen_at"`
		EscapeClaimStatus       string   `json:"escape_claim_status"`
		SubmissionActions       []string `json:"submission_actions"`
	} `json:"rollup"`
	ReadbackCommands []string `json:"readback_commands"`
	BlindSpots       []string `json:"residual_blind_spots"`
}

type resolutionReadback struct {
	Status          string
	ResolvedOutcome string
	ResolverType    string
	ResolverRef     string
	Evidence        json.RawMessage
}

type localOracleHarness struct {
	cancel    context.CancelFunc
	fixture   *httptest.Server
	publisher sharedkafka.Publisher
	db        *sql.DB
}

type acceptedBatchReadback struct {
	AcceptedBatchCount      int64
	LatestAcceptedBatchID   int64
	LatestAcceptedStateRoot string
	LatestSubmissionStatus  string
}

func runTradingKeyOracleLifecycle(opts lifecycleOptions, logger *slog.Logger, cfg, apiCfg config.ServiceConfig, buyer, maker, operator walletIdentity) error {
	if !shouldUsePersistentLocalChain(cfg) {
		return fmt.Errorf("flow %q requires FUNNYOPTION_LOCAL_CHAIN_MODE=anvil with .run/dev/local-chain.env sourced so trading-key auth and deposits share one configured vault", tradingKeyOracleFlowName)
	}
	if strings.TrimSpace(cfg.RollupCoreAddress) == "" {
		return fmt.Errorf("flow %q requires FUNNYOPTION_ROLLUP_CORE_ADDRESS so the harness can drive rollup submission through FunnyRollupCore", tradingKeyOracleFlowName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	depositEnv, err := buildDepositEnvironment(ctx, cfg, buyer)
	if err != nil {
		return fmt.Errorf("setup deposit environment: %w", err)
	}
	defer depositEnv.Close()

	client := &apiClient{
		baseURL: opts.BaseURL,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
	if err := client.ping(ctx); err != nil {
		return fmt.Errorf("local API is not reachable: %w", err)
	}

	summary := localFullFlowSummary{
		Flow:             tradingKeyOracleFlowName,
		RunCommand:       "go run ./cmd/local-lifecycle --flow trading-key-oracle",
		ProofEnvironment: depositEnv.summary(),
		PassFailMatrix:   make([]flowStepResult, 0, 9),
		BlindSpots: []string{
			"trading-key wallet authorization is signed by local deterministic test EOAs, not a browser wallet popup or hardware wallet",
			"truthful restore is verified through in-process metadata plus GET /api/v1/trading-keys readback, not real localStorage/IndexedDB/React hydration behavior",
			"oracle settlement uses a local fake Binance HTTP fixture, not the live external provider network path",
			"the flow now drives local rollup acceptance, freeze, and escape-collateral claim execution, but it still does not cover canonical slow-withdraw claim processing",
		},
	}

	vaultAddr := os.Getenv("FUNNYOPTION_VAULT_ADDRESS")
	buyerSession, buyerChallenge, err := client.registerTradingKey(ctx, buyer, cfg.ChainID, vaultAddr)
	if err != nil {
		return fmt.Errorf("register buyer trading key: %w", err)
	}
	makerSession, makerChallenge, err := client.registerTradingKey(ctx, maker, cfg.ChainID, vaultAddr)
	if err != nil {
		return fmt.Errorf("register maker trading key: %w", err)
	}
	summary.PassFailMatrix = append(summary.PassFailMatrix, flowStepResult{
		Step:          "1. trading-key challenge + wallet authorization register",
		Status:        "PASS",
		SignatureMode: "real EIP-712 payload shape, signed by local deterministic test EOAs as wallet substitutes",
		Notes:         fmt.Sprintf("buyer=%s maker=%s vault=%s chain_id=%d", buyerSession.SessionID, makerSession.SessionID, vaultAddr, cfg.ChainID),
	})
	summary.IDs.BuyerChallengeID = buyerChallenge.ChallengeID
	summary.IDs.BuyerTradingKeyID = buyerSession.SessionID
	summary.IDs.MakerChallengeID = makerChallenge.ChallengeID
	summary.IDs.MakerTradingKeyID = makerSession.SessionID
	summary.Buyer.UserID = buyerSession.UserID
	summary.Buyer.WalletAddress = buyerSession.WalletAddress
	summary.Buyer.TradingPublicKey = buyerSession.SessionPubKey
	summary.Maker.UserID = makerSession.UserID
	summary.Maker.WalletAddress = makerSession.WalletAddress
	summary.Maker.TradingPublicKey = makerSession.SessionPubKey

	buyerRestore, err := client.verifyTruthfulRestore(ctx, buyerSession)
	if err != nil {
		return fmt.Errorf("verify buyer restore: %w", err)
	}
	makerRestore, err := client.verifyTruthfulRestore(ctx, makerSession)
	if err != nil {
		return fmt.Errorf("verify maker restore: %w", err)
	}
	summary.PassFailMatrix = append(summary.PassFailMatrix, flowStepResult{
		Step:          "2. truthful restore",
		Status:        "PASS",
		SignatureMode: "no new signature; in-process metadata is reconciled against GET /api/v1/trading-keys readback",
		Notes:         fmt.Sprintf("buyer_restore=%s maker_restore=%s", buyerRestore.SessionID, makerRestore.SessionID),
	})
	summary.Buyer.RestoreSessionID = buyerRestore.SessionID
	summary.Buyer.RestoreUserID = buyerRestore.UserID
	summary.Maker.RestoreSessionID = makerRestore.SessionID
	summary.Maker.RestoreUserID = makerRestore.UserID

	initialBuyerUSDT, err := client.fetchUSDTBalance(ctx, buyerSession.UserID)
	if err != nil {
		return fmt.Errorf("fetch initial buyer balance: %w", err)
	}
	makerBootstrapDepositAmount := opts.DepositAmount
	makerRequiredAccounting, err := assets.WinningPayoutAmount(opts.Quantity)
	if err != nil {
		return fmt.Errorf("calculate maker first-liquidity collateral: %w", err)
	}
	makerRequiredChainAmount, err := assets.AccountingToAssetChainAmount(assets.DefaultCollateralAsset, makerRequiredAccounting)
	if err != nil {
		return fmt.Errorf("convert maker first-liquidity collateral to chain units: %w", err)
	}
	if makerBootstrapDepositAmount < makerRequiredChainAmount {
		makerBootstrapDepositAmount = makerRequiredChainAmount
	}
	depositTxHash, err := depositEnv.submitDeposit(ctx, buyer, opts.DepositAmount)
	if err != nil {
		return fmt.Errorf("submit wallet deposit: %w", err)
	}
	var creditedDeposit depositResponse
	if err := waitFor(ctx, 500*time.Millisecond, func() (bool, error) {
		items, err := client.listDeposits(ctx, buyerSession.UserID)
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
		return fmt.Errorf("wait for credited deposit: %w", err)
	}
	postDepositBuyerUSDT, err := client.fetchUSDTBalance(ctx, buyerSession.UserID)
	if err != nil {
		return fmt.Errorf("fetch buyer balance after deposit: %w", err)
	}
	makerDepositTxHash, err := depositEnv.submitDeposit(ctx, maker, makerBootstrapDepositAmount)
	if err != nil {
		return fmt.Errorf("submit maker wallet deposit: %w", err)
	}
	var creditedMakerDeposit depositResponse
	if err := waitFor(ctx, 500*time.Millisecond, func() (bool, error) {
		items, err := client.listDeposits(ctx, makerSession.UserID)
		if err != nil {
			return false, err
		}
		for _, item := range items {
			if strings.EqualFold(item.TxHash, normalizeLifecycleTxHash(makerDepositTxHash)) && item.CreditedAt > 0 {
				creditedMakerDeposit = item
				return true, nil
			}
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("wait for maker credited deposit: %w", err)
	}
	summary.PassFailMatrix = append(summary.PassFailMatrix, flowStepResult{
		Step:          "3. deposit credit",
		Status:        "PASS",
		SignatureMode: "real EVM approve + deposit transactions signed by the buyer and maker test EOAs on the persistent local chain",
		Notes:         fmt.Sprintf("buyer_deposit_id=%s buyer_tx=%s maker_deposit_id=%s maker_tx=%s", creditedDeposit.DepositID, depositTxHash, creditedMakerDeposit.DepositID, makerDepositTxHash),
	})
	summary.IDs.DepositID = creditedDeposit.DepositID
	summary.IDs.DepositTxHash = depositTxHash
	summary.Buyer.InitialUSDT = initialBuyerUSDT
	summary.Buyer.PostDepositUSDT = postDepositBuyerUSDT
	summary.Deposit.Status = creditedDeposit.Status
	summary.Deposit.VaultAddress = creditedDeposit.VaultAddress
	summary.Deposit.BlockNumber = creditedDeposit.BlockNumber
	summary.Deposit.LogIndex = creditedDeposit.LogIndex

	oracleHarness, err := startLocalOracleHarness(ctx, logger, apiCfg)
	if err != nil {
		return fmt.Errorf("start local oracle harness: %w", err)
	}
	defer oracleHarness.Close()
	baselineAccepted, err := fetchAcceptedBatchReadback(ctx, oracleHarness.db)
	if err != nil {
		return fmt.Errorf("fetch baseline accepted batch readback: %w", err)
	}
	summary.Rollup.BaselineAcceptedBatchID = baselineAccepted.LatestAcceptedBatchID
	summary.Oracle.ProviderKey = oracleservice.OracleProviderKeyBinance
	summary.Oracle.Symbol = localOracleSymbol
	summary.Oracle.ThresholdPrice = localOracleThresholdPrice
	summary.Oracle.ObservedPrice = localOracleObservedPrice
	summary.Oracle.FixtureBaseURL = oracleHarness.fixture.URL

	now := time.Now()
	resolveAt := now.Add(localOracleResolveDelay).Unix()
	market, err := client.createMarket(ctx, operator, dto.CreateMarketRequest{
		Title:           fmt.Sprintf("Local full-flow harness %d", now.Unix()),
		Description:     "Admin-created oracle market used by cmd/local-lifecycle to verify trading-key auth through oracle auto settlement.",
		CategoryKey:     "CRYPTO",
		CollateralAsset: "USDT",
		Status:          "OPEN",
		OpenAt:          now.Add(-5 * time.Minute).Unix(),
		CloseAt:         resolveAt,
		ResolveAt:       resolveAt,
		CreatedBy:       9001,
		Metadata: mustJSONRaw(map[string]any{
			"category":   "Crypto",
			"sourceKind": "local-full-flow",
			"resolution": map[string]any{
				"version":                 1,
				"mode":                    oracleservice.ResolutionModeOraclePrice,
				"market_kind":             oracleservice.ResolutionMarketKindCryptoPrice,
				"manual_fallback_allowed": true,
				"oracle": map[string]any{
					"source_kind":  oracleservice.OracleSourceKindHTTPJSON,
					"provider_key": oracleservice.OracleProviderKeyBinance,
					"instrument": map[string]any{
						"kind":        oracleservice.OracleInstrumentKindSpot,
						"base_asset":  "BTC",
						"quote_asset": "USDT",
						"symbol":      localOracleSymbol,
					},
					"price": map[string]any{
						"field":            oracleservice.OraclePriceFieldLastPrice,
						"scale":            8,
						"rounding_mode":    oracleservice.OracleRoundingModeRoundHalfUp,
						"max_data_age_sec": 120,
					},
					"window": map[string]any{
						"anchor":     oracleservice.OracleWindowAnchorResolveAt,
						"before_sec": 300,
						"after_sec":  300,
					},
					"rule": map[string]any{
						"type":            oracleservice.OracleRuleTypePriceThreshold,
						"comparator":      "GTE",
						"threshold_price": localOracleThresholdPrice,
					},
				},
			},
		}),
	})
	if err != nil {
		return fmt.Errorf("create oracle market: %w", err)
	}
	summary.PassFailMatrix = append(summary.PassFailMatrix, flowStepResult{
		Step:          "4. oracle crypto market create",
		Status:        "PASS",
		SignatureMode: "operator create payload is signed by the local deterministic operator EOA as a test-wallet substitute",
		Notes:         fmt.Sprintf("market_id=%d resolve_at=%d", market.MarketID, resolveAt),
	})
	summary.IDs.MarketID = market.MarketID

	firstLiquidity, err := client.createFirstLiquidity(ctx, operator, market.MarketID, makerSession.UserID, opts.Quantity, "YES", opts.Price)
	if err != nil {
		return fmt.Errorf("issue first-liquidity inventory: %w", err)
	}
	if err := waitFor(ctx, 500*time.Millisecond, func() (bool, error) {
		items, err := client.listPositions(ctx, makerSession.UserID, market.MarketID)
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
				yesReady = item.Quantity >= opts.Quantity
			case "NO":
				noReady = item.Quantity >= opts.Quantity
			}
		}
		return yesReady && noReady, nil
	}); err != nil {
		return fmt.Errorf("wait for first-liquidity positions: %w", err)
	}
	if strings.TrimSpace(firstLiquidity.OrderID) == "" {
		return fmt.Errorf("first-liquidity response missing bootstrap order id")
	}

	var bootstrapOrder orderResponse
	if err := waitFor(ctx, 500*time.Millisecond, func() (bool, error) {
		items, err := client.listOrders(ctx, makerSession.UserID, market.MarketID)
		if err != nil {
			return false, err
		}
		for _, item := range items {
			if item.OrderID != firstLiquidity.OrderID {
				continue
			}
			bootstrapOrder = item
			return item.Side == "SELL" && item.RemainingQuantity >= opts.Quantity, nil
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("wait for bootstrap sell order visibility: %w", err)
	}

	buyResult, err := client.createSignedOrder(ctx, &buyerSession, market.MarketID, "YES", "BUY", opts.Price, opts.Quantity)
	if err != nil {
		return fmt.Errorf("create buy order: %w", err)
	}
	var matchedTrade tradeResponse
	if err := waitFor(ctx, 500*time.Millisecond, func() (bool, error) {
		items, err := client.listTrades(ctx, market.MarketID)
		if err != nil {
			return false, err
		}
		for _, item := range items {
			if item.MarketID == market.MarketID && item.Quantity == opts.Quantity && item.Price == opts.Price {
				matchedTrade = item
				return true, nil
			}
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("wait for matched trade: %w", err)
	}

	var filledBuyerOrder orderResponse
	if err := waitFor(ctx, 500*time.Millisecond, func() (bool, error) {
		items, err := client.listOrders(ctx, buyerSession.UserID, market.MarketID)
		if err != nil {
			return false, err
		}
		for _, item := range items {
			if item.OrderID == buyResult.OrderID {
				filledBuyerOrder = item
				return item.Status == "FILLED" && item.FilledQuantity >= opts.Quantity, nil
			}
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("wait for buyer order fill: %w", err)
	}

	if err := waitFor(ctx, 500*time.Millisecond, func() (bool, error) {
		items, err := client.listOrders(ctx, makerSession.UserID, market.MarketID)
		if err != nil {
			return false, err
		}
		for _, item := range items {
			if item.OrderID == bootstrapOrder.OrderID {
				bootstrapOrder = item
				return item.Status == "FILLED" && item.FilledQuantity >= opts.Quantity, nil
			}
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("wait for bootstrap order fill: %w", err)
	}
	_ = filledBuyerOrder

	summary.PassFailMatrix = append(summary.PassFailMatrix, flowStepResult{
		Step:          "5. order placement + matching",
		Status:        "PASS",
		SignatureMode: "bootstrap operator action uses the operator test wallet; taker order uses ED25519 trading-key signatures generated inside the harness process",
		Notes:         fmt.Sprintf("bootstrap=%s buy=%s trade=%s", bootstrapOrder.OrderID, buyResult.OrderID, matchedTrade.TradeID),
	})
	summary.IDs.FirstLiquidityID = firstLiquidity.FirstLiquidityID
	summary.IDs.BootstrapOrderID = bootstrapOrder.OrderID
	summary.IDs.BuyOrderID = buyResult.OrderID
	summary.IDs.TradeID = matchedTrade.TradeID

	var finalMarket marketResponse
	if err := waitFor(ctx, 500*time.Millisecond, func() (bool, error) {
		item, err := client.getMarket(ctx, market.MarketID)
		if err != nil {
			return false, err
		}
		finalMarket = item
		return item.Status == "RESOLVED" && item.ResolvedOutcome == "YES", nil
	}); err != nil {
		return fmt.Errorf("wait for oracle-resolved market: %w", err)
	}

	resolution, err := fetchResolutionReadback(ctx, oracleHarness.db, market.MarketID)
	if err != nil {
		return fmt.Errorf("fetch oracle resolution readback: %w", err)
	}
	dispatchStatus, dispatchAttempts, observationID := decodeOracleDispatch(resolution.Evidence)
	if strings.ToUpper(resolution.Status) != "RESOLVED" || strings.ToUpper(resolution.ResolverType) != oracleservice.ResolverTypeOraclePrice {
		return fmt.Errorf("unexpected resolution row status=%s resolver_type=%s", resolution.Status, resolution.ResolverType)
	}
	summary.PassFailMatrix = append(summary.PassFailMatrix, flowStepResult{
		Step:          "6. oracle auto settlement",
		Status:        "PASS",
		SignatureMode: "no wallet signature; a local fake Binance HTTP fixture drives the real oracle worker and settlement path",
		Notes:         fmt.Sprintf("resolver_ref=%s dispatch=%s", resolution.ResolverRef, dispatchStatus),
	})
	summary.IDs.ResolutionResolver = resolution.ResolverRef
	summary.IDs.OracleObservationID = observationID

	var buyerPayout payoutResponse
	var buyerPosition positionResponse
	if err := waitFor(ctx, 500*time.Millisecond, func() (bool, error) {
		payouts, err := client.listPayouts(ctx, buyerSession.UserID, market.MarketID)
		if err != nil {
			return false, err
		}
		for _, item := range payouts {
			if item.MarketID == market.MarketID && item.PayoutAmount > 0 {
				buyerPayout = item
				break
			}
		}
		if buyerPayout.EventID == "" {
			return false, nil
		}
		positions, err := client.listPositions(ctx, buyerSession.UserID, market.MarketID)
		if err != nil {
			return false, err
		}
		for _, item := range positions {
			if item.MarketID == market.MarketID && item.Outcome == "YES" {
				buyerPosition = item
				return item.SettledQuantity >= opts.Quantity, nil
			}
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("wait for payout and position readback: %w", err)
	}

	finalBuyerUSDT, err := client.fetchUSDTBalance(ctx, buyerSession.UserID)
	if err != nil {
		return fmt.Errorf("fetch buyer final balance: %w", err)
	}
	finalMakerUSDT, err := client.fetchUSDTBalance(ctx, makerSession.UserID)
	if err != nil {
		return fmt.Errorf("fetch maker final balance: %w", err)
	}
	depositAccountingAmount, err := assets.ChainToAssetAccountingAmount(assets.DefaultCollateralAsset, opts.DepositAmount)
	if err != nil {
		return fmt.Errorf("convert buyer deposit amount to accounting units: %w", err)
	}
	expectedAcceptedBuyerUSDT := depositAccountingAmount - opts.Price*opts.Quantity + buyerPayout.PayoutAmount
	summary.PassFailMatrix = append(summary.PassFailMatrix, flowStepResult{
		Step:          "7. payout/readback verification",
		Status:        "PASS",
		SignatureMode: "no signature; verification is done through API readbacks plus direct market_resolutions SQL readback",
		Notes:         fmt.Sprintf("payout=%s settled_yes=%d", buyerPayout.EventID, buyerPosition.SettledQuantity),
	})

	acceptedAfter, err := waitForAcceptedBatchAdvance(ctx, oracleHarness.db, baselineAccepted.LatestAcceptedBatchID)
	if err != nil {
		return fmt.Errorf("wait for accepted batch advance: %w", err)
	}

	var acceptedPayout payoutResponse
	var acceptedPosition positionResponse
	acceptedBuyerUSDT := int64(0)
	if err := waitFor(ctx, 500*time.Millisecond, func() (bool, error) {
		acceptedBuyerUSDT, err = client.fetchUSDTBalance(ctx, buyerSession.UserID)
		if err != nil {
			return false, err
		}
		acceptedPayouts, err := client.listPayouts(ctx, buyerSession.UserID, market.MarketID)
		if err != nil {
			return false, err
		}
		acceptedPayout = payoutResponse{}
		for _, item := range acceptedPayouts {
			if item.EventID == buyerPayout.EventID {
				acceptedPayout = item
				break
			}
		}
		if acceptedPayout.EventID == "" {
			return false, nil
		}
		acceptedPositions, err := client.listPositions(ctx, buyerSession.UserID, market.MarketID)
		if err != nil {
			return false, err
		}
		acceptedPosition = positionResponse{}
		for _, item := range acceptedPositions {
			if item.MarketID == market.MarketID && item.Outcome == "YES" {
				acceptedPosition = item
				break
			}
		}
		if acceptedPosition.MarketID == 0 {
			return false, nil
		}
		if acceptedPosition.SettledQuantity < opts.Quantity {
			return false, nil
		}
		return acceptedBuyerUSDT == expectedAcceptedBuyerUSDT, nil
	}); err != nil {
		return fmt.Errorf("wait for accepted payout/position/balance readback: %w", err)
	}
	if acceptedBuyerUSDT != expectedAcceptedBuyerUSDT {
		return fmt.Errorf("accepted buyer USDT = %d, want %d", acceptedBuyerUSDT, expectedAcceptedBuyerUSDT)
	}
	latestAcceptedAfterReadback, err := fetchAcceptedBatchReadback(ctx, oracleHarness.db)
	if err != nil {
		return fmt.Errorf("refresh accepted batch readback: %w", err)
	}
	acceptedAfter = latestAcceptedAfterReadback
	summary.PassFailMatrix = append(summary.PassFailMatrix, flowStepResult{
		Step:          "8. rollup acceptance + accepted readback",
		Status:        "PASS",
		SignatureMode: "no end-user signature; the background chain submitter records, publishes, and accepts the batch on FunnyRollupCore and the harness re-reads accepted mirrors",
		Notes:         fmt.Sprintf("latest_submission=%s latest_accepted_batch=%d accepted_usdt=%d", acceptedAfter.LatestSubmissionStatus, acceptedAfter.LatestAcceptedBatchID, acceptedBuyerUSDT),
	})

	summary.IDs.PayoutEventID = buyerPayout.EventID
	summary.Buyer.FinalUSDT = finalBuyerUSDT
	summary.Buyer.PayoutAmount = buyerPayout.PayoutAmount
	summary.Buyer.SettledYesQty = buyerPosition.SettledQuantity
	summary.Maker.FinalUSDT = finalMakerUSDT
	summary.Maker.BootstrapStatus = bootstrapOrder.Status
	summary.Market.Status = finalMarket.Status
	summary.Market.ResolvedOutcome = finalMarket.ResolvedOutcome
	summary.Market.TradeCount = finalMarket.Runtime.TradeCount
	summary.Market.ActiveOrderCount = finalMarket.Runtime.ActiveOrderCount
	summary.Market.ResolutionStatus = resolution.Status
	summary.Market.ResolutionResolverType = resolution.ResolverType
	summary.Market.OracleDispatchStatus = dispatchStatus
	summary.Market.OracleDispatchAttempts = dispatchAttempts
	summary.Rollup.AcceptedBatchCount = acceptedAfter.AcceptedBatchCount
	summary.Rollup.LatestAcceptedBatchID = acceptedAfter.LatestAcceptedBatchID
	summary.Rollup.LatestAcceptedStateRoot = acceptedAfter.LatestAcceptedStateRoot
	summary.Rollup.LatestSubmissionStatus = acceptedAfter.LatestSubmissionStatus
	summary.Rollup.SubmissionActions = []string{acceptedAfter.LatestSubmissionStatus}

	var anchoredEscapeClaim rollupEscapeCollateralClaimResponse
	if err := waitFor(ctx, 500*time.Millisecond, func() (bool, error) {
		items, err := client.listRollupEscapeCollateralClaims(ctx, buyerSession.UserID, buyerSession.WalletAddress, "", 20)
		if err != nil {
			return false, err
		}
		anchoredEscapeClaim = rollupEscapeCollateralClaimResponse{}
		for _, item := range items {
			if item.AccountID == buyerSession.UserID && strings.EqualFold(item.WalletAddress, buyerSession.WalletAddress) {
				if item.AnchorStatus == "ANCHORED" && item.ClaimAmount > 0 {
					if anchoredEscapeClaim.ClaimID == "" || item.BatchID > anchoredEscapeClaim.BatchID {
						anchoredEscapeClaim = item
					}
				}
			}
		}
		if anchoredEscapeClaim.ClaimID != "" {
			if acceptedAfter.LatestAcceptedBatchID > 0 && anchoredEscapeClaim.BatchID < acceptedAfter.LatestAcceptedBatchID {
				return false, nil
			}
			if anchoredEscapeClaim.BatchID > 0 {
				return true, nil
			}
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("wait for anchored escape collateral root: %w", err)
	}

	forcedWithdrawalAccounting, err := assets.ChainToAssetAccountingAmount(assets.DefaultCollateralAsset, anchoredEscapeClaim.ClaimAmount)
	if err != nil {
		return fmt.Errorf("convert escape collateral claim amount to accounting units: %w", err)
	}
	buyerWalletPrivateKey := hexutil.Encode(crypto.FromECDSA(buyer.PrivateKey))
	requestProgress, err := chainservice.RunRequestForcedWithdrawalOnce(
		ctx,
		logger.With("component", "local-full-flow-forced-withdraw"),
		cfg,
		buyerWalletPrivateKey,
		forcedWithdrawalAccounting,
		buyer.Address,
	)
	if err != nil {
		return fmt.Errorf("request forced withdrawal: %w", err)
	}

	var forcedWithdrawal rollupForcedWithdrawalResponse
	if err := waitFor(ctx, 500*time.Millisecond, func() (bool, error) {
		items, err := client.listRollupForcedWithdrawals(ctx, buyerSession.WalletAddress, "", 20)
		if err != nil {
			return false, err
		}
		for _, item := range items {
			if item.RequestID == int64(requestProgress.RequestID) {
				forcedWithdrawal = item
				return item.Status == "REQUESTED" && item.DeadlineAt > 0, nil
			}
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("wait for forced-withdrawal mirror readback: %w", err)
	}

	if err := waitFor(ctx, 250*time.Millisecond, func() (bool, error) {
		return time.Now().Unix() > forcedWithdrawal.DeadlineAt, nil
	}); err != nil {
		return fmt.Errorf("wait for forced-withdrawal deadline: %w", err)
	}

	freezeProgress, err := chainservice.RunFreezeForcedWithdrawalOnce(
		ctx,
		logger.With("component", "local-full-flow-freeze"),
		cfg,
		uint64(forcedWithdrawal.RequestID),
	)
	if err != nil {
		return fmt.Errorf("freeze missed forced withdrawal: %w", err)
	}

	var freezeState rollupFreezeStateResponse
	if err := waitFor(ctx, 500*time.Millisecond, func() (bool, error) {
		item, err := client.getRollupFreezeState(ctx)
		if err != nil {
			return false, err
		}
		freezeState = item
		return item.Frozen && item.RequestID == forcedWithdrawal.RequestID, nil
	}); err != nil {
		return fmt.Errorf("wait for frozen readback: %w", err)
	}

	if err := waitFor(ctx, 500*time.Millisecond, func() (bool, error) {
		items, err := client.listRollupForcedWithdrawals(ctx, buyerSession.WalletAddress, "", 20)
		if err != nil {
			return false, err
		}
		for _, item := range items {
			if item.RequestID == forcedWithdrawal.RequestID {
				forcedWithdrawal = item
				return item.Status == "FROZEN", nil
			}
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("wait for forced-withdrawal frozen status: %w", err)
	}

	escapeProgress, err := chainservice.RunClaimEscapeCollateralOnce(
		ctx,
		logger.With("component", "local-full-flow-escape-claim"),
		cfg,
		buyerWalletPrivateKey,
		buyerSession.UserID,
		anchoredEscapeClaim.ClaimID,
		buyer.Address,
	)
	if err != nil {
		return fmt.Errorf("claim escape collateral: %w", err)
	}
	if escapeProgress.Action == "ESCAPE_COLLATERAL_CLAIM_FAILED" {
		return fmt.Errorf("claim escape collateral: %s", strings.TrimSpace(escapeProgress.Note))
	}

	var claimedEscapeClaim rollupEscapeCollateralClaimResponse
	postEscapeBuyerUSDT := int64(-1)
	if err := waitFor(ctx, 500*time.Millisecond, func() (bool, error) {
		items, err := client.listRollupEscapeCollateralClaims(ctx, buyerSession.UserID, buyerSession.WalletAddress, "", 20)
		if err != nil {
			return false, err
		}
		for _, item := range items {
			if item.ClaimID == anchoredEscapeClaim.ClaimID {
				claimedEscapeClaim = item
				break
			}
		}
		if claimedEscapeClaim.ClaimID == "" || claimedEscapeClaim.ClaimStatus != "CLAIMED" {
			return false, nil
		}
		postEscapeBuyerUSDT, err = client.fetchUSDTBalance(ctx, buyerSession.UserID)
		if err != nil {
			return false, err
		}
		return postEscapeBuyerUSDT == 0, nil
	}); err != nil {
		return fmt.Errorf("wait for escape collateral claim readback: %w", err)
	}
	escapedBuyerAccounting, err := assets.ChainToAssetAccountingAmount(assets.DefaultCollateralAsset, claimedEscapeClaim.ClaimAmount)
	if err != nil {
		return fmt.Errorf("convert claimed escape collateral amount to accounting units: %w", err)
	}

	summary.PassFailMatrix = append(summary.PassFailMatrix, flowStepResult{
		Step:          "9. forced-withdraw freeze + escape claim",
		Status:        "PASS",
		SignatureMode: "buyer wallet signs the forced-withdraw request and escape collateral claim on FunnyRollupCore; operator signs the freeze transaction after the missed deadline",
		Notes:         fmt.Sprintf("request_id=%d freeze_action=%s escape_action=%s claim_id=%s", forcedWithdrawal.RequestID, freezeProgress.Action, escapeProgress.Action, claimedEscapeClaim.ClaimID),
	})
	summary.IDs.EscapeClaimID = claimedEscapeClaim.ClaimID
	summary.Buyer.EscapedUSDT = escapedBuyerAccounting
	summary.Rollup.LatestEscapeBatchID = claimedEscapeClaim.BatchID
	summary.Rollup.LatestEscapeMerkleRoot = claimedEscapeClaim.MerkleRoot
	summary.Rollup.ForcedRequestID = forcedWithdrawal.RequestID
	summary.Rollup.ForcedRequestStatus = forcedWithdrawal.Status
	summary.Rollup.Frozen = freezeState.Frozen
	summary.Rollup.FrozenAt = freezeState.FrozenAt
	summary.Rollup.EscapeClaimStatus = claimedEscapeClaim.ClaimStatus

	summary.ReadbackCommands = []string{
		fmt.Sprintf("curl -sS '%s/api/v1/trading-keys?wallet_address=%s&vault_address=%s&status=ACTIVE&limit=20'", opts.BaseURL, buyerSession.WalletAddress, buyerSession.VaultAddress),
		fmt.Sprintf("curl -sS '%s/api/v1/deposits?user_id=%d&limit=20'", opts.BaseURL, buyerSession.UserID),
		fmt.Sprintf("curl -sS '%s/api/v1/orders?user_id=%d&market_id=%d&limit=20'", opts.BaseURL, buyerSession.UserID, market.MarketID),
		fmt.Sprintf("curl -sS '%s/api/v1/orders?user_id=%d&market_id=%d&limit=20'", opts.BaseURL, makerSession.UserID, market.MarketID),
		fmt.Sprintf("curl -sS '%s/api/v1/markets/%d'", opts.BaseURL, market.MarketID),
		fmt.Sprintf("curl -sS '%s/api/v1/balances?user_id=%d&limit=20'", opts.BaseURL, buyerSession.UserID),
		fmt.Sprintf("curl -sS '%s/api/v1/positions?user_id=%d&market_id=%d&limit=20'", opts.BaseURL, buyerSession.UserID, market.MarketID),
		fmt.Sprintf("curl -sS '%s/api/v1/payouts?user_id=%d&market_id=%d&limit=20'", opts.BaseURL, buyerSession.UserID, market.MarketID),
		"background chain submitter advances record/publish/accept on FunnyRollupCore",
		fmt.Sprintf("psql '%s' -c \"SELECT batch_id, status, next_state_root FROM rollup_accepted_batches ORDER BY batch_id DESC LIMIT 5;\"", cfg.PostgresDSN),
		fmt.Sprintf("psql '%s' -c \"SELECT submission_id, batch_id, status, record_tx_hash, accept_tx_hash FROM rollup_shadow_submissions ORDER BY batch_id DESC LIMIT 5;\"", cfg.PostgresDSN),
		fmt.Sprintf("curl -sS '%s/api/v1/rollup/forced-withdrawals?wallet_address=%s&limit=20'", opts.BaseURL, buyerSession.WalletAddress),
		fmt.Sprintf("curl -sS '%s/api/v1/rollup/escape-collateral?user_id=%d&wallet_address=%s&limit=20'", opts.BaseURL, buyerSession.UserID, buyerSession.WalletAddress),
		fmt.Sprintf("curl -sS '%s/api/v1/rollup/freeze-state'", opts.BaseURL),
		fmt.Sprintf("psql '%s' -c \"SELECT market_id, status, resolved_outcome, resolver_type, resolver_ref FROM market_resolutions WHERE market_id = %d;\"", cfg.PostgresDSN, market.MarketID),
	}

	out, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal full-flow summary: %w", err)
	}
	fmt.Println(string(out))
	return nil
}

func (c *apiClient) registerTradingKey(ctx context.Context, wallet walletIdentity, chainID int64, vaultAddress string) (sessionContext, tradingKeyChallengeResponse, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return sessionContext{}, tradingKeyChallengeResponse{}, err
	}
	publicKey := hexutil.Encode(pub)
	challenge, err := c.createTradingKeyChallenge(ctx, wallet.Address, chainID, vaultAddress)
	if err != nil {
		return sessionContext{}, tradingKeyChallengeResponse{}, err
	}

	authz := sharedauth.TradingKeyAuthorization{
		WalletAddress:            wallet.Address,
		TradingPublicKey:         publicKey,
		TradingKeyScheme:         sharedauth.DefaultTradingKeyScheme,
		Scope:                    sharedauth.DefaultSessionScope,
		Challenge:                challenge.Challenge,
		ChallengeExpiresAtMillis: challenge.ChallengeExpiresAt,
		KeyExpiresAtMillis:       0,
		ChainID:                  chainID,
		VaultAddress:             vaultAddress,
	}
	signature, err := signTradingKeyAuthorization(authz, wallet.PrivateKey)
	if err != nil {
		return sessionContext{}, tradingKeyChallengeResponse{}, err
	}

	var remote remoteSession
	err = c.doJSON(ctx, http.MethodPost, "/api/v1/trading-keys", map[string]any{
		"wallet_address":            wallet.Address,
		"chain_id":                  chainID,
		"vault_address":             vaultAddress,
		"challenge_id":              challenge.ChallengeID,
		"challenge":                 challenge.Challenge,
		"challenge_expires_at":      challenge.ChallengeExpiresAt,
		"trading_public_key":        publicKey,
		"trading_key_scheme":        sharedauth.DefaultTradingKeyScheme,
		"scope":                     sharedauth.DefaultSessionScope,
		"key_expires_at":            0,
		"wallet_signature_standard": sharedauth.DefaultWalletSignatureStandard,
		"wallet_signature":          signature,
	}, &remote)
	if err != nil {
		return sessionContext{}, tradingKeyChallengeResponse{}, err
	}
	c.rememberSession(remote.UserID, remote.SessionID)

	return sessionContext{
		UserID:        remote.UserID,
		WalletAddress: strings.ToLower(remote.WalletAddress),
		SessionID:     remote.SessionID,
		SessionPubKey: strings.ToLower(remote.SessionPublicKey),
		SessionPriv:   priv,
		LastNonce:     remote.LastOrderNonce,
		ChainID:       remote.ChainID,
		VaultAddress:  strings.ToLower(strings.TrimSpace(remote.VaultAddress)),
	}, challenge, nil
}

func (c *apiClient) createTradingKeyChallenge(ctx context.Context, walletAddress string, chainID int64, vaultAddress string) (tradingKeyChallengeResponse, error) {
	var response tradingKeyChallengeResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/trading-keys/challenge", map[string]any{
		"wallet_address": walletAddress,
		"chain_id":       chainID,
		"vault_address":  vaultAddress,
	}, &response)
	return response, err
}

func (c *apiClient) listRemoteSessions(ctx context.Context, walletAddress, vaultAddress, status string, limit int) ([]remoteSession, error) {
	query := url.Values{}
	if strings.TrimSpace(walletAddress) != "" {
		query.Set("wallet_address", walletAddress)
	}
	if strings.TrimSpace(vaultAddress) != "" {
		query.Set("vault_address", vaultAddress)
	}
	if strings.TrimSpace(status) != "" {
		query.Set("status", status)
	}
	if limit <= 0 {
		limit = defaultSessionListReadback
	}
	query.Set("limit", fmt.Sprintf("%d", limit))

	var result collectionResponse[remoteSession]
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/trading-keys?"+query.Encode(), nil, &result)
	return result.Items, err
}

func (c *apiClient) verifyTruthfulRestore(ctx context.Context, session sessionContext) (remoteSession, error) {
	items, err := c.listRemoteSessions(ctx, session.WalletAddress, session.VaultAddress, "ACTIVE", defaultSessionListReadback)
	if err != nil {
		return remoteSession{}, err
	}
	for _, item := range items {
		if item.SessionID != session.SessionID {
			continue
		}
		if strings.ToLower(item.WalletAddress) != session.WalletAddress {
			return remoteSession{}, fmt.Errorf("wallet mismatch for session %s", item.SessionID)
		}
		if strings.ToLower(strings.TrimSpace(item.VaultAddress)) != session.VaultAddress {
			return remoteSession{}, fmt.Errorf("vault mismatch for session %s", item.SessionID)
		}
		if item.ChainID != session.ChainID {
			return remoteSession{}, fmt.Errorf("chain mismatch for session %s", item.SessionID)
		}
		if strings.ToLower(item.SessionPublicKey) != session.SessionPubKey {
			return remoteSession{}, fmt.Errorf("trading public key mismatch for session %s", item.SessionID)
		}
		if strings.ToUpper(strings.TrimSpace(item.Status)) != "ACTIVE" {
			return remoteSession{}, fmt.Errorf("session %s is not active", item.SessionID)
		}
		if item.ExpiresAtMillis > 0 && item.ExpiresAtMillis <= time.Now().UnixMilli() {
			return remoteSession{}, fmt.Errorf("session %s is expired", item.SessionID)
		}
		return item, nil
	}
	return remoteSession{}, fmt.Errorf("session %s was not found during restore readback", session.SessionID)
}

func signTradingKeyAuthorization(authz sharedauth.TradingKeyAuthorization, key *ecdsa.PrivateKey) (string, error) {
	typedData := authz.Normalize().TypedData()
	hash, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		return "", err
	}
	signature, err := crypto.Sign(hash, key)
	if err != nil {
		return "", err
	}
	return hexutil.Encode(signature), nil
}

func startLocalOracleHarness(parent context.Context, logger *slog.Logger, cfg config.ServiceConfig) (*localOracleHarness, error) {
	dbConn, err := shareddb.OpenPostgres(parent, cfg.PostgresDSN)
	if err != nil {
		return nil, err
	}
	publisher := sharedkafka.NewJSONPublisher(logger, cfg.KafkaBrokers)
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/trades" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("symbol") != localOracleSymbol {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"code":-1121,"msg":"Invalid symbol."}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fmt.Sprintf(`[{"id":1,"price":"%s","qty":"1.00000000","quoteQty":"1.00000000","time":%d,"isBuyerMaker":false,"isBestMatch":true}]`, localOracleObservedPrice, time.Now().UnixMilli())))
	}))

	workerCtx, cancel := context.WithCancel(parent)
	store := oracleservice.NewSQLStore(dbConn)
	provider := oracleservice.NewBinanceProvider(fixture.URL, &http.Client{Timeout: 2 * time.Second})
	worker := oracleservice.NewWorker(logger.With("component", "local-full-flow-oracle"), store, provider, publisher, cfg.KafkaTopics, localOracleWorkerPoll)
	go worker.Start(workerCtx)

	return &localOracleHarness{
		cancel:    cancel,
		fixture:   fixture,
		publisher: publisher,
		db:        dbConn,
	}, nil
}

func (h *localOracleHarness) Close() {
	if h == nil {
		return
	}
	if h.cancel != nil {
		h.cancel()
	}
	if h.fixture != nil {
		h.fixture.Close()
	}
	if h.publisher != nil {
		_ = h.publisher.Close()
	}
	if h.db != nil {
		_ = h.db.Close()
	}
}

func fetchResolutionReadback(ctx context.Context, db *sql.DB, marketID int64) (resolutionReadback, error) {
	var item resolutionReadback
	err := db.QueryRowContext(ctx, `
		SELECT status, COALESCE(resolved_outcome, ''), COALESCE(resolver_type, ''), COALESCE(resolver_ref, ''), COALESCE(evidence, '{}'::jsonb)
		FROM market_resolutions
		WHERE market_id = $1
	`, marketID).Scan(
		&item.Status,
		&item.ResolvedOutcome,
		&item.ResolverType,
		&item.ResolverRef,
		&item.Evidence,
	)
	if err != nil {
		return resolutionReadback{}, err
	}
	return item, nil
}

func decodeOracleDispatch(raw json.RawMessage) (string, int, string) {
	var evidence oracleservice.StoredEvidence
	if err := json.Unmarshal(raw, &evidence); err != nil {
		return "", 0, ""
	}
	dispatchStatus := ""
	dispatchAttempts := 0
	observationID := ""
	if evidence.Dispatch != nil {
		dispatchStatus = evidence.Dispatch.Status
		dispatchAttempts = evidence.Dispatch.AttemptCount
	}
	if evidence.Observation != nil {
		observationID = evidence.Observation.ObservationID
	}
	return dispatchStatus, dispatchAttempts, observationID
}

func fetchAcceptedBatchReadback(ctx context.Context, db *sql.DB) (acceptedBatchReadback, error) {
	var item acceptedBatchReadback
	if err := db.QueryRowContext(ctx, `
		SELECT COALESCE(COUNT(*), 0), COALESCE(MAX(batch_id), 0)
		FROM rollup_accepted_batches
	`).Scan(&item.AcceptedBatchCount, &item.LatestAcceptedBatchID); err != nil {
		return acceptedBatchReadback{}, err
	}
	if item.LatestAcceptedBatchID > 0 {
		if err := db.QueryRowContext(ctx, `
			SELECT COALESCE(next_state_root, '')
			FROM rollup_accepted_batches
			WHERE batch_id = $1
		`, item.LatestAcceptedBatchID).Scan(&item.LatestAcceptedStateRoot); err != nil {
			return acceptedBatchReadback{}, err
		}
	}
	if err := db.QueryRowContext(ctx, `
		SELECT COALESCE(status, '')
		FROM rollup_shadow_submissions
		ORDER BY batch_id DESC, created_at DESC
		LIMIT 1
	`).Scan(&item.LatestSubmissionStatus); err != nil {
		if err == sql.ErrNoRows {
			return item, nil
		}
		return acceptedBatchReadback{}, err
	}
	return item, nil
}

func waitForAcceptedBatchAdvance(ctx context.Context, db *sql.DB, baselineAcceptedBatchID int64) (acceptedBatchReadback, error) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		item, err := fetchAcceptedBatchReadback(ctx, db)
		if err != nil {
			return acceptedBatchReadback{}, err
		}
		if item.LatestAcceptedBatchID > baselineAcceptedBatchID {
			return item, nil
		}
		switch item.LatestSubmissionStatus {
		case "FAILED", "FAILED_BLOCKED", "BLOCKED_AUTH":
			return acceptedBatchReadback{}, fmt.Errorf(
				"accepted batch did not advance and latest submission is %s",
				item.LatestSubmissionStatus,
			)
		}

		select {
		case <-ctx.Done():
			return acceptedBatchReadback{}, ctx.Err()
		case <-ticker.C:
		}
	}
}
