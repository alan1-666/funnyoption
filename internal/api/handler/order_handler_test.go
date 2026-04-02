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
	"github.com/gin-gonic/gin"
)

type fakeAccountClient struct {
	freezeResp      accountclient.FreezeRecord
	freezeErr       error
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
	_ = req
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
	createMarketResp   dto.MarketResponse
	createMarketErr    error
	createMarketReq    dto.CreateMarketRequest
	createSessionResp  dto.SessionResponse
	createSessionErr   error
	createClaimResp    dto.ChainTransactionResponse
	createClaimErr     error
	createClaimReq     dto.ClaimPayoutRequest
	createClaimCalled  bool
	getSessionResp     dto.SessionResponse
	getSessionErr      error
	revokeSessionResp  dto.SessionResponse
	revokeSessionErr   error
	advanceSessionResp dto.SessionResponse
	advanceSessionErr  error
	getMarketResp      dto.MarketResponse
	getMarketErr       error
	listMarketsResp    []dto.MarketResponse
	listMarketsErr     error
	listSessionsResp   []dto.SessionResponse
	listSessionsErr    error
	listDepositsResp   []dto.DepositResponse
	listDepositsErr    error
	listWithdrawResp   []dto.WithdrawalResponse
	listWithdrawErr    error
	listChainTxResp    []dto.ChainTransactionResponse
	listChainTxErr     error
	listOrdersResp     []dto.OrderResponse
	listOrdersErr      error
	listTradesResp     []dto.TradeResponse
	listTradesErr      error
	listBalancesResp   []dto.BalanceResponse
	listBalancesErr    error
	listPositionsResp  []dto.PositionResponse
	listPositionsErr   error
	listPayoutsResp    []dto.PayoutResponse
	listPayoutsErr     error
	listFreezesResp    []dto.FreezeResponse
	listFreezesErr     error
	listEntriesResp    []dto.LedgerEntryResponse
	listEntriesErr     error
	listPostingsResp   []dto.LedgerPostingResponse
	listPostingsErr    error
	liabilityResp      []dto.LiabilityReportLine
	liabilityErr       error
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
	wallet := sharedauth.NormalizeHex(crypto.PubkeyToAddress(key.PublicKey).Hex())
	req.Operator = &dto.OperatorAction{
		WalletAddress: wallet,
		RequestedAt:   time.Now().UnixMilli(),
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

func (f *fakeQueryStore) RevokeSession(ctx context.Context, sessionID string) (dto.SessionResponse, error) {
	_ = ctx
	_ = sessionID
	if f.revokeSessionResp.SessionID == "" {
		return f.getSessionResp, f.revokeSessionErr
	}
	return f.revokeSessionResp, f.revokeSessionErr
}

func (f *fakeQueryStore) AdvanceSessionNonce(ctx context.Context, sessionID string, nonce uint64) (dto.SessionResponse, error) {
	_ = ctx
	_ = sessionID
	_ = nonce
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
	_ = req
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
			SessionID:        "sess_live",
			UserID:           1001,
			WalletAddress:    "0x00000000000000000000000000000000000000aa",
			SessionPublicKey: hexutil.Encode(sessionPub),
			Scope:            "TRADE",
			ChainID:          97,
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
		SessionID:         "sess_live",
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
	handler = NewOrderHandler(Dependencies{
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

func TestCreateFirstLiquidityIssuesPairedInventory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	account := &fakeAccountClient{
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
	if got := account.debits[0]; got.Asset != "USDT" || got.Amount != 40 || got.RefType != "FIRST_LIQUIDITY_COLLATERAL" {
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
	if len(publisher.calls) != 2 {
		t.Fatalf("expected 2 position events, got %+v", publisher.calls)
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
	if !strings.Contains(w.Body.String(), "\"status\":\"ISSUED\"") {
		t.Fatalf("unexpected response body: %s", w.Body.String())
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
	if got := account.debits[1]; got.RefType != "FIRST_LIQUIDITY_POSITION_ROLLBACK" || got.Asset != "POSITION:99:YES" {
		t.Fatalf("unexpected rollback debit: %+v", got)
	}
	if len(account.credits) != 2 {
		t.Fatalf("expected initial inventory credit plus collateral rollback credit, got %+v", account.credits)
	}
	if got := account.credits[1]; got.RefType != "FIRST_LIQUIDITY_COLLATERAL_ROLLBACK" || got.Asset != "USDT" {
		t.Fatalf("unexpected collateral rollback credit: %+v", got)
	}
	if len(publisher.calls) != 1 {
		t.Fatalf("expected only the failed forward publish, got %+v", publisher.calls)
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

func TestResolveMarketPublishesEventForAuthorizedOperator(t *testing.T) {
	gin.SetMode(gin.TestMode)

	publisher := &fakePublisher{}
	reqBody := dto.ResolveMarketRequest{
		Outcome: "YES",
	}
	wallet := attachSignedResolveOperator(t, 88, &reqBody)
	handler := NewOrderHandler(Dependencies{
		Logger:          slog.Default(),
		KafkaPublisher:  publisher,
		KafkaTopics:     kafka.NewTopics("funnyoption."),
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
