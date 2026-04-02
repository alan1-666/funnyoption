package api

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	accountclient "funnyoption/internal/account/client"
	"funnyoption/internal/api/dto"
	"funnyoption/internal/api/handler"
	sharedauth "funnyoption/internal/shared/auth"
	"funnyoption/internal/shared/kafka"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
)

func TestEnginePublicReadRouteReturnsMarkets(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := newEngine(Meta{Service: "api", Env: "test"}, handler.Dependencies{
		Logger: slog.Default(),
		QueryStore: &testQueryStore{
			listMarketsResp: []dto.MarketResponse{
				{MarketID: 88, Title: "BTC Above 100k", Status: "OPEN", CollateralAsset: "USDT"},
			},
		},
	}, routerOptions{
		rateLimiter: newRateLimiter(defaultRateLimitPolicies()),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/markets", nil)
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "\"market_id\":88") {
		t.Fatalf("expected market in body, got %s", w.Body.String())
	}
}

func TestEngineTradeWriteSupportsSessionSignedOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sessionPub, sessionPriv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}

	engine := newEngine(Meta{Service: "api", Env: "test"}, handler.Dependencies{
		Logger:         slog.Default(),
		KafkaPublisher: &testPublisher{},
		KafkaTopics:    kafka.NewTopics("funnyoption."),
		AccountClient: &testAccountClient{
			freezeResp: accountclient.FreezeRecord{
				FreezeID: "frz_session",
				UserID:   1001,
				Asset:    "USDT",
				RefType:  "ORDER",
				RefID:    "ord_x",
				Amount:   200,
			},
		},
		QueryStore: &testQueryStore{
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
			getMarketResp: dto.MarketResponse{
				MarketID:        88,
				Status:          "OPEN",
				CollateralAsset: "USDT",
			},
		},
	}, routerOptions{
		rateLimiter: newRateLimiter(defaultRateLimitPolicies()),
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

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestEngineTradeWriteSupportsOperatorBootstrapOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)

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
	wallet := attachSignedRouterBootstrapOrderOperator(t, &reqBody)

	engine := newEngine(Meta{Service: "api", Env: "test"}, handler.Dependencies{
		Logger:         slog.Default(),
		KafkaPublisher: &testPublisher{},
		KafkaTopics:    kafka.NewTopics("funnyoption."),
		AccountClient: &testAccountClient{
			freezeResp: accountclient.FreezeRecord{
				FreezeID: "frz_operator",
				UserID:   1001,
				Asset:    "POSITION:88:YES",
				RefType:  "ORDER",
				RefID:    "ord_x",
				Amount:   20,
			},
		},
		QueryStore: &testQueryStore{
			getMarketResp: dto.MarketResponse{
				MarketID:        88,
				Status:          "OPEN",
				CollateralAsset: "USDT",
			},
		},
		OperatorWallets: []string{wallet},
	}, routerOptions{
		rateLimiter: newRateLimiter(defaultRateLimitPolicies()),
	})

	raw, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestEngineTradeWriteRejectsBareUserIDWithoutAuthEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := newEngine(Meta{Service: "api", Env: "test"}, handler.Dependencies{
		Logger: slog.Default(),
	}, routerOptions{
		rateLimiter: newRateLimiter(defaultRateLimitPolicies()),
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

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "session-backed trade authorization or operator proof is required") {
		t.Fatalf("unexpected response body: %s", w.Body.String())
	}
}

func TestEnginePrivilegedRouteRequiresOperatorEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := newEngine(Meta{Service: "api", Env: "test"}, handler.Dependencies{
		Logger:          slog.Default(),
		QueryStore:      &testQueryStore{},
		OperatorWallets: []string{"0x1234"},
	}, routerOptions{
		rateLimiter: newRateLimiter(defaultRateLimitPolicies()),
	})

	body := map[string]any{"title": "BTC Above 100k"}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/markets", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "operator proof is required") {
		t.Fatalf("unexpected response body: %s", w.Body.String())
	}
}

func TestEngineRateLimitsSessionCreate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	policies := defaultRateLimitPolicies()
	policies[rateLimitSessionCreate] = rateLimitPolicy{
		Limit: requestsPerWindow(1, time.Hour),
		Burst: 1,
		KeyFn: func(ctx *gin.Context) string { return clientIdentifier(ctx) },
		Label: "session create",
	}

	engine := newEngine(Meta{Service: "api", Env: "test"}, handler.Dependencies{
		Logger:     slog.Default(),
		QueryStore: &testQueryStore{},
	}, routerOptions{
		rateLimiter: newRateLimiter(policies),
	})

	body := validSignedSessionBody(t)
	raw, _ := json.Marshal(body)

	firstReq := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", bytes.NewReader(raw))
	firstReq.Header.Set("Content-Type", "application/json")
	firstReq.RemoteAddr = "198.51.100.10:1234"
	firstResp := httptest.NewRecorder()
	engine.ServeHTTP(firstResp, firstReq)
	if firstResp.Code != http.StatusCreated {
		t.Fatalf("expected first request 201, got %d body=%s", firstResp.Code, firstResp.Body.String())
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", bytes.NewReader(raw))
	secondReq.Header.Set("Content-Type", "application/json")
	secondReq.RemoteAddr = "198.51.100.10:9999"
	secondResp := httptest.NewRecorder()
	engine.ServeHTTP(secondResp, secondReq)
	if secondResp.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request 429, got %d body=%s", secondResp.Code, secondResp.Body.String())
	}
}

func validSignedSessionBody(t *testing.T) map[string]any {
	t.Helper()

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

	return map[string]any{
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
}

func signRouterOperatorMessage(t *testing.T, key *ecdsa.PrivateKey, message string) string {
	t.Helper()

	signature, err := crypto.Sign(accounts.TextHash([]byte(message)), key)
	if err != nil {
		t.Fatalf("Sign returned error: %v", err)
	}
	return hexutil.Encode(signature)
}

func attachSignedRouterBootstrapOrderOperator(t *testing.T, req *dto.CreateOrderRequest) string {
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
	req.Operator.Signature = signRouterOperatorMessage(t, key, req.BootstrapOperatorMessage())
	return wallet
}

type testAccountClient struct {
	freezeResp accountclient.FreezeRecord
	freezeErr  error
}

func (c *testAccountClient) PreFreeze(ctx context.Context, req accountclient.FreezeRequest) (accountclient.FreezeRecord, error) {
	_ = ctx
	_ = req
	return c.freezeResp, c.freezeErr
}

func (c *testAccountClient) ReleaseFreeze(ctx context.Context, freezeID string) error {
	_ = ctx
	_ = freezeID
	return nil
}

func (c *testAccountClient) GetBalance(ctx context.Context, userID int64, asset string) (accountclient.Balance, error) {
	_ = ctx
	_ = userID
	_ = asset
	return accountclient.Balance{}, nil
}

func (c *testAccountClient) CreditBalance(ctx context.Context, userID int64, asset string, amount int64, refType, refID string) (accountclient.CreditResult, error) {
	_ = ctx
	_ = userID
	_ = asset
	_ = amount
	_ = refType
	_ = refID
	return accountclient.CreditResult{Applied: true}, nil
}

func (c *testAccountClient) DebitBalance(ctx context.Context, userID int64, asset string, amount int64, refType, refID string) (accountclient.DebitResult, error) {
	_ = ctx
	_ = userID
	_ = asset
	_ = amount
	_ = refType
	_ = refID
	return accountclient.DebitResult{Applied: true}, nil
}

func (c *testAccountClient) Close() error {
	return nil
}

type testPublisher struct{}

func (p *testPublisher) PublishJSON(ctx context.Context, topic, key string, payload any) error {
	_ = ctx
	_ = topic
	_ = key
	_ = payload
	return nil
}

func (p *testPublisher) Close() error {
	return nil
}

type testQueryStore struct {
	listMarketsResp   []dto.MarketResponse
	getSessionResp    dto.SessionResponse
	getMarketResp     dto.MarketResponse
	createSessionResp dto.SessionResponse
}

func (s *testQueryStore) CreateMarket(ctx context.Context, req dto.CreateMarketRequest) (dto.MarketResponse, error) {
	_ = ctx
	_ = req
	return dto.MarketResponse{}, nil
}

func (s *testQueryStore) CreateSession(ctx context.Context, req dto.CreateSessionRequest) (dto.SessionResponse, error) {
	_ = ctx
	if s.createSessionResp.SessionID != "" {
		return s.createSessionResp, nil
	}
	return dto.SessionResponse{
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
	}, nil
}

func (s *testQueryStore) CreateClaimRequest(ctx context.Context, req dto.ClaimPayoutRequest) (dto.ChainTransactionResponse, error) {
	_ = ctx
	_ = req
	return dto.ChainTransactionResponse{}, nil
}

func (s *testQueryStore) GetSession(ctx context.Context, sessionID string) (dto.SessionResponse, error) {
	_ = ctx
	_ = sessionID
	return s.getSessionResp, nil
}

func (s *testQueryStore) RevokeSession(ctx context.Context, sessionID string) (dto.SessionResponse, error) {
	_ = ctx
	_ = sessionID
	return dto.SessionResponse{}, nil
}

func (s *testQueryStore) AdvanceSessionNonce(ctx context.Context, sessionID string, nonce uint64) (dto.SessionResponse, error) {
	_ = ctx
	_ = sessionID
	_ = nonce
	return dto.SessionResponse{}, nil
}

func (s *testQueryStore) GetMarket(ctx context.Context, marketID int64) (dto.MarketResponse, error) {
	_ = ctx
	_ = marketID
	return s.getMarketResp, nil
}

func (s *testQueryStore) ListMarkets(ctx context.Context, req dto.ListMarketsRequest) ([]dto.MarketResponse, error) {
	_ = ctx
	_ = req
	return s.listMarketsResp, nil
}

func (s *testQueryStore) ListSessions(ctx context.Context, req dto.ListSessionsRequest) ([]dto.SessionResponse, error) {
	_ = ctx
	_ = req
	return nil, nil
}

func (s *testQueryStore) ListDeposits(ctx context.Context, req dto.ListDepositsRequest) ([]dto.DepositResponse, error) {
	_ = ctx
	_ = req
	return nil, nil
}

func (s *testQueryStore) ListWithdrawals(ctx context.Context, req dto.ListWithdrawalsRequest) ([]dto.WithdrawalResponse, error) {
	_ = ctx
	_ = req
	return nil, nil
}

func (s *testQueryStore) ListChainTransactions(ctx context.Context, req dto.ListChainTransactionsRequest) ([]dto.ChainTransactionResponse, error) {
	_ = ctx
	_ = req
	return nil, nil
}

func (s *testQueryStore) ListOrders(ctx context.Context, req dto.ListOrdersRequest) ([]dto.OrderResponse, error) {
	_ = ctx
	_ = req
	return nil, nil
}

func (s *testQueryStore) ListTrades(ctx context.Context, req dto.ListTradesRequest) ([]dto.TradeResponse, error) {
	_ = ctx
	_ = req
	return nil, nil
}

func (s *testQueryStore) ListBalances(ctx context.Context, req dto.ListBalancesRequest) ([]dto.BalanceResponse, error) {
	_ = ctx
	_ = req
	return nil, nil
}

func (s *testQueryStore) ListPositions(ctx context.Context, req dto.ListPositionsRequest) ([]dto.PositionResponse, error) {
	_ = ctx
	_ = req
	return nil, nil
}

func (s *testQueryStore) ListPayouts(ctx context.Context, req dto.ListPayoutsRequest) ([]dto.PayoutResponse, error) {
	_ = ctx
	_ = req
	return nil, nil
}

func (s *testQueryStore) ListFreezes(ctx context.Context, req dto.ListFreezesRequest) ([]dto.FreezeResponse, error) {
	_ = ctx
	_ = req
	return nil, nil
}

func (s *testQueryStore) ListLedgerEntries(ctx context.Context, req dto.ListLedgerEntriesRequest) ([]dto.LedgerEntryResponse, error) {
	_ = ctx
	_ = req
	return nil, nil
}

func (s *testQueryStore) ListLedgerPostings(ctx context.Context, entryID string) ([]dto.LedgerPostingResponse, error) {
	_ = ctx
	_ = entryID
	return nil, nil
}

func (s *testQueryStore) BuildLiabilityReport(ctx context.Context) ([]dto.LiabilityReportLine, error) {
	_ = ctx
	return nil, nil
}
