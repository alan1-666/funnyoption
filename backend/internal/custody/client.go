package custody

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type SaaSClient struct {
	baseURL  string
	token    string
	tenantID string
	http     *http.Client
}

func NewSaaSClient(baseURL, token, tenantID string) *SaaSClient {
	return &SaaSClient{
		baseURL:  strings.TrimRight(baseURL, "/"),
		token:    token,
		tenantID: tenantID,
		http:     &http.Client{Timeout: 10 * time.Second},
	}
}

type UpsertAccountRequest struct {
	TenantID  string `json:"tenant_id"`
	AccountID string `json:"account_id"`
	Status    string `json:"status"`
}

type CreateAddressRequest struct {
	TenantID  string `json:"tenant_id"`
	AccountID string `json:"account_id"`
	Chain     string `json:"chain"`
	Coin      string `json:"coin"`
	Network   string `json:"network"`
	SignType  string `json:"sign_type"`
	Model     string `json:"model"`
}

type CreateAddressResponse struct {
	TenantID  string `json:"tenant_id"`
	AccountID string `json:"account_id"`
	Chain     string `json:"chain"`
	Coin      string `json:"coin"`
	Network   string `json:"network"`
	Address   string `json:"address"`
	KeyID     string `json:"key_id"`
}

type CreateWithdrawRequest struct {
	TenantID  string `json:"tenant_id"`
	AccountID string `json:"account_id"`
	OrderID   string `json:"order_id"`
	KeyID     string `json:"key_id"`
	Chain     string `json:"chain"`
	Network   string `json:"network"`
	Coin      string `json:"coin"`
	To        string `json:"to"`
	Amount    string `json:"amount"`
}

type CreateWithdrawResponse struct {
	TxHash string `json:"tx_hash"`
	Status string `json:"status"`
}

func (c *SaaSClient) UpsertAccount(ctx context.Context, accountID string) error {
	body := UpsertAccountRequest{
		TenantID:  c.tenantID,
		AccountID: accountID,
		Status:    "ACTIVE",
	}
	_, err := c.post(ctx, "/v1/account/upsert", body)
	return err
}

func (c *SaaSClient) CreateAddress(ctx context.Context, accountID, chain, coin, network string) (*CreateAddressResponse, error) {
	body := CreateAddressRequest{
		TenantID:  c.tenantID,
		AccountID: accountID,
		Chain:     chain,
		Coin:      coin,
		Network:   network,
		SignType:  "ecdsa",
		Model:     "account",
	}
	raw, err := c.post(ctx, "/v1/address/create", body)
	if err != nil {
		return nil, err
	}
	var resp CreateAddressResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("decode address response: %w", err)
	}
	return &resp, nil
}

func (c *SaaSClient) SubmitWithdraw(ctx context.Context, req CreateWithdrawRequest) (*CreateWithdrawResponse, error) {
	req.TenantID = c.tenantID
	raw, err := c.post(ctx, "/v1/withdraw", req)
	if err != nil {
		return nil, err
	}
	var resp CreateWithdrawResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("decode withdraw response: %w", err)
	}
	return &resp, nil
}

func (c *SaaSClient) post(ctx context.Context, path string, body any) ([]byte, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("saas request %s: %w", path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("saas %s returned %d: %s", path, resp.StatusCode, string(raw))
	}
	return raw, nil
}
