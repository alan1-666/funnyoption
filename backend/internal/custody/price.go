package custody

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type PriceProvider struct {
	mu       sync.RWMutex
	cache    map[string]cachedPrice
	cacheTTL time.Duration
	http     *http.Client
}

type cachedPrice struct {
	price     float64
	fetchedAt time.Time
}

func NewPriceProvider(cacheTTL time.Duration) *PriceProvider {
	if cacheTTL <= 0 {
		cacheTTL = 15 * time.Second
	}
	return &PriceProvider{
		cache:    make(map[string]cachedPrice),
		cacheTTL: cacheTTL,
		http:     &http.Client{Timeout: 5 * time.Second},
	}
}

// GetUSDTPrice returns the price of the given symbol in USDT.
// For USDT itself, returns 1.0.
func (p *PriceProvider) GetUSDTPrice(ctx context.Context, symbol string) (float64, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "USDT" || symbol == "" {
		return 1.0, nil
	}

	p.mu.RLock()
	cached, ok := p.cache[symbol]
	p.mu.RUnlock()
	if ok && time.Since(cached.fetchedAt) < p.cacheTTL {
		return cached.price, nil
	}

	price, err := p.fetchBinancePrice(ctx, symbol+"USDT")
	if err != nil {
		if ok {
			return cached.price, nil
		}
		return 0, err
	}

	p.mu.Lock()
	p.cache[symbol] = cachedPrice{price: price, fetchedAt: time.Now()}
	p.mu.Unlock()
	return price, nil
}

func (p *PriceProvider) fetchBinancePrice(ctx context.Context, pair string) (float64, error) {
	url := "https://api.binance.com/api/v3/ticker/price?symbol=" + pair
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := p.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("binance price request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("binance price API %d: %s", resp.StatusCode, string(raw))
	}

	var result struct {
		Price string `json:"price"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return 0, fmt.Errorf("decode binance price: %w", err)
	}

	price, err := strconv.ParseFloat(result.Price, 64)
	if err != nil {
		return 0, fmt.Errorf("parse binance price %q: %w", result.Price, err)
	}
	if price <= 0 {
		return 0, fmt.Errorf("invalid binance price %f", price)
	}
	return price, nil
}
