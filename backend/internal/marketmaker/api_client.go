package marketmaker

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
)

// OperatorAPIClient calls the platform API with operator authentication.
type OperatorAPIClient struct {
	baseURL    string
	httpClient *http.Client
	privateKey string
	wallet     string
	botUserID  int64
	writePacer *writePacer
}

func NewOperatorAPIClient(cfg Config) *OperatorAPIClient {
	return &OperatorAPIClient{
		baseURL:    strings.TrimRight(cfg.APIURL, "/"),
		httpClient: &http.Client{Timeout: 15 * time.Second},
		privateKey: cfg.OperatorPrivateKey,
		wallet:     cfg.OperatorWallet,
		botUserID:  cfg.BotUserID,
		writePacer: newWritePacer(cfg.WriteInterval),
	}
}

type operatorAction struct {
	WalletAddress string `json:"wallet_address"`
	RequestedAt   int64  `json:"requested_at"`
	Signature     string `json:"signature"`
}

type firstLiquidityRequest struct {
	UserID   int64           `json:"user_id"`
	Quantity int64           `json:"quantity"`
	Outcome  string          `json:"outcome"`
	Price    int64           `json:"price"`
	Operator *operatorAction `json:"operator"`
}

type firstLiquidityResponse struct {
	FirstLiquidityID string `json:"first_liquidity_id"`
	OrderID          string `json:"order_id,omitempty"`
	Status           string `json:"status"`
}

type createOrderRequest struct {
	UserID      int64           `json:"user_id"`
	MarketID    int64           `json:"market_id"`
	Outcome     string          `json:"outcome"`
	Side        string          `json:"side"`
	Type        string          `json:"type"`
	TimeInForce string          `json:"time_in_force"`
	Price       int64           `json:"price"`
	Quantity    int64           `json:"quantity"`
	Operator    *operatorAction `json:"operator"`
}

type createOrderResponse struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
}

type marketResponse struct {
	MarketID        int64  `json:"market_id"`
	Title           string `json:"title"`
	Status          string `json:"status"`
	CollateralAsset string `json:"collateral_asset"`
	OpenAt          int64  `json:"open_at"`
	CloseAt         int64  `json:"close_at"`
}

type marketsListResponse struct {
	Items []marketResponse `json:"items"`
}

// ListOpenMarkets returns all OPEN markets from the API.
func (c *OperatorAPIClient) ListOpenMarkets(ctx context.Context) ([]marketResponse, error) {
	resp, err := c.get(ctx, "/v1/markets?status=OPEN")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list markets: HTTP %d", resp.StatusCode)
	}
	var result marketsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Items, nil
}

// SeedFirstLiquidity calls the admin first-liquidity endpoint to mint complete
// sets and place a bootstrap SELL order on the specified outcome.
func (c *OperatorAPIClient) SeedFirstLiquidity(ctx context.Context, marketID, quantity, price int64, outcome string) (*firstLiquidityResponse, error) {
	now := time.Now().UnixMilli()
	message := fmt.Sprintf(
		"FunnyOption Operator Authorization\n\naction: ISSUE_FIRST_LIQUIDITY\nwallet: %s\nmarket_id: %d\nuser_id: %d\nquantity: %d\noutcome: %s\nprice: %d\nrequested_at: %d\n",
		strings.ToLower(c.wallet), marketID, c.botUserID, quantity, outcome, price, now,
	)
	sig, err := c.signPersonalMessage(message)
	if err != nil {
		return nil, fmt.Errorf("sign first-liquidity: %w", err)
	}

	body := firstLiquidityRequest{
		UserID:   c.botUserID,
		Quantity: quantity,
		Outcome:  outcome,
		Price:    price,
		Operator: &operatorAction{
			WalletAddress: c.wallet,
			RequestedAt:   now,
			Signature:     sig,
		},
	}

	resp, err := c.post(ctx, fmt.Sprintf("/v1/admin/markets/%d/first-liquidity", marketID), body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return nil, ErrAlreadySeeded
	}
	if resp.StatusCode != http.StatusAccepted {
		msg, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("first-liquidity: HTTP %d: %s", resp.StatusCode, string(msg))
	}
	var result firstLiquidityResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PlaceBootstrapOrder places an operator-authenticated SELL LIMIT GTC order.
func (c *OperatorAPIClient) PlaceBootstrapOrder(ctx context.Context, marketID int64, outcome string, price, quantity int64) (*createOrderResponse, error) {
	now := time.Now().UnixMilli()
	message := fmt.Sprintf(
		"FunnyOption Operator Authorization\n\naction: ISSUE_FIRST_LIQUIDITY\nwallet: %s\nmarket_id: %d\nuser_id: %d\nquantity: %d\noutcome: %s\nprice: %d\nrequested_at: %d\n",
		strings.ToLower(c.wallet), marketID, c.botUserID, quantity, outcome, price, now,
	)
	sig, err := c.signPersonalMessage(message)
	if err != nil {
		return nil, fmt.Errorf("sign bootstrap order: %w", err)
	}

	body := createOrderRequest{
		UserID:      c.botUserID,
		MarketID:    marketID,
		Outcome:     outcome,
		Side:        "SELL",
		Type:        "LIMIT",
		TimeInForce: "GTC",
		Price:       price,
		Quantity:    quantity,
		Operator: &operatorAction{
			WalletAddress: c.wallet,
			RequestedAt:   now,
			Signature:     sig,
		},
	}

	resp, err := c.post(ctx, "/v1/orders", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return nil, ErrOrderAlreadyExists
	}
	if resp.StatusCode != http.StatusAccepted {
		msg, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bootstrap order: HTTP %d: %s", resp.StatusCode, string(msg))
	}
	var result createOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *OperatorAPIClient) signPersonalMessage(message string) (string, error) {
	privKey, err := crypto.HexToECDSA(strings.TrimPrefix(c.privateKey, "0x"))
	if err != nil {
		return "", fmt.Errorf("parse private key: %w", err)
	}
	hash := accounts.TextHash([]byte(message))
	sig, err := crypto.Sign(hash, privKey)
	if err != nil {
		return "", fmt.Errorf("sign message: %w", err)
	}
	sig[crypto.RecoveryIDOffset] += 27
	return "0x" + hex.EncodeToString(sig), nil
}

func (c *OperatorAPIClient) get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.httpClient.Do(req)
}

func (c *OperatorAPIClient) post(ctx context.Context, path string, body any) (*http.Response, error) {
	if err := c.writePacer.Wait(ctx); err != nil {
		return nil, err
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		c.writePacer.PushBack(retryAfterDelay(resp))
	}
	return resp, nil
}

var (
	ErrAlreadySeeded      = fmt.Errorf("market already seeded")
	ErrOrderAlreadyExists = fmt.Errorf("bootstrap order already exists")
)
