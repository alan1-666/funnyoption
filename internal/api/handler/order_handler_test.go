package handler

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	accountclient "funnyoption/internal/account/client"
	"funnyoption/internal/api/dto"
	sharedauth "funnyoption/internal/shared/auth"
	"funnyoption/internal/shared/kafka"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/gin-gonic/gin"
)

type fakeAccountClient struct {
	freezeResp      accountclient.FreezeRecord
	freezeErr       error
	freezeReq       accountclient.FreezeRequest
	preFreezeCalled bool
	releasedID      string
	releaseCalled   bool
	releaseErr      error
	creditResults   []accountclient.CreditResult
	creditErrs      []error
	debitResults    []accountclient.DebitResult
	debitErrs       []error
	credits         []balanceMutationCall
	debits          []balanceMutationCall
}

type balanceMutationCall struct {
	UserID  int64
	Asset   string
	Amount  int64
	RefType string
	RefID   string
}

func (f *fakeAccountClient) PreFreeze(ctx context.Context, req accountclient.FreezeRequest) (accountclient.FreezeRecord, error) {
	_ = ctx
	f.freezeReq = req
	f.preFreezeCalled = true
	return f.freezeResp, f.freezeErr
}

func (f *fakeAccountClient) ReleaseFreeze(ctx context.Context, freezeID string) error {
	_ = ctx
	f.releaseCalled = true
	f.releasedID = freezeID
	return f.releaseErr
}

func (f *fakeAccountClient) GetBalance(ctx context.Context, userID int64, asset string) (accountclient.Balance, error) {
	_ = ctx
	_ = userID
	_ = asset
	return accountclient.Balance{}, nil
}

func (f *fakeAccountClient) CreditBalance(ctx context.Context, userID int64, asset string, amount int64, refType, refID string) (accountclient.CreditResult, error) {
	_ = ctx
	f.credits = append(f.credits, balanceMutationCall{
		UserID:  userID,
		Asset:   asset,
		Amount:  amount,
		RefType: refType,
		RefID:   refID,
	})
	if len(f.creditErrs) > 0 {
		err := f.creditErrs[0]
		f.creditErrs = f.creditErrs[1:]
		if err != nil {
			return accountclient.CreditResult{}, err
		}
	}
	if len(f.creditResults) > 0 {
		result := f.creditResults[0]
		f.creditResults = f.creditResults[1:]
		if result.Asset == "" {
			result.Asset = asset
		}
		if result.UserID == 0 {
			result.UserID = userID
		}
		return result, nil
	}
	return accountclient.CreditResult{
		UserID:    userID,
		Asset:     asset,
		Available: amount,
		Total:     amount,
		Applied:   true,
	}, nil
}

func (f *fakeAccountClient) DebitBalance(ctx context.Context, userID int64, asset string, amount int64, refType, refID string) (accountclient.DebitResult, error) {
	_ = ctx
	f.debits = append(f.debits, balanceMutationCall{
		UserID:  userID,
		Asset:   asset,
		Amount:  amount,
		RefType: refType,
		RefID:   refID,
	})
	if len(f.debitErrs) > 0 {
		err := f.debitErrs[0]
		f.debitErrs = f.debitErrs[1:]
		if err != nil {
			return accountclient.DebitResult{}, err
		}
	}
	if len(f.debitResults) > 0 {
		result := f.debitResults[0]
		f.debitResults = f.debitResults[1:]
		if result.Asset == "" {
			result.Asset = asset
		}
		if result.UserID == 0 {
			result.UserID = userID
		}
		return result, nil
	}
	return accountclient.DebitResult{
		UserID:  userID,
		Asset:   asset,
		Total:   amount,
		Applied: true,
	}, nil
}

func (f *fakeAccountClient) Close() error {
	return nil
}

type fakePublisher struct {
	topic   string
	key     string
	payload any
	err     error
	errAt   int
	calls   []publishCall
}

type publishCall struct {
	Topic   string
	Key     string
	Payload any
}

type fakeQueryStore struct {
	createMarketResp              dto.MarketResponse
	createMarketErr               error
	createMarketReq               dto.CreateMarketRequest
	createSessionResp             dto.SessionResponse
	createSessionErr              error
	createTradingKeyChallengeResp dto.TradingKeyChallengeResponse
	createTradingKeyChallengeErr  error
	createTradingKeyChallengeReq  dto.CreateTradingKeyChallengeRequest
	registerTradingKeyResp        dto.SessionResponse
	registerTradingKeyErr         error
	registerTradingKeyReq         dto.RegisterTradingKeyRequest
	createClaimResp               dto.ChainTransactionResponse
	createClaimErr                error
	createClaimReq                dto.ClaimPayoutRequest
	createClaimCalled             bool
	getProfileResp                dto.UserProfileResponse
	getProfileErr                 error
	updateProfileResp             dto.UserProfileResponse
	updateProfileErr              error
	updateProfileReq              dto.UpdateUserProfileRequest
	updateProfileWallet           string
	getSessionResp                dto.SessionResponse
	getSessionErr                 error
	revokeSessionResp             dto.SessionResponse
	revokeSessionErr              error
	advanceSessionReq             dto.AdvanceSessionNonceRequest
	advanceSessionResp            dto.SessionResponse
	advanceSessionErr             error
	getMarketResp                 dto.MarketResponse
	getMarketErr                  error
	getMarketResolution           MarketResolutionState
	hasMarketResolution           bool
	listMarketsResp               []dto.MarketResponse
	listMarketsErr                error
	listSessionsReq               dto.ListSessionsRequest
	listSessionsResp              []dto.SessionResponse
	listSessionsErr               error
	listDepositsResp              []dto.DepositResponse
	listDepositsErr               error
	listWithdrawResp              []dto.WithdrawalResponse
	listWithdrawErr               error
	listRollupForcedWithdrawResp  []dto.RollupForcedWithdrawalResponse
	listRollupForcedWithdrawErr   error
	getRollupFreezeResp           dto.RollupFreezeStateResponse
	getRollupFreezeErr            error
	listChainTxResp               []dto.ChainTransactionResponse
	listChainTxErr                error
	listOrdersResp                []dto.OrderResponse
	listOrdersErr                 error
	listTradesResp                []dto.TradeResponse
	listTradesErr                 error
	listBalancesResp              []dto.BalanceResponse
	listBalancesErr               error
	listPositionsResp             []dto.PositionResponse
	listPositionsErr              error
	listPayoutsResp               []dto.PayoutResponse
	listPayoutsErr                error
	listFreezesResp               []dto.FreezeResponse
	listFreezesErr                error
	listEntriesResp               []dto.LedgerEntryResponse
	listEntriesErr                error
	listPostingsResp              []dto.LedgerPostingResponse
	listPostingsErr               error
	liabilityResp                 []dto.LiabilityReportLine
	liabilityErr                  error
	getOrderResp                  dto.OrderResponse
	getOrderErr                   error
	getFreezeResp                 dto.FreezeResponse
	getFreezeErr                  error
}

func (f *fakeQueryStore) CreateMarket(ctx context.Context, req dto.CreateMarketRequest) (dto.MarketResponse, error) {
	_ = ctx
	f.createMarketReq = req
	return f.createMarketResp, f.createMarketErr
}

func (f *fakeQueryStore) CreateSession(ctx context.Context, req dto.CreateSessionRequest) (dto.SessionResponse, error) {
	_ = ctx
	if f.createSessionResp.SessionID == "" {
		f.createSessionResp = dto.SessionResponse{
			SessionID:        req.SessionID,
			UserID:           req.UserID,
			WalletAddress:    req.WalletAddress,
			SessionPublicKey: req.SessionPublicKey,
			Scope:            req.Scope,
			ChainID:          req.ChainID,
			SessionNonce:     req.Nonce,
			Status:           "ACTIVE",
			IssuedAtMillis:   req.IssuedAtMillis,
			ExpiresAtMillis:  req.ExpiresAtMillis,
		}
	}
	return f.createSessionResp, f.createSessionErr
}

func (f *fakeQueryStore) CreateTradingKeyChallenge(ctx context.Context, req dto.CreateTradingKeyChallengeRequest) (dto.TradingKeyChallengeResponse, error) {
	_ = ctx
	f.createTradingKeyChallengeReq = req
	if f.createTradingKeyChallengeResp.ChallengeID == "" {
		f.createTradingKeyChallengeResp = dto.TradingKeyChallengeResponse{
			ChallengeID:        req.ChallengeID,
			Challenge:          req.Challenge,
			ChallengeExpiresAt: req.ChallengeExpiresAt,
		}
	}
	return f.createTradingKeyChallengeResp, f.createTradingKeyChallengeErr
}

func (f *fakeQueryStore) RegisterTradingKey(ctx context.Context, req dto.RegisterTradingKeyRequest) (dto.SessionResponse, error) {
	_ = ctx
	f.registerTradingKeyReq = req
	if f.registerTradingKeyResp.SessionID == "" {
		f.registerTradingKeyResp = dto.SessionResponse{
			SessionID:        req.SessionID,
			UserID:           1001,
			WalletAddress:    req.WalletAddress,
			SessionPublicKey: req.TradingPublicKey,
			Scope:            req.Scope,
			ChainID:          req.ChainID,
			SessionNonce:     strings.TrimPrefix(req.Challenge, "0x"),
			LastOrderNonce:   0,
			Status:           "ACTIVE",
			IssuedAtMillis:   time.Now().UnixMilli(),
			ExpiresAtMillis:  req.KeyExpiresAtMillis,
		}
	}
	return f.registerTradingKeyResp, f.registerTradingKeyErr
}

func (f *fakeQueryStore) CreateClaimRequest(ctx context.Context, req dto.ClaimPayoutRequest) (dto.ChainTransactionResponse, error) {
	_ = ctx
	f.createClaimReq = req
	f.createClaimCalled = true
	if f.createClaimResp.ID == 0 {
		f.createClaimResp = dto.ChainTransactionResponse{
			ID:            1,
			BizType:       "CLAIM",
			RefID:         req.EventID,
			ChainName:     "bsc",
			NetworkName:   "testnet",
			WalletAddress: req.WalletAddress,
			Status:        "PENDING",
		}
	}
	return f.createClaimResp, f.createClaimErr
}

func (f *fakeQueryStore) GetUserProfile(ctx context.Context, req dto.GetUserProfileRequest) (dto.UserProfileResponse, error) {
	_ = ctx
	_ = req
	return f.getProfileResp, f.getProfileErr
}

func (f *fakeQueryStore) UpsertUserProfile(ctx context.Context, req dto.UpdateUserProfileRequest, walletAddress string) (dto.UserProfileResponse, error) {
	_ = ctx
	f.updateProfileReq = req
	f.updateProfileWallet = walletAddress
	if f.updateProfileResp.UserID == 0 {
		f.updateProfileResp = dto.UserProfileResponse{
			UserID:        req.UserID,
			WalletAddress: walletAddress,
			DisplayName:   req.DisplayName,
			AvatarPreset:  req.AvatarPreset,
		}
	}
	return f.updateProfileResp, f.updateProfileErr
}

func (f *fakeQueryStore) GetMarket(ctx context.Context, marketID int64) (dto.MarketResponse, error) {
	_ = ctx
	if f.getMarketResp.MarketID == 0 && f.getMarketErr == nil {
		f.getMarketResp = dto.MarketResponse{
			MarketID:        marketID,
			CollateralAsset: "USDT",
			Status:          "OPEN",
		}
	}
	return f.getMarketResp, f.getMarketErr
}

func (f *fakeQueryStore) GetMarketResolution(ctx context.Context, marketID int64) (MarketResolutionState, bool, error) {
	_ = ctx
	_ = marketID
	return f.getMarketResolution, f.hasMarketResolution, nil
}

func signOperatorMessage(t *testing.T, key *ecdsa.PrivateKey, message string) string {
	t.Helper()

	signature, err := crypto.Sign(accounts.TextHash([]byte(message)), key)
	if err != nil {
		t.Fatalf("Sign returned error: %v", err)
	}
	return hexutil.Encode(signature)
}

func attachSignedCreateMarketOperator(t *testing.T, req *dto.CreateMarketRequest) string {
	t.Helper()

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	wallet := sharedauth.NormalizeHex(crypto.PubkeyToAddress(key.PublicKey).Hex())
	req.Operator = &dto.OperatorAction{
		WalletAddress: wallet,
		RequestedAt:   time.Now().UnixMilli(),
	}
	req.Operator.Signature = signOperatorMessage(t, key, req.OperatorMessage())
	return wallet
}

func validOracleResolutionMetadata() json.RawMessage {
	return json.RawMessage(`{
		"resolution": {
			"version": 1,
			"mode": "ORACLE_PRICE",
			"market_kind": "CRYPTO_PRICE_THRESHOLD",
			"manual_fallback_allowed": true,
			"oracle": {
				"source_kind": "HTTP_JSON",
				"provider_key": "BINANCE",
				"instrument": {
					"kind": "SPOT",
					"base_asset": "BTC",
					"quote_asset": "USDT",
					"symbol": "BTCUSDT"
				},
				"price": {
					"field": "LAST_PRICE",
					"scale": 8,
					"rounding_mode": "ROUND_HALF_UP",
					"max_data_age_sec": 120
				},
				"window": {
					"anchor": "RESOLVE_AT",
					"before_sec": 300,
					"after_sec": 300
				},
				"rule": {
					"type": "PRICE_THRESHOLD",
					"comparator": "GTE",
					"threshold_price": "85000"
				}
			}
		}
	}`)
}

func attachSignedResolveOperator(t *testing.T, marketID int64, req *dto.ResolveMarketRequest) string {
	t.Helper()

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	wallet := sharedauth.NormalizeHex(crypto.PubkeyToAddress(key.PublicKey).Hex())
	req.Operator = &dto.OperatorAction{
		WalletAddress: wallet,
		RequestedAt:   time.Now().UnixMilli(),
	}
	req.Operator.Signature = signOperatorMessage(t, key, req.OperatorMessage(marketID))
	return wallet
}

func attachSignedFirstLiquidityOperator(t *testing.T, marketID int64, req *dto.CreateFirstLiquidityRequest) string {
	t.Helper()

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	wallet := sharedauth.NormalizeHex(crypto.PubkeyToAddress(key.PublicKey).Hex())
	req.Operator = &dto.OperatorAction{
		WalletAddress: wallet,
		RequestedAt:   time.Now().UnixMilli(),
	}
	req.Operator.Signature = signOperatorMessage(t, key, req.OperatorMessage(marketID))
	return wallet
}

func attachSignedBootstrapOrderOperator(t *testing.T, req *dto.CreateOrderRequest) string {
	t.Helper()

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	return signBootstrapOrderOperator(t, key, time.Now().UnixMilli(), req)
}

func signBootstrapOrderOperator(t *testing.T, key *ecdsa.PrivateKey, requestedAt int64, req *dto.CreateOrderRequest) string {
	t.Helper()

	wallet := sharedauth.NormalizeHex(crypto.PubkeyToAddress(key.PublicKey).Hex())
	req.Operator = &dto.OperatorAction{
		WalletAddress: wallet,
		RequestedAt:   requestedAt,
	}
	req.RequestedAtMillis = req.Operator.RequestedAt
	req.Operator.Signature = signOperatorMessage(t, key, req.BootstrapOperatorMessage())
	return wallet
}

func (f *fakeQueryStore) GetSession(ctx context.Context, sessionID string) (dto.SessionResponse, error) {
	_ = ctx
	_ = sessionID
	return f.getSessionResp, f.getSessionErr
}

func (f *fakeQueryStore) GetOrder(ctx context.Context, orderID string) (dto.OrderResponse, error) {
	_ = ctx
	_ = orderID
	if f.getOrderResp.OrderID == "" && f.getOrderErr == nil {
		return dto.OrderResponse{}, ErrNotFound
	}
	return f.getOrderResp, f.getOrderErr
}

func (f *fakeQueryStore) RevokeSession(ctx context.Context, sessionID string) (dto.SessionResponse, error) {
	_ = ctx
	_ = sessionID
	if f.revokeSessionResp.SessionID == "" {
		return f.getSessionResp, f.revokeSessionErr
	}
	return f.revokeSessionResp, f.revokeSessionErr
}

func (f *fakeQueryStore) GetLatestFreezeByRef(ctx context.Context, refType, refID string) (dto.FreezeResponse, error) {
	_ = ctx
	_ = refType
	_ = refID
	if f.getFreezeResp.FreezeID == "" && f.getFreezeErr == nil {
		return dto.FreezeResponse{}, ErrNotFound
	}
	return f.getFreezeResp, f.getFreezeErr
}

func (f *fakeQueryStore) AdvanceSessionNonce(ctx context.Context, req dto.AdvanceSessionNonceRequest) (dto.SessionResponse, error) {
	_ = ctx
	f.advanceSessionReq = req
	if f.advanceSessionResp.SessionID == "" {
		return f.getSessionResp, f.advanceSessionErr
	}
	return f.advanceSessionResp, f.advanceSessionErr
}

func (f *fakeQueryStore) ListMarkets(ctx context.Context, req dto.ListMarketsRequest) ([]dto.MarketResponse, error) {
	_ = ctx
	_ = req
	return f.listMarketsResp, f.listMarketsErr
}

func (f *fakeQueryStore) ListSessions(ctx context.Context, req dto.ListSessionsRequest) ([]dto.SessionResponse, error) {
	_ = ctx
	f.listSessionsReq = req
	return f.listSessionsResp, f.listSessionsErr
}

func (f *fakeQueryStore) ListDeposits(ctx context.Context, req dto.ListDepositsRequest) ([]dto.DepositResponse, error) {
	_ = ctx
	_ = req
	return f.listDepositsResp, f.listDepositsErr
}

func (f *fakeQueryStore) ListWithdrawals(ctx context.Context, req dto.ListWithdrawalsRequest) ([]dto.WithdrawalResponse, error) {
	_ = ctx
	_ = req
	return f.listWithdrawResp, f.listWithdrawErr
}

func (f *fakeQueryStore) ListRollupForcedWithdrawals(ctx context.Context, req dto.ListRollupForcedWithdrawalsRequest) ([]dto.RollupForcedWithdrawalResponse, error) {
	_ = ctx
	_ = req
	return f.listRollupForcedWithdrawResp, f.listRollupForcedWithdrawErr
}

func (f *fakeQueryStore) GetRollupFreezeState(ctx context.Context) (dto.RollupFreezeStateResponse, error) {
	_ = ctx
	return f.getRollupFreezeResp, f.getRollupFreezeErr
}

func (f *fakeQueryStore) ListChainTransactions(ctx context.Context, req dto.ListChainTransactionsRequest) ([]dto.ChainTransactionResponse, error) {
	_ = ctx
	_ = req
	return f.listChainTxResp, f.listChainTxErr
}

func (f *fakeQueryStore) ListOrders(ctx context.Context, req dto.ListOrdersRequest) ([]dto.OrderResponse, error) {
	_ = ctx
	_ = req
	return f.listOrdersResp, f.listOrdersErr
}

func (f *fakeQueryStore) ListTrades(ctx context.Context, req dto.ListTradesRequest) ([]dto.TradeResponse, error) {
	_ = ctx
	_ = req
	return f.listTradesResp, f.listTradesErr
}

func (f *fakeQueryStore) ListBalances(ctx context.Context, req dto.ListBalancesRequest) ([]dto.BalanceResponse, error) {
	_ = ctx
	_ = req
	return f.listBalancesResp, f.listBalancesErr
}

func (f *fakeQueryStore) ListPositions(ctx context.Context, req dto.ListPositionsRequest) ([]dto.PositionResponse, error) {
	_ = ctx
	_ = req
	return f.listPositionsResp, f.listPositionsErr
}

func (f *fakeQueryStore) ListPayouts(ctx context.Context, req dto.ListPayoutsRequest) ([]dto.PayoutResponse, error) {
	_ = ctx
	_ = req
	return f.listPayoutsResp, f.listPayoutsErr
}

func (f *fakeQueryStore) ListFreezes(ctx context.Context, req dto.ListFreezesRequest) ([]dto.FreezeResponse, error) {
	_ = ctx
	_ = req
	return f.listFreezesResp, f.listFreezesErr
}

func (f *fakeQueryStore) ListLedgerEntries(ctx context.Context, req dto.ListLedgerEntriesRequest) ([]dto.LedgerEntryResponse, error) {
	_ = ctx
	_ = req
	return f.listEntriesResp, f.listEntriesErr
}

func (f *fakeQueryStore) ListLedgerPostings(ctx context.Context, entryID string) ([]dto.LedgerPostingResponse, error) {
	_ = ctx
	_ = entryID
	return f.listPostingsResp, f.listPostingsErr
}

func (f *fakeQueryStore) BuildLiabilityReport(ctx context.Context) ([]dto.LiabilityReportLine, error) {
	_ = ctx
	return f.liabilityResp, f.liabilityErr
}

func (f *fakePublisher) PublishJSON(ctx context.Context, topic, key string, payload any) error {
	_ = ctx
	f.topic = topic
	f.key = key
	f.payload = payload
	f.calls = append(f.calls, publishCall{Topic: topic, Key: key, Payload: payload})
	if f.errAt > 0 && len(f.calls) == f.errAt {
		return f.err
	}
	if f.errAt > 0 {
		return nil
	}
	return f.err
}

func (f *fakePublisher) Close() error {
	return nil
}

func TestCreateOrderWithOperatorBootstrapProofPublishesCommand(t *testing.T) {
	gin.SetMode(gin.TestMode)

	account := &fakeAccountClient{
		freezeResp: accountclient.FreezeRecord{
			FreezeID: "frz_1",
			UserID:   1001,
			Asset:    "USDT",
			RefType:  "ORDER",
			RefID:    "ord_x",
			Amount:   200,
		},
	}
	publisher := &fakePublisher{}
	reqBody := dto.CreateOrderRequest{
		UserID:      1001,
		MarketID:    88,
		Outcome:     "yes",
		Side:        "sell",
		Type:        "limit",
		TimeInForce: "gtc",
		Price:       10,
		Quantity:    20,
	}
	wallet := attachSignedBootstrapOrderOperator(t, &reqBody)
	handler := NewOrderHandler(Dependencies{
		Logger:          slog.Default(),
		KafkaPublisher:  publisher,
		KafkaTopics:     kafka.NewTopics("funnyoption."),
		AccountClient:   account,
		QueryStore:      &fakeQueryStore{},
		OperatorWallets: []string{wallet},
	})

	raw, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.CreateOrder(ctx)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", w.Code, w.Body.String())
	}
	if publisher.topic == "" || publisher.key != "88:YES" {
		t.Fatalf("unexpected publish target: topic=%s key=%s", publisher.topic, publisher.key)
	}
	command, ok := publisher.payload.(kafka.OrderCommand)
	if !ok {
		t.Fatalf("expected kafka.OrderCommand payload, got %T", publisher.payload)
	}
	if command.FreezeID != "frz_1" || command.FreezeAmount != 200 || command.CollateralAsset != "USDT" {
		t.Fatalf("unexpected command freeze payload: %+v", command)
	}
	if command.RequestedAtMillis != reqBody.RequestedAtMillis {
		t.Fatalf("expected command requested_at %d, got %d", reqBody.RequestedAtMillis, command.RequestedAtMillis)
	}
	if account.releaseCalled {
		t.Fatalf("release should not be called on success")
	}
}

func TestCreateOrderRejectsBareUserIDWithoutAuthEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	account := &fakeAccountClient{}
	handler := NewOrderHandler(Dependencies{
		Logger:        slog.Default(),
		KafkaTopics:   kafka.NewTopics("funnyoption."),
		AccountClient: account,
		QueryStore:    &fakeQueryStore{},
	})

	body := map[string]any{
		"user_id":       1001,
		"market_id":     88,
		"outcome":       "yes",
		"side":          "buy",
		"type":          "limit",
		"time_in_force": "gtc",
		"price":         10,
		"quantity":      20,
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.CreateOrder(ctx)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}
	if account.preFreezeCalled {
		t.Fatalf("expected unauthorized request to stop before pre-freeze")
	}
}

func TestCreateOrderRejectsReplayedOperatorBootstrapOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	account := &fakeAccountClient{}
	reqBody := dto.CreateOrderRequest{
		UserID:      1001,
		MarketID:    88,
		Outcome:     "yes",
		Side:        "sell",
		Type:        "limit",
		TimeInForce: "gtc",
		Price:       10,
		Quantity:    20,
	}
	wallet := attachSignedBootstrapOrderOperator(t, &reqBody)
	orderID := reqBody.BootstrapOrderID()

	handler := NewOrderHandler(Dependencies{
		Logger:         slog.Default(),
		KafkaPublisher: &fakePublisher{},
		KafkaTopics:    kafka.NewTopics("funnyoption."),
		AccountClient:  account,
		QueryStore: &fakeQueryStore{
			getFreezeResp: dto.FreezeResponse{
				FreezeID: "frz_replay",
				UserID:   1001,
				Asset:    "POSITION:88:YES",
				RefType:  "ORDER",
				RefID:    orderID,
				Status:   "ACTIVE",
			},
		},
		OperatorWallets: []string{wallet},
	})

	raw, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.CreateOrder(ctx)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), orderID) {
		t.Fatalf("expected replay response to include order id, got %s", w.Body.String())
	}
	if account.preFreezeCalled {
		t.Fatalf("expected replay to stop before pre-freeze")
	}
}

func TestCreateOrderRejectsSemanticDuplicateBootstrapOrderWithFreshProof(t *testing.T) {
	gin.SetMode(gin.TestMode)

	account := &fakeAccountClient{}
	reqBody := dto.CreateOrderRequest{
		UserID:      1001,
		MarketID:    88,
		Outcome:     "yes",
		Side:        "sell",
		Type:        "limit",
		TimeInForce: "gtc",
		Price:       10,
		Quantity:    20,
	}

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}

	firstRequestedAt := time.Now().Add(-time.Second).UnixMilli()
	wallet := signBootstrapOrderOperator(t, key, firstRequestedAt, &reqBody)
	orderID := reqBody.BootstrapOrderID()

	secondRequestedAt := time.Now().UnixMilli()
	signBootstrapOrderOperator(t, key, secondRequestedAt, &reqBody)
	if reqBody.BootstrapOrderID() != orderID {
		t.Fatalf("expected semantic bootstrap order id to stay stable across requested_at changes")
	}

	handler := NewOrderHandler(Dependencies{
		Logger:         slog.Default(),
		KafkaPublisher: &fakePublisher{},
		KafkaTopics:    kafka.NewTopics("funnyoption."),
		AccountClient:  account,
		QueryStore: &fakeQueryStore{
			getOrderResp: dto.OrderResponse{
				OrderID:  orderID,
				UserID:   1001,
				MarketID: 88,
				Outcome:  "YES",
				Side:     "SELL",
			},
		},
		OperatorWallets: []string{wallet},
	})

	raw, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.CreateOrder(ctx)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), orderID) {
		t.Fatalf("expected duplicate response to include order id, got %s", w.Body.String())
	}
	if account.preFreezeCalled {
		t.Fatalf("expected semantic duplicate to stop before pre-freeze")
	}
}

func TestCreateTradingKeyChallengeReturnsCreated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakeQueryStore{}
	handler := NewOrderHandler(Dependencies{
		Logger:               slog.Default(),
		QueryStore:           store,
		ExpectedChainID:      97,
		ExpectedVaultAddress: "0x00000000000000000000000000000000000000bb",
	})

	body := map[string]any{
		"wallet_address": "0x00000000000000000000000000000000000000aa",
		"chain_id":       97,
		"vault_address":  "0x00000000000000000000000000000000000000bb",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trading-keys/challenge", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.CreateTradingKeyChallenge(ctx)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"challenge_id\":\"tkc_") {
		t.Fatalf("expected challenge response, got %s", w.Body.String())
	}
	if store.createTradingKeyChallengeReq.WalletAddress != "0x00000000000000000000000000000000000000aa" {
		t.Fatalf("unexpected wallet normalization: %s", store.createTradingKeyChallengeReq.WalletAddress)
	}
}

func TestRegisterTradingKeyVerifiesWalletSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	publicKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}

	authz := sharedauth.TradingKeyAuthorization{
		WalletAddress:            crypto.PubkeyToAddress(privateKey.PublicKey).Hex(),
		TradingPublicKey:         hexutil.Encode(publicKey),
		TradingKeyScheme:         "ED25519",
		Scope:                    "TRADE",
		Challenge:                "0x5fbe9af9d6ab53d4df3bcb43f9e6c5f26a4d9bc2a8f44a0ab2997f7dc2c5c94a",
		ChallengeExpiresAtMillis: time.Now().Add(5 * time.Minute).UnixMilli(),
		KeyExpiresAtMillis:       0,
		ChainID:                  97,
		VaultAddress:             "0x00000000000000000000000000000000000000bb",
	}
	digest, _, err := apitypes.TypedDataAndHash(authz.TypedData())
	if err != nil {
		t.Fatalf("TypedDataAndHash returned error: %v", err)
	}
	signature, err := crypto.Sign(digest, privateKey)
	if err != nil {
		t.Fatalf("Sign returned error: %v", err)
	}

	store := &fakeQueryStore{}
	handler := NewOrderHandler(Dependencies{
		Logger:               slog.Default(),
		QueryStore:           store,
		ExpectedChainID:      97,
		ExpectedVaultAddress: authz.VaultAddress,
	})

	body := map[string]any{
		"wallet_address":            authz.WalletAddress,
		"chain_id":                  authz.ChainID,
		"vault_address":             authz.VaultAddress,
		"challenge_id":              "tkc_01HTY5V1S8E9Q3P8W2V5K19J4P",
		"challenge":                 authz.Challenge,
		"challenge_expires_at":      authz.ChallengeExpiresAtMillis,
		"trading_public_key":        authz.TradingPublicKey,
		"trading_key_scheme":        authz.TradingKeyScheme,
		"scope":                     authz.Scope,
		"key_expires_at":            authz.KeyExpiresAtMillis,
		"wallet_signature_standard": "EIP712_V4",
		"wallet_signature":          hexutil.Encode(signature),
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trading-keys", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.RegisterTradingKey(ctx)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}
	if store.registerTradingKeyReq.SessionID != authz.TradingKeyID() {
		t.Fatalf("expected deterministic trading key id, got %s", store.registerTradingKeyReq.SessionID)
	}
	if store.registerTradingKeyReq.WalletAddress != sharedauth.NormalizeHex(authz.WalletAddress) {
		t.Fatalf("unexpected wallet normalization: %s", store.registerTradingKeyReq.WalletAddress)
	}
}

func TestListSessionsPassesVaultFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakeQueryStore{
		listSessionsResp: []dto.SessionResponse{
			{
				SessionID:        "tk_01",
				UserID:           1001,
				WalletAddress:    "0x00000000000000000000000000000000000000aa",
				SessionPublicKey: "0x1111",
				Scope:            "TRADE",
				ChainID:          97,
				VaultAddress:     "0x00000000000000000000000000000000000000bb",
				Status:           "ACTIVE",
			},
		},
	}
	handler := NewOrderHandler(Dependencies{
		Logger:     slog.Default(),
		QueryStore: store,
	})

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/sessions?wallet_address=0x00000000000000000000000000000000000000aa&vault_address=0x00000000000000000000000000000000000000bb&status=ACTIVE&limit=10",
		nil,
	)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.ListSessions(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if store.listSessionsReq.WalletAddress != "0x00000000000000000000000000000000000000aa" {
		t.Fatalf("unexpected wallet filter: %s", store.listSessionsReq.WalletAddress)
	}
	if store.listSessionsReq.VaultAddress != "0x00000000000000000000000000000000000000bb" {
		t.Fatalf("unexpected vault filter: %s", store.listSessionsReq.VaultAddress)
	}
	if !strings.Contains(w.Body.String(), "\"vault_address\":\"0x00000000000000000000000000000000000000bb\"") {
		t.Fatalf("expected vault_address in response body, got %s", w.Body.String())
	}
}

func TestCreateSessionVerifiesWalletSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	walletAddress := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
	grant := sharedauth.SessionGrant{
		WalletAddress:    walletAddress,
		SessionPublicKey: "0xsessionpub",
		Scope:            "TRADE",
		ChainID:          97,
		Nonce:            "sess_123",
		IssuedAtMillis:   time.Now().Add(-time.Minute).UnixMilli(),
		ExpiresAtMillis:  time.Now().Add(time.Hour).UnixMilli(),
	}
	digest := accounts.TextHash([]byte(grant.Message()))
	signature, err := crypto.Sign(digest, privateKey)
	if err != nil {
		t.Fatalf("Sign returned error: %v", err)
	}

	store := &fakeQueryStore{}
	handler := NewOrderHandler(Dependencies{
		Logger:     slog.Default(),
		QueryStore: store,
	})

	body := map[string]any{
		"user_id":            1001,
		"wallet_address":     walletAddress,
		"session_public_key": "0xsessionpub",
		"scope":              "TRADE",
		"chain_id":           97,
		"nonce":              "sess_123",
		"issued_at":          grant.IssuedAtMillis,
		"expires_at":         grant.ExpiresAtMillis,
		"wallet_signature":   hexutil.Encode(signature),
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.CreateSession(ctx)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "sess_") {
		t.Fatalf("expected session_id in response, got %s", w.Body.String())
	}
}

func TestRevokeSessionReturnsRevokedRecord(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakeQueryStore{
		revokeSessionResp: dto.SessionResponse{
			SessionID:       "sess_revoke",
			UserID:          1001,
			WalletAddress:   "0x00000000000000000000000000000000000000aa",
			Status:          "REVOKED",
			RevokedAtMillis: time.Now().UnixMilli(),
		},
	}
	handler := NewOrderHandler(Dependencies{
		Logger:     slog.Default(),
		QueryStore: store,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/sess_revoke/revoke", nil)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "session_id", Value: "sess_revoke"}}
	ctx.Request = req

	handler.RevokeSession(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"REVOKED\"") {
		t.Fatalf("expected revoked response, got %s", w.Body.String())
	}
}

func TestGetProfileReturnsRecord(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakeQueryStore{
		getProfileResp: dto.UserProfileResponse{
			UserID:        1001,
			WalletAddress: "0x00000000000000000000000000000000000000aa",
			DisplayName:   "Alice",
			AvatarPreset:  "ocean",
		},
	}
	handler := NewOrderHandler(Dependencies{
		Logger:     slog.Default(),
		QueryStore: store,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile?user_id=1001", nil)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.GetProfile(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"avatar_preset\":\"ocean\"") {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestUpdateProfileRequiresActiveSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakeQueryStore{
		getSessionResp: dto.SessionResponse{
			SessionID:       "sess_profile",
			UserID:          1001,
			WalletAddress:   "0x00000000000000000000000000000000000000aa",
			Status:          "ACTIVE",
			IssuedAtMillis:  time.Now().Add(-time.Minute).UnixMilli(),
			ExpiresAtMillis: time.Now().Add(time.Hour).UnixMilli(),
		},
	}
	handler := NewOrderHandler(Dependencies{
		Logger:     slog.Default(),
		QueryStore: store,
	})

	body := map[string]any{
		"user_id":       1001,
		"session_id":    "sess_profile",
		"display_name":  "Alice",
		"avatar_preset": "forest",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/profile", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.UpdateProfile(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if store.updateProfileReq.AvatarPreset != "forest" {
		t.Fatalf("expected normalized preset to reach store, got %q", store.updateProfileReq.AvatarPreset)
	}
	if store.updateProfileWallet != "0x00000000000000000000000000000000000000aa" {
		t.Fatalf("unexpected wallet propagated to store: %s", store.updateProfileWallet)
	}
}

func TestUpdateProfileRejectsInvalidPreset(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewOrderHandler(Dependencies{
		Logger:     slog.Default(),
		QueryStore: &fakeQueryStore{},
	})

	body := map[string]any{
		"user_id":       1001,
		"session_id":    "sess_profile",
		"display_name":  "Alice",
		"avatar_preset": "not-real",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/profile", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.UpdateProfile(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestCreateOrderWithSessionSignaturePublishesCommand(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sessionPub, sessionPriv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}

	account := &fakeAccountClient{
		freezeResp: accountclient.FreezeRecord{
			FreezeID: "frz_session",
			UserID:   1001,
			Asset:    "USDT",
			RefType:  "ORDER",
			RefID:    "ord_x",
			Amount:   200,
		},
	}
	store := &fakeQueryStore{
		getSessionResp: dto.SessionResponse{
			SessionID:        "tk_live",
			UserID:           1001,
			WalletAddress:    "0x00000000000000000000000000000000000000aa",
			SessionPublicKey: hexutil.Encode(sessionPub),
			Scope:            "TRADE",
			ChainID:          97,
			VaultAddress:     "0x00000000000000000000000000000000000000bb",
			SessionNonce:     "0x5fbe9af9d6ab53d4df3bcb43f9e6c5f26a4d9bc2a8f44a0ab2997f7dc2c5c94a",
			Status:           "ACTIVE",
			IssuedAtMillis:   time.Now().Add(-time.Minute).UnixMilli(),
			ExpiresAtMillis:  time.Now().Add(time.Hour).UnixMilli(),
		},
	}
	publisher := &fakePublisher{}
	handler := NewOrderHandler(Dependencies{
		Logger:         slog.Default(),
		KafkaPublisher: publisher,
		KafkaTopics:    kafka.NewTopics("funnyoption."),
		AccountClient:  account,
		QueryStore:     store,
	})

	intent := sharedauth.OrderIntent{
		SessionID:         "tk_live",
		WalletAddress:     "0x00000000000000000000000000000000000000aa",
		UserID:            1001,
		MarketID:          88,
		Outcome:           "YES",
		Side:              "BUY",
		OrderType:         "LIMIT",
		TimeInForce:       "GTC",
		Price:             10,
		Quantity:          20,
		ClientOrderID:     "cli-sess-1",
		Nonce:             7,
		RequestedAtMillis: time.Now().UnixMilli(),
	}
	signature := ed25519.Sign(sessionPriv, []byte(intent.Message()))

	body := map[string]any{
		"session_id":        intent.SessionID,
		"session_signature": hexutil.Encode(signature),
		"order_nonce":       intent.Nonce,
		"requested_at":      intent.RequestedAtMillis,
		"market_id":         intent.MarketID,
		"outcome":           "yes",
		"side":              "buy",
		"type":              "limit",
		"time_in_force":     "gtc",
		"price":             intent.Price,
		"quantity":          intent.Quantity,
		"client_order_id":   intent.ClientOrderID,
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.CreateOrder(ctx)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", w.Code, w.Body.String())
	}
	command, ok := publisher.payload.(kafka.OrderCommand)
	if !ok {
		t.Fatalf("expected kafka.OrderCommand payload, got %T", publisher.payload)
	}
	if command.UserID != 1001 {
		t.Fatalf("expected command user_id 1001, got %d", command.UserID)
	}
	if command.RequestedAtMillis != intent.RequestedAtMillis {
		t.Fatalf("unexpected requested_at: %d", command.RequestedAtMillis)
	}
	if store.advanceSessionReq.SessionID != intent.SessionID {
		t.Fatalf("unexpected nonce advance session_id: %s", store.advanceSessionReq.SessionID)
	}
	if store.advanceSessionReq.AuthorizationWitness == nil {
		t.Fatalf("expected nonce advance auth witness")
	}
	if !store.advanceSessionReq.AuthorizationWitness.VerifierEligible {
		t.Fatalf("expected verifier-eligible auth witness, got %+v", store.advanceSessionReq.AuthorizationWitness)
	}
	if store.advanceSessionReq.AuthorizationWitness.AuthorizationRef != "tk_live:0x5fbe9af9d6ab53d4df3bcb43f9e6c5f26a4d9bc2a8f44a0ab2997f7dc2c5c94a" {
		t.Fatalf("unexpected authorization_ref: %s", store.advanceSessionReq.AuthorizationWitness.AuthorizationRef)
	}
	if store.advanceSessionReq.AuthorizationWitness.Intent.MessageHash == "" {
		t.Fatalf("expected order intent message hash in auth witness")
	}
}

func TestCreateClaimPayoutReturnsAccepted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakeQueryStore{}
	handler := NewOrderHandler(Dependencies{
		Logger:     slog.Default(),
		QueryStore: store,
	})

	body := map[string]any{
		"user_id":           1001,
		"wallet_address":    "0x00000000000000000000000000000000000000aa",
		"recipient_address": "0x00000000000000000000000000000000000000bb",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payouts/evt_1/claim", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "event_id", Value: "evt_1"}}
	ctx.Request = req

	handler.CreateClaimPayout(ctx)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"biz_type\":\"CLAIM\"") {
		t.Fatalf("unexpected response body: %s", w.Body.String())
	}
	if !store.createClaimCalled {
		t.Fatalf("expected claim request to be created")
	}
	if store.createClaimReq.WalletAddress != "0x00000000000000000000000000000000000000aa" {
		t.Fatalf("unexpected normalized wallet address: %s", store.createClaimReq.WalletAddress)
	}
	if store.createClaimReq.RecipientAddress != "0x00000000000000000000000000000000000000bb" {
		t.Fatalf("unexpected normalized recipient address: %s", store.createClaimReq.RecipientAddress)
	}
}

func TestCreateClaimPayoutRejectsMalformedWalletAddress(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakeQueryStore{}
	handler := NewOrderHandler(Dependencies{
		Logger:     slog.Default(),
		QueryStore: store,
	})

	body := map[string]any{
		"user_id":           1001,
		"wallet_address":    "not-an-address",
		"recipient_address": "0x00000000000000000000000000000000000000bb",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payouts/evt_1/claim", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "event_id", Value: "evt_1"}}
	ctx.Request = req

	handler.CreateClaimPayout(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
	if store.createClaimCalled {
		t.Fatalf("expected malformed request to be rejected before queue creation")
	}
	if !strings.Contains(w.Body.String(), "wallet_address must be a valid EVM address") {
		t.Fatalf("unexpected response body: %s", w.Body.String())
	}
}

func TestCreateClaimPayoutRejectsFrozenRollup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakeQueryStore{
		getRollupFreezeResp: dto.RollupFreezeStateResponse{Frozen: true},
	}
	handler := NewOrderHandler(Dependencies{
		Logger:     slog.Default(),
		QueryStore: store,
	})

	body := map[string]any{
		"user_id":           1001,
		"wallet_address":    "0x1111111111111111111111111111111111111111",
		"recipient_address": "0x1111111111111111111111111111111111111111",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payouts/evt_1/claim", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "event_id", Value: "evt_1"}}
	ctx.Request = req

	handler.CreateClaimPayout(ctx)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "rollup is frozen") {
		t.Fatalf("expected frozen rollup error, got %s", w.Body.String())
	}
	if store.createClaimCalled {
		t.Fatalf("expected no claim request creation while frozen")
	}
}

func TestCreateOrderReleasesFreezeWhenPublishFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	account := &fakeAccountClient{
		freezeResp: accountclient.FreezeRecord{
			FreezeID: "frz_fail",
			UserID:   1001,
			Asset:    "USDT",
			RefType:  "ORDER",
			RefID:    "ord_x",
			Amount:   200,
		},
	}
	publisher := &fakePublisher{err: errors.New("kafka down")}
	reqBody := dto.CreateOrderRequest{
		UserID:      1001,
		MarketID:    88,
		Outcome:     "yes",
		Side:        "sell",
		Type:        "limit",
		TimeInForce: "gtc",
		Price:       10,
		Quantity:    20,
	}
	wallet := attachSignedBootstrapOrderOperator(t, &reqBody)
	handler := NewOrderHandler(Dependencies{
		Logger:          slog.Default(),
		KafkaPublisher:  publisher,
		KafkaTopics:     kafka.NewTopics("funnyoption."),
		AccountClient:   account,
		QueryStore:      &fakeQueryStore{},
		OperatorWallets: []string{wallet},
	})

	raw, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.CreateOrder(ctx)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d body=%s", w.Code, w.Body.String())
	}
	if !account.releaseCalled || account.releasedID != "frz_fail" {
		t.Fatalf("expected release freeze call, got called=%v freeze_id=%s", account.releaseCalled, account.releasedID)
	}
}

func TestCreateOrderRejectsResolvedMarketBeforeFreeze(t *testing.T) {
	gin.SetMode(gin.TestMode)

	account := &fakeAccountClient{
		freezeResp: accountclient.FreezeRecord{
			FreezeID: "frz_should_not_exist",
			UserID:   1001,
			Asset:    "USDT",
			RefType:  "ORDER",
			RefID:    "ord_x",
			Amount:   200,
		},
	}
	reqBody := dto.CreateOrderRequest{
		UserID:      1001,
		MarketID:    88,
		Outcome:     "yes",
		Side:        "sell",
		Type:        "limit",
		TimeInForce: "gtc",
		Price:       10,
		Quantity:    20,
	}
	wallet := attachSignedBootstrapOrderOperator(t, &reqBody)
	handler := NewOrderHandler(Dependencies{
		Logger:        slog.Default(),
		KafkaTopics:   kafka.NewTopics("funnyoption."),
		AccountClient: account,
		QueryStore: &fakeQueryStore{
			getMarketResp: dto.MarketResponse{
				MarketID: 88,
				Status:   "RESOLVED",
			},
		},
		OperatorWallets: []string{wallet},
	})

	raw, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.CreateOrder(ctx)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", w.Code, w.Body.String())
	}
	if account.preFreezeCalled {
		t.Fatalf("expected no pre-freeze attempt for resolved market")
	}
}

func TestCreateOrderRejectsPastCloseAtMarketBeforeFreeze(t *testing.T) {
	gin.SetMode(gin.TestMode)

	account := &fakeAccountClient{
		freezeResp: accountclient.FreezeRecord{
			FreezeID: "frz_should_not_exist",
			UserID:   1001,
			Asset:    "USDT",
			RefType:  "ORDER",
			RefID:    "ord_x",
			Amount:   200,
		},
	}
	reqBody := dto.CreateOrderRequest{
		UserID:      1001,
		MarketID:    88,
		Outcome:     "yes",
		Side:        "sell",
		Type:        "limit",
		TimeInForce: "gtc",
		Price:       10,
		Quantity:    20,
	}
	wallet := attachSignedBootstrapOrderOperator(t, &reqBody)
	handler := NewOrderHandler(Dependencies{
		Logger:        slog.Default(),
		KafkaTopics:   kafka.NewTopics("funnyoption."),
		AccountClient: account,
		QueryStore: &fakeQueryStore{
			getMarketResp: dto.MarketResponse{
				MarketID: 88,
				Status:   "OPEN",
				CloseAt:  time.Now().Unix() - 1,
			},
		},
		OperatorWallets: []string{wallet},
	})

	raw, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.CreateOrder(ctx)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", w.Code, w.Body.String())
	}
	if account.preFreezeCalled {
		t.Fatalf("expected no pre-freeze attempt for post-close market")
	}
}

func TestCreateOrderRejectsFrozenRollupBeforeFreeze(t *testing.T) {
	gin.SetMode(gin.TestMode)

	account := &fakeAccountClient{
		freezeResp: accountclient.FreezeRecord{
			FreezeID: "frz_should_not_exist",
			UserID:   1001,
			Asset:    "USDT",
			RefType:  "ORDER",
			RefID:    "ord_x",
			Amount:   200,
		},
	}
	reqBody := dto.CreateOrderRequest{
		UserID:      1001,
		MarketID:    88,
		Outcome:     "yes",
		Side:        "sell",
		Type:        "limit",
		TimeInForce: "gtc",
		Price:       10,
		Quantity:    20,
	}
	wallet := attachSignedBootstrapOrderOperator(t, &reqBody)
	handler := NewOrderHandler(Dependencies{
		Logger:        slog.Default(),
		KafkaTopics:   kafka.NewTopics("funnyoption."),
		AccountClient: account,
		QueryStore: &fakeQueryStore{
			getRollupFreezeResp: dto.RollupFreezeStateResponse{Frozen: true},
		},
		OperatorWallets: []string{wallet},
	})

	raw, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.CreateOrder(ctx)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "rollup is frozen") {
		t.Fatalf("expected frozen rollup error, got %s", w.Body.String())
	}
	if account.preFreezeCalled {
		t.Fatalf("expected no pre-freeze attempt for frozen rollup")
	}
}

func TestCreateMarketReturnsCreatedMarket(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakeQueryStore{
		createMarketResp: dto.MarketResponse{
			MarketID:        101,
			Title:           "BTC Above 100k",
			CollateralAsset: "USDT",
			Status:          "OPEN",
			CreatedBy:       42,
		},
	}
	reqBody := dto.CreateMarketRequest{
		Title: "BTC Above 100k",
	}
	wallet := attachSignedCreateMarketOperator(t, &reqBody)
	handler := NewOrderHandler(Dependencies{
		Logger:                slog.Default(),
		QueryStore:            store,
		OperatorWallets:       []string{wallet},
		DefaultOperatorUserID: 42,
	})

	raw, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/markets", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.CreateMarket(ctx)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"market_id\":101") {
		t.Fatalf("unexpected response body: %s", w.Body.String())
	}
	if store.createMarketReq.CreatedBy != 42 {
		t.Fatalf("expected create market to use configured operator user id, got %d", store.createMarketReq.CreatedBy)
	}
}

func TestCreateMarketRejectsFrozenRollup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakeQueryStore{
		getRollupFreezeResp: dto.RollupFreezeStateResponse{Frozen: true},
	}
	reqBody := dto.CreateMarketRequest{
		Title: "BTC Above 100k",
	}
	wallet := attachSignedCreateMarketOperator(t, &reqBody)
	handler := NewOrderHandler(Dependencies{
		Logger:                slog.Default(),
		QueryStore:            store,
		OperatorWallets:       []string{wallet},
		DefaultOperatorUserID: 42,
	})

	raw, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/markets", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.CreateMarket(ctx)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "rollup is frozen") {
		t.Fatalf("expected frozen rollup error, got %s", w.Body.String())
	}
	if store.createMarketReq.Title != "" {
		t.Fatalf("expected no market creation write while frozen, got %+v", store.createMarketReq)
	}
}

func TestCreateMarketMergesCoverMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakeQueryStore{
		createMarketResp: dto.MarketResponse{
			MarketID:        202,
			Title:           "ETH Breakout",
			CollateralAsset: "USDT",
			Status:          "OPEN",
			CreatedBy:       55,
		},
	}
	reqBody := dto.CreateMarketRequest{
		Title:           "ETH Breakout",
		Description:     "Will ETH break above the trigger?",
		Metadata:        json.RawMessage(`{"category":"crypto","volume":12345,"sourceSlug":"eth-breakout","sourceKind":"manual"}`),
		CoverImageURL:   "https://cdn.example.com/eth-cover.png",
		CoverSourceURL:  "https://polymarket.com/event/eth-breakout",
		CoverSourceName: "Polymarket",
	}
	wallet := attachSignedCreateMarketOperator(t, &reqBody)
	handler := NewOrderHandler(Dependencies{
		Logger:                slog.Default(),
		QueryStore:            store,
		OperatorWallets:       []string{wallet},
		DefaultOperatorUserID: 55,
	})

	raw, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/markets", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.CreateMarket(ctx)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}

	var metadata map[string]any
	if err := json.Unmarshal(store.createMarketReq.Metadata, &metadata); err != nil {
		t.Fatalf("metadata should be valid json, got %s: %v", string(store.createMarketReq.Metadata), err)
	}
	if got := metadata["cover_image_url"]; got != "https://cdn.example.com/eth-cover.png" {
		t.Fatalf("unexpected cover_image_url: %#v", got)
	}
	if got := metadata["cover_source_url"]; got != "https://polymarket.com/event/eth-breakout" {
		t.Fatalf("unexpected cover_source_url: %#v", got)
	}
	if got := metadata["cover_source_name"]; got != "Polymarket" {
		t.Fatalf("unexpected cover_source_name: %#v", got)
	}
	if got := metadata["coverImage"]; got != "https://cdn.example.com/eth-cover.png" {
		t.Fatalf("unexpected coverImage: %#v", got)
	}
	if got := metadata["sourceUrl"]; got != "https://polymarket.com/event/eth-breakout" {
		t.Fatalf("unexpected sourceUrl: %#v", got)
	}
	if got := metadata["sourceName"]; got != "Polymarket" {
		t.Fatalf("unexpected sourceName: %#v", got)
	}
	if got := metadata["category"]; got != "crypto" {
		t.Fatalf("expected existing metadata to be preserved, got category=%#v", got)
	}
	if got := metadata["operatorWalletAddress"]; got != wallet {
		t.Fatalf("unexpected operatorWalletAddress: %#v", got)
	}
	if got := metadata["operatorService"]; got != "shared-api" {
		t.Fatalf("unexpected operatorService: %#v", got)
	}
	if store.createMarketReq.CreatedBy != 55 {
		t.Fatalf("expected create market to use configured operator user id, got %d", store.createMarketReq.CreatedBy)
	}
}

func TestCreateMarketRejectsMissingOperatorProof(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakeQueryStore{}
	handler := NewOrderHandler(Dependencies{
		Logger:          slog.Default(),
		QueryStore:      store,
		OperatorWallets: []string{"0x1234"},
	})

	body := map[string]any{
		"title": "BTC Above 100k",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/markets", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.CreateMarket(ctx)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}
	if store.createMarketReq.Title != "" {
		t.Fatalf("expected store create to be skipped, got %+v", store.createMarketReq)
	}
}

func TestCreateMarketRejectsNonBinaryOpenOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakeQueryStore{}
	reqBody := dto.CreateMarketRequest{
		Title:  "今晚英超谁会赢",
		Status: "OPEN",
		Options: []dto.MarketOption{
			{Key: "ARS", Label: "阿森纳"},
			{Key: "DRAW", Label: "平局"},
			{Key: "MCI", Label: "曼城"},
		},
	}
	wallet := attachSignedCreateMarketOperator(t, &reqBody)
	handler := NewOrderHandler(Dependencies{
		Logger:                slog.Default(),
		QueryStore:            store,
		OperatorWallets:       []string{wallet},
		DefaultOperatorUserID: 42,
	})

	raw, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/markets", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.CreateMarket(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
	if store.createMarketReq.Title != "" {
		t.Fatalf("expected create market to be rejected before hitting store, got %+v", store.createMarketReq)
	}
}

func TestCreateMarketRejectsInvalidOracleResolutionMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakeQueryStore{}
	reqBody := dto.CreateMarketRequest{
		Title:       "BTC Above 85k",
		ResolveAt:   1775886400,
		Metadata:    json.RawMessage(`{"resolution":{"mode":"ORACLE_PRICE","version":1,"market_kind":"CRYPTO_PRICE_THRESHOLD","manual_fallback_allowed":true,"oracle":{"source_kind":"HTTP_JSON","provider_key":"COINBASE"}}}`),
		CategoryKey: "CRYPTO",
	}
	wallet := attachSignedCreateMarketOperator(t, &reqBody)
	handler := NewOrderHandler(Dependencies{
		Logger:                slog.Default(),
		QueryStore:            store,
		OperatorWallets:       []string{wallet},
		DefaultOperatorUserID: 42,
	})

	raw, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/markets", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.CreateMarket(ctx)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
	if store.createMarketReq.Title != "" {
		t.Fatalf("expected invalid oracle metadata to be rejected before hitting store, got %+v", store.createMarketReq)
	}
}

func TestCreateFirstLiquidityIssuesPairedInventory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	account := &fakeAccountClient{
		freezeResp: accountclient.FreezeRecord{
			FreezeID: "frz_bootstrap_88",
			UserID:   1002,
			Asset:    "POSITION:88:YES",
			RefType:  "ORDER",
			RefID:    "ord_bootstrap",
			Amount:   40,
		},
		debitResults: []accountclient.DebitResult{
			{Applied: true, Asset: "USDT", UserID: 1002},
		},
		creditResults: []accountclient.CreditResult{
			{Applied: true, Asset: "POSITION:88:YES", UserID: 1002},
			{Applied: true, Asset: "POSITION:88:NO", UserID: 1002},
		},
	}
	publisher := &fakePublisher{}
	handler := NewOrderHandler(Dependencies{
		Logger:         slog.Default(),
		KafkaPublisher: publisher,
		KafkaTopics:    kafka.NewTopics("funnyoption."),
		AccountClient:  account,
		QueryStore: &fakeQueryStore{
			getMarketResp: dto.MarketResponse{
				MarketID:        88,
				CollateralAsset: "USDT",
				Status:          "OPEN",
			},
		},
	})
	reqBody := dto.CreateFirstLiquidityRequest{
		UserID:   1002,
		Quantity: 40,
		Outcome:  "YES",
		Price:    55,
	}
	wallet := attachSignedFirstLiquidityOperator(t, 88, &reqBody)
	handler = NewOrderHandler(Dependencies{
		Logger:         slog.Default(),
		KafkaPublisher: publisher,
		KafkaTopics:    kafka.NewTopics("funnyoption."),
		AccountClient:  account,
		QueryStore: &fakeQueryStore{
			getMarketResp: dto.MarketResponse{
				MarketID:        88,
				CollateralAsset: "USDT",
				Status:          "OPEN",
			},
		},
		OperatorWallets: []string{wallet},
	})

	raw, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/markets/88/first-liquidity", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "market_id", Value: "88"}}
	ctx.Request = req

	handler.CreateFirstLiquidity(ctx)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", w.Code, w.Body.String())
	}
	if len(account.debits) != 1 {
		t.Fatalf("expected 1 debit call, got %+v", account.debits)
	}
	if got := account.debits[0]; got.Asset != "USDT" || got.Amount != 4000 || got.RefType != "FIRST_LIQUIDITY_COLLATERAL" {
		t.Fatalf("unexpected collateral debit: %+v", got)
	}
	if len(account.credits) != 2 {
		t.Fatalf("expected 2 inventory credits, got %+v", account.credits)
	}
	if got := account.credits[0]; got.Asset != "POSITION:88:YES" || got.Amount != 40 || got.RefType != "FIRST_LIQUIDITY_POSITION" {
		t.Fatalf("unexpected YES inventory credit: %+v", got)
	}
	if got := account.credits[1]; got.Asset != "POSITION:88:NO" || got.Amount != 40 || got.RefType != "FIRST_LIQUIDITY_POSITION" {
		t.Fatalf("unexpected NO inventory credit: %+v", got)
	}
	if !account.preFreezeCalled {
		t.Fatalf("expected bootstrap order pre-freeze to be created")
	}
	if got := account.freezeReq; got.UserID != 1002 || got.Asset != "POSITION:88:YES" || got.RefType != "ORDER" || got.Amount != 40 {
		t.Fatalf("unexpected bootstrap order freeze request: %+v", got)
	}
	if got := account.freezeReq.RefID; got == "" || !strings.HasPrefix(got, "ord_bootstrap_") {
		t.Fatalf("expected semantic bootstrap order ref id, got %+v", account.freezeReq)
	}
	if len(publisher.calls) != 3 {
		t.Fatalf("expected 2 position events plus 1 order command, got %+v", publisher.calls)
	}
	firstEvent, ok := publisher.calls[0].Payload.(kafka.PositionChangedEvent)
	if !ok {
		t.Fatalf("expected first publish to be position event, got %#v", publisher.calls[0].Payload)
	}
	if firstEvent.Outcome != "YES" || firstEvent.DeltaQuantity != 40 {
		t.Fatalf("unexpected first position event: %+v", firstEvent)
	}
	secondEvent, ok := publisher.calls[1].Payload.(kafka.PositionChangedEvent)
	if !ok {
		t.Fatalf("expected second publish to be position event, got %#v", publisher.calls[1].Payload)
	}
	if secondEvent.Outcome != "NO" || secondEvent.DeltaQuantity != 40 {
		t.Fatalf("unexpected second position event: %+v", secondEvent)
	}
	command, ok := publisher.calls[2].Payload.(kafka.OrderCommand)
	if !ok {
		t.Fatalf("expected third publish to be order command, got %#v", publisher.calls[2].Payload)
	}
	if command.OrderID != reqBody.BootstrapOrderID(88) || command.FreezeID != "frz_bootstrap_88" || command.FreezeAmount != 40 || command.Outcome != "YES" || command.Side != "SELL" {
		t.Fatalf("unexpected first-liquidity bootstrap command: %+v", command)
	}
	if !strings.Contains(w.Body.String(), "\"status\":\"ISSUED\"") {
		t.Fatalf("unexpected response body: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"collateral_debit\":4000") {
		t.Fatalf("expected response to expose 100x collateral debit, got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"order_id\":\""+reqBody.BootstrapOrderID(88)+"\"") || !strings.Contains(w.Body.String(), "\"order_status\":\"QUEUED\"") {
		t.Fatalf("expected response to expose queued bootstrap order, got %s", w.Body.String())
	}
}

func TestCreateFirstLiquidityRejectsSemanticDuplicateBeforeInventoryMutation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reqBody := dto.CreateFirstLiquidityRequest{
		UserID:   1002,
		Quantity: 40,
		Outcome:  "YES",
		Price:    55,
	}
	wallet := attachSignedFirstLiquidityOperator(t, 88, &reqBody)
	orderID := reqBody.BootstrapOrderID(88)

	account := &fakeAccountClient{}
	publisher := &fakePublisher{}
	handler := NewOrderHandler(Dependencies{
		Logger:         slog.Default(),
		KafkaPublisher: publisher,
		KafkaTopics:    kafka.NewTopics("funnyoption."),
		AccountClient:  account,
		QueryStore: &fakeQueryStore{
			getOrderResp: dto.OrderResponse{
				OrderID:  orderID,
				UserID:   1002,
				MarketID: 88,
				Outcome:  "YES",
				Side:     "SELL",
			},
		},
		OperatorWallets: []string{wallet},
	})

	raw, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/markets/88/first-liquidity", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "market_id", Value: "88"}}
	ctx.Request = req

	handler.CreateFirstLiquidity(ctx)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "operator bootstrap order already accepted") || !strings.Contains(w.Body.String(), orderID) {
		t.Fatalf("unexpected duplicate response body: %s", w.Body.String())
	}
	if len(account.debits) != 0 || len(account.credits) != 0 || account.preFreezeCalled {
		t.Fatalf("expected duplicate first-liquidity request to stop before balance mutations, got debits=%+v credits=%+v preFreezeCalled=%v", account.debits, account.credits, account.preFreezeCalled)
	}
	if len(publisher.calls) != 0 {
		t.Fatalf("expected duplicate first-liquidity request to publish nothing, got %+v", publisher.calls)
	}
}

func TestCreateFirstLiquidityRollsBackWhenPublishFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	account := &fakeAccountClient{
		debitResults: []accountclient.DebitResult{
			{Applied: true, Asset: "USDT", UserID: 1002},
			{Applied: true, Asset: "POSITION:99:YES", UserID: 1002},
		},
		creditResults: []accountclient.CreditResult{
			{Applied: true, Asset: "POSITION:99:YES", UserID: 1002},
			{Applied: true, Asset: "USDT", UserID: 1002},
		},
	}
	publisher := &fakePublisher{
		err:   errors.New("kafka down"),
		errAt: 1,
	}
	reqBody := dto.CreateFirstLiquidityRequest{
		UserID:   1002,
		Quantity: 40,
		Outcome:  "YES",
		Price:    55,
	}
	wallet := attachSignedFirstLiquidityOperator(t, 99, &reqBody)
	handler := NewOrderHandler(Dependencies{
		Logger:         slog.Default(),
		KafkaPublisher: publisher,
		KafkaTopics:    kafka.NewTopics("funnyoption."),
		AccountClient:  account,
		QueryStore: &fakeQueryStore{
			getMarketResp: dto.MarketResponse{
				MarketID:        99,
				CollateralAsset: "USDT",
				Status:          "OPEN",
			},
		},
		OperatorWallets: []string{wallet},
	})

	raw, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/markets/99/first-liquidity", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "market_id", Value: "99"}}
	ctx.Request = req

	handler.CreateFirstLiquidity(ctx)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d body=%s", w.Code, w.Body.String())
	}
	if len(account.debits) != 2 {
		t.Fatalf("expected collateral debit plus rollback debit, got %+v", account.debits)
	}
	if got := account.debits[0]; got.RefType != "FIRST_LIQUIDITY_COLLATERAL" || got.Asset != "USDT" || got.Amount != 4000 {
		t.Fatalf("unexpected collateral debit before rollback: %+v", got)
	}
	if got := account.debits[1]; got.RefType != "FIRST_LIQUIDITY_POSITION_ROLLBACK" || got.Asset != "POSITION:99:YES" {
		t.Fatalf("unexpected rollback debit: %+v", got)
	}
	if len(account.credits) != 2 {
		t.Fatalf("expected initial inventory credit plus collateral rollback credit, got %+v", account.credits)
	}
	if got := account.credits[1]; got.RefType != "FIRST_LIQUIDITY_COLLATERAL_ROLLBACK" || got.Asset != "USDT" || got.Amount != 4000 {
		t.Fatalf("unexpected collateral rollback credit: %+v", got)
	}
	if len(publisher.calls) != 1 {
		t.Fatalf("expected only the failed forward publish, got %+v", publisher.calls)
	}
}

func TestCreateFirstLiquidityRejectsPastCloseAtMarket(t *testing.T) {
	gin.SetMode(gin.TestMode)

	account := &fakeAccountClient{}
	reqBody := dto.CreateFirstLiquidityRequest{
		UserID:   1002,
		Quantity: 40,
		Outcome:  "YES",
		Price:    55,
	}
	wallet := attachSignedFirstLiquidityOperator(t, 88, &reqBody)
	handler := NewOrderHandler(Dependencies{
		Logger:         slog.Default(),
		KafkaPublisher: &fakePublisher{},
		KafkaTopics:    kafka.NewTopics("funnyoption."),
		AccountClient:  account,
		QueryStore: &fakeQueryStore{
			getMarketResp: dto.MarketResponse{
				MarketID:        88,
				CollateralAsset: "USDT",
				Status:          "OPEN",
				CloseAt:         time.Now().Unix() - 1,
			},
		},
		OperatorWallets: []string{wallet},
	})

	raw, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/markets/88/first-liquidity", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "market_id", Value: "88"}}
	ctx.Request = req

	handler.CreateFirstLiquidity(ctx)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", w.Code, w.Body.String())
	}
	if len(account.debits) != 0 || len(account.credits) != 0 || account.preFreezeCalled {
		t.Fatalf("expected post-close first-liquidity request to stop before balance mutation, got debits=%+v credits=%+v preFreezeCalled=%v", account.debits, account.credits, account.preFreezeCalled)
	}
}

func TestCreateFirstLiquidityRejectsFrozenRollup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	account := &fakeAccountClient{}
	reqBody := dto.CreateFirstLiquidityRequest{
		UserID:   1002,
		Quantity: 40,
		Outcome:  "YES",
		Price:    55,
	}
	wallet := attachSignedFirstLiquidityOperator(t, 88, &reqBody)
	handler := NewOrderHandler(Dependencies{
		Logger:         slog.Default(),
		KafkaPublisher: &fakePublisher{},
		KafkaTopics:    kafka.NewTopics("funnyoption."),
		AccountClient:  account,
		QueryStore: &fakeQueryStore{
			getRollupFreezeResp: dto.RollupFreezeStateResponse{Frozen: true},
		},
		OperatorWallets: []string{wallet},
	})

	raw, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/markets/88/first-liquidity", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "market_id", Value: "88"}}
	ctx.Request = req

	handler.CreateFirstLiquidity(ctx)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "rollup is frozen") {
		t.Fatalf("expected frozen rollup error, got %s", w.Body.String())
	}
	if len(account.debits) != 0 || len(account.credits) != 0 || account.preFreezeCalled {
		t.Fatalf("expected frozen first-liquidity request to stop before balance mutation, got debits=%+v credits=%+v preFreezeCalled=%v", account.debits, account.credits, account.preFreezeCalled)
	}
}

func TestCreateFirstLiquidityRejectsMissingOperatorProof(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewOrderHandler(Dependencies{
		Logger:          slog.Default(),
		QueryStore:      &fakeQueryStore{},
		OperatorWallets: []string{"0x1234"},
	})

	body := map[string]any{
		"user_id":  1002,
		"quantity": 40,
		"outcome":  "YES",
		"price":    55,
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/markets/99/first-liquidity", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "market_id", Value: "99"}}
	ctx.Request = req

	handler.CreateFirstLiquidity(ctx)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestResolveMarketRejectsMissingOperatorProof(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewOrderHandler(Dependencies{
		Logger:          slog.Default(),
		KafkaPublisher:  &fakePublisher{},
		KafkaTopics:     kafka.NewTopics("funnyoption."),
		OperatorWallets: []string{"0x1234"},
	})

	body := map[string]any{
		"outcome": "YES",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/markets/88/resolve", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "market_id", Value: "88"}}
	ctx.Request = req

	handler.ResolveMarket(ctx)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestResolveMarketRejectsAlreadyResolvedMarket(t *testing.T) {
	gin.SetMode(gin.TestMode)

	publisher := &fakePublisher{}
	reqBody := dto.ResolveMarketRequest{Outcome: "YES"}
	wallet := attachSignedResolveOperator(t, 88, &reqBody)
	handler := NewOrderHandler(Dependencies{
		Logger:         slog.Default(),
		KafkaPublisher: publisher,
		KafkaTopics:    kafka.NewTopics("funnyoption."),
		QueryStore: &fakeQueryStore{
			getMarketResp: dto.MarketResponse{
				MarketID: 88,
				Status:   "RESOLVED",
			},
		},
		OperatorWallets: []string{wallet},
	})

	raw, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/markets/88/resolve", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "market_id", Value: "88"}}
	ctx.Request = req

	handler.ResolveMarket(ctx)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", w.Code, w.Body.String())
	}
	if len(publisher.calls) != 0 {
		t.Fatalf("expected resolved market to stop before publish, got %+v", publisher.calls)
	}
}

func TestResolveMarketRejectsFrozenRollup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	publisher := &fakePublisher{}
	reqBody := dto.ResolveMarketRequest{Outcome: "YES"}
	wallet := attachSignedResolveOperator(t, 88, &reqBody)
	handler := NewOrderHandler(Dependencies{
		Logger:         slog.Default(),
		KafkaPublisher: publisher,
		KafkaTopics:    kafka.NewTopics("funnyoption."),
		QueryStore: &fakeQueryStore{
			getRollupFreezeResp: dto.RollupFreezeStateResponse{Frozen: true},
		},
		OperatorWallets: []string{wallet},
	})

	raw, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/markets/88/resolve", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "market_id", Value: "88"}}
	ctx.Request = req

	handler.ResolveMarket(ctx)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "rollup is frozen") {
		t.Fatalf("expected frozen rollup error, got %s", w.Body.String())
	}
	if len(publisher.calls) != 0 {
		t.Fatalf("expected frozen resolve to publish nothing, got %+v", publisher.calls)
	}
}

func TestResolveMarketPublishesEventForAuthorizedOperator(t *testing.T) {
	gin.SetMode(gin.TestMode)

	publisher := &fakePublisher{}
	nowUnix := time.Now().Unix()
	store := &fakeQueryStore{
		getMarketResp: dto.MarketResponse{
			MarketID:  88,
			Status:    "OPEN",
			CloseAt:   nowUnix - 120,
			ResolveAt: nowUnix - 60,
		},
	}
	reqBody := dto.ResolveMarketRequest{
		Outcome: "YES",
	}
	wallet := attachSignedResolveOperator(t, 88, &reqBody)
	handler := NewOrderHandler(Dependencies{
		Logger:          slog.Default(),
		KafkaPublisher:  publisher,
		KafkaTopics:     kafka.NewTopics("funnyoption."),
		QueryStore:      store,
		OperatorWallets: []string{wallet},
	})

	raw, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/markets/88/resolve", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "market_id", Value: "88"}}
	ctx.Request = req

	handler.ResolveMarket(ctx)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", w.Code, w.Body.String())
	}
	if len(publisher.calls) != 1 {
		t.Fatalf("expected one market event, got %+v", publisher.calls)
	}
	event, ok := publisher.calls[0].Payload.(kafka.MarketEvent)
	if !ok {
		t.Fatalf("expected market event payload, got %#v", publisher.calls[0].Payload)
	}
	if event.MarketID != 88 || event.ResolvedOutcome != "YES" {
		t.Fatalf("unexpected market event: %+v", event)
	}
}

func TestResolveMarketRejectsBeforeWaitingResolution(t *testing.T) {
	gin.SetMode(gin.TestMode)

	publisher := &fakePublisher{}
	nowUnix := time.Now().Unix()
	store := &fakeQueryStore{
		getMarketResp: dto.MarketResponse{
			MarketID:  88,
			Status:    "OPEN",
			CloseAt:   nowUnix - 120,
			ResolveAt: nowUnix + 600,
		},
	}
	reqBody := dto.ResolveMarketRequest{Outcome: "YES"}
	wallet := attachSignedResolveOperator(t, 88, &reqBody)
	handler := NewOrderHandler(Dependencies{
		Logger:          slog.Default(),
		KafkaPublisher:  publisher,
		KafkaTopics:     kafka.NewTopics("funnyoption."),
		QueryStore:      store,
		OperatorWallets: []string{wallet},
	})

	raw, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/markets/88/resolve", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "market_id", Value: "88"}}
	ctx.Request = req

	handler.ResolveMarket(ctx)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "market is not waiting for resolution") {
		t.Fatalf("expected waiting-resolution conflict, got %s", w.Body.String())
	}
	if len(publisher.calls) != 0 {
		t.Fatalf("expected no market event publish before adjudication window, got %+v", publisher.calls)
	}
}

func TestResolveMarketRejectsOracleMarketFromManualLane(t *testing.T) {
	gin.SetMode(gin.TestMode)

	publisher := &fakePublisher{}
	nowUnix := time.Now().Unix()
	store := &fakeQueryStore{
		getMarketResp: dto.MarketResponse{
			MarketID:  88,
			Status:    "OPEN",
			CloseAt:   nowUnix - 120,
			ResolveAt: nowUnix - 60,
			Metadata:  validOracleResolutionMetadata(),
			Options:   dto.DefaultBinaryMarketOptions(),
			Category: &dto.MarketCategory{
				CategoryKey: "CRYPTO",
			},
		},
	}
	reqBody := dto.ResolveMarketRequest{
		Outcome: "YES",
	}
	wallet := attachSignedResolveOperator(t, 88, &reqBody)
	handler := NewOrderHandler(Dependencies{
		Logger:          slog.Default(),
		KafkaPublisher:  publisher,
		KafkaTopics:     kafka.NewTopics("funnyoption."),
		QueryStore:      store,
		OperatorWallets: []string{wallet},
	})

	raw, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/markets/88/resolve", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "market_id", Value: "88"}}
	ctx.Request = req

	handler.ResolveMarket(ctx)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "oracle market must resolve through oracle worker") {
		t.Fatalf("expected oracle-lane conflict, got %s", w.Body.String())
	}
	if len(publisher.calls) != 0 {
		t.Fatalf("expected no market event publish for oracle market, got %+v", publisher.calls)
	}
}

func TestGetMarketReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakeQueryStore{getMarketErr: ErrNotFound}
	handler := NewOrderHandler(Dependencies{
		Logger:     slog.Default(),
		QueryStore: store,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/markets/42", nil)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "market_id", Value: "42"}}
	ctx.Request = req

	handler.GetMarket(ctx)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestListTradesReturnsEmptyArrayForNilSlice(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewOrderHandler(Dependencies{
		Logger:     slog.Default(),
		QueryStore: &fakeQueryStore{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trades?market_id=88", nil)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.ListTrades(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"items\":[]") {
		t.Fatalf("expected empty array response, got %s", w.Body.String())
	}

	var resp struct {
		Items []dto.TradeResponse `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Items == nil {
		t.Fatalf("expected non-nil items slice, got nil body=%s", w.Body.String())
	}
	if len(resp.Items) != 0 {
		t.Fatalf("expected empty items slice, got %+v", resp.Items)
	}
}

func TestListChainTransactionsReturnsEmptyArrayForNilSlice(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewOrderHandler(Dependencies{
		Logger:     slog.Default(),
		QueryStore: &fakeQueryStore{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chain-transactions?limit=5", nil)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.ListChainTransactions(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"items\":[]") {
		t.Fatalf("expected empty array response, got %s", w.Body.String())
	}

	var resp struct {
		Items []dto.ChainTransactionResponse `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Items == nil {
		t.Fatalf("expected non-nil items slice, got nil body=%s", w.Body.String())
	}
	if len(resp.Items) != 0 {
		t.Fatalf("expected empty items slice, got %+v", resp.Items)
	}
}

func TestListRollupForcedWithdrawalsReturnsItems(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakeQueryStore{
		listRollupForcedWithdrawResp: []dto.RollupForcedWithdrawalResponse{
			{RequestID: 1, Status: "REQUESTED", SatisfactionStatus: "READY"},
		},
	}
	handler := NewOrderHandler(Dependencies{
		Logger:     slog.Default(),
		QueryStore: store,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rollup/forced-withdrawals?limit=5", nil)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.ListRollupForcedWithdrawals(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"request_id\":1") {
		t.Fatalf("expected request payload, got %s", w.Body.String())
	}
}

func TestGetRollupFreezeStateReturnsObject(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewOrderHandler(Dependencies{
		Logger: slog.Default(),
		QueryStore: &fakeQueryStore{
			getRollupFreezeResp: dto.RollupFreezeStateResponse{
				Frozen:    true,
				FrozenAt:  1234,
				RequestID: 7,
				UpdatedAt: 1235,
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rollup/freeze-state", nil)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.GetRollupFreezeState(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"frozen\":true") {
		t.Fatalf("expected frozen response, got %s", w.Body.String())
	}
}

func TestListBalancesReturnsItems(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakeQueryStore{
		listBalancesResp: []dto.BalanceResponse{
			{UserID: 1001, Asset: "USDT", Available: 900, Frozen: 100},
		},
	}
	handler := NewOrderHandler(Dependencies{
		Logger:     slog.Default(),
		QueryStore: store,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/balances?user_id=1001", nil)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.ListBalances(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"asset\":\"USDT\"") {
		t.Fatalf("unexpected response body: %s", w.Body.String())
	}
}

func TestGetLiabilityReportReturnsItems(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakeQueryStore{
		liabilityResp: []dto.LiabilityReportLine{
			{Asset: "USDT", UserAvailable: 900, UserFrozen: 100, InternalTotal: 1000},
		},
	}
	handler := NewOrderHandler(Dependencies{
		Logger:     slog.Default(),
		QueryStore: store,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/liabilities", nil)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	handler.GetLiabilityReport(ctx)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"internal_total\":1000") {
		t.Fatalf("unexpected response body: %s", w.Body.String())
	}
}
