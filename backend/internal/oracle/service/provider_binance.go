package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type priceProvider interface {
	Observe(ctx context.Context, contract *Contract) (*Observation, *providerError)
}

type providerError struct {
	Code      string
	Retryable bool
	Message   string
}

func (e *providerError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

type BinanceProvider struct {
	baseURL    string
	httpClient *http.Client
}

type binanceTrade struct {
	ID           int64  `json:"id"`
	Price        string `json:"price"`
	Quantity     string `json:"qty"`
	QuoteQty     string `json:"quoteQty"`
	Time         int64  `json:"time"`
	IsBuyerMaker bool   `json:"isBuyerMaker"`
	IsBestMatch  bool   `json:"isBestMatch"`
}

func NewBinanceProvider(baseURL string, httpClient *http.Client) *BinanceProvider {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.binance.com"
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	return &BinanceProvider{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

func (p *BinanceProvider) Observe(ctx context.Context, contract *Contract) (*Observation, *providerError) {
	if contract == nil {
		return nil, &providerError{Code: ErrorCodeInvalidMetadata, Retryable: false, Message: "resolution contract is required"}
	}

	endpoint := p.baseURL + "/api/v3/trades"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, &providerError{Code: ErrorCodeSourceUnavailable, Retryable: true, Message: err.Error()}
	}
	query := url.Values{}
	query.Set("symbol", contract.Metadata.Oracle.Instrument.Symbol)
	query.Set("limit", "1")
	req.URL.RawQuery = query.Encode()
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		if isTimeoutError(err) {
			return nil, &providerError{Code: ErrorCodeSourceTimeout, Retryable: true, Message: err.Error()}
		}
		return nil, &providerError{Code: ErrorCodeSourceUnavailable, Retryable: true, Message: err.Error()}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, &providerError{Code: ErrorCodeSourceUnavailable, Retryable: true, Message: err.Error()}
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
		return nil, &providerError{
			Code:      ErrorCodeSourceUnavailable,
			Retryable: true,
			Message:   fmt.Sprintf("binance returned %d", resp.StatusCode),
		}
	}
	if resp.StatusCode >= http.StatusBadRequest {
		message := strings.ToLower(string(body))
		code := ErrorCodeInvalidMetadata
		if strings.Contains(message, "invalid symbol") {
			code = ErrorCodeUnsupportedSymbol
		}
		return nil, &providerError{
			Code:      code,
			Retryable: false,
			Message:   fmt.Sprintf("binance returned %d", resp.StatusCode),
		}
	}

	var trades []binanceTrade
	if err := json.Unmarshal(body, &trades); err != nil {
		return nil, &providerError{Code: ErrorCodeSourceUnavailable, Retryable: true, Message: err.Error()}
	}
	if len(trades) == 0 {
		return nil, &providerError{Code: ErrorCodeSourceUnavailable, Retryable: true, Message: "binance returned no trades"}
	}

	observedScaled, observedPrice, err := contract.NormalizeObservedPrice(trades[0].Price)
	if err != nil {
		return nil, &providerError{Code: ErrorCodeInvalidMetadata, Retryable: false, Message: err.Error()}
	}

	var rawPayload any
	if err := json.Unmarshal(body, &rawPayload); err != nil {
		rawPayload = trades[0]
	} else if array, ok := rawPayload.([]any); ok && len(array) > 0 {
		rawPayload = array[0]
	}

	return &Observation{
		FetchedAt:      time.Now().Unix(),
		EffectiveAt:    trades[0].Time / 1000,
		ObservedPrice:  observedPrice,
		ObservedScaled: observedScaled,
		RawPayload:     rawPayload,
		RawPayloadHash: hashRawPayload(body),
	}, nil
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "timeout")
}

func hashRawPayload(body []byte) string {
	sum := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(sum[:])
}
