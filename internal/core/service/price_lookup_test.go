package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"monity/internal/core/port"
	"monity/internal/models"
)

// slowPriceService answers each price after delay, failing for symbols in fail, and records the
// lookups and the most run at once.
type slowPriceService struct {
	port.PriceService
	delay time.Duration
	fail  map[string]bool

	mu        sync.Mutex
	calls     []string
	running   int
	maxAtOnce int
}

func (f *slowPriceService) lookup(symbol, currency string) (*port.PriceData, error) {
	f.mu.Lock()
	f.calls = append(f.calls, symbol)
	f.running++
	f.maxAtOnce = max(f.maxAtOnce, f.running)
	f.mu.Unlock()
	time.Sleep(f.delay)
	f.mu.Lock()
	f.running--
	f.mu.Unlock()
	if f.fail[symbol] {
		return nil, errors.New("upstream down")
	}
	return &port.PriceData{Symbol: symbol, Price: 10, Currency: currency}, nil
}

func (f *slowPriceService) GetCryptoPriceWithCurrency(_ context.Context, symbol, currency string) (*port.PriceData, error) {
	return f.lookup(symbol, currency)
}

func (f *slowPriceService) GetStockPriceWithCurrency(_ context.Context, symbol, currency string) (*port.PriceData, error) {
	return f.lookup(symbol, currency)
}

func assetOf(t models.AssetType, symbol string) models.Asset {
	return models.Asset{Type: t, Symbol: &symbol}
}

func TestFetchMarketPrices_ConcurrentAndDeduplicated(t *testing.T) {
	ps := &slowPriceService{delay: 50 * time.Millisecond, fail: map[string]bool{"BAD.JK": true}}
	assets := []models.Asset{
		assetOf(models.AssetTypeStock, "BBCA.JK"),
		assetOf(models.AssetTypeStock, "bbca.jk"), // same symbol, looked up once
		assetOf(models.AssetTypeStock, "BAD.JK"),
		assetOf(models.AssetTypeCrypto, "BTC"),
		assetOf(models.AssetTypeCrypto, "ETH"),
		{Type: models.AssetTypeCash}, // no market price
	}

	start := time.Now()
	prices := fetchMarketPrices(context.Background(), ps, assets, "IDR")
	elapsed := time.Since(start)

	if len(ps.calls) != 4 {
		t.Errorf("lookups = %v, want 4 (one per distinct symbol)", ps.calls)
	}
	if ps.maxAtOnce < 2 || elapsed > 150*time.Millisecond {
		t.Errorf("max at once = %d, took %v; want lookups in parallel", ps.maxAtOnce, elapsed)
	}
	if p, err := prices.get(&assets[1]); err != nil || p.Price != 10 {
		t.Errorf("bbca.jk = %+v, %v; want the BBCA.JK price", p, err)
	}
	if _, err := prices.get(&assets[2]); err == nil {
		t.Error("BAD.JK: want its lookup error")
	}
	if _, err := prices.get(&assets[5]); !errors.Is(err, errPriceNotFetched) {
		t.Errorf("cash: err = %v, want errPriceNotFetched", err)
	}
}

func TestFetchMarketPrices_BoundedConcurrency(t *testing.T) {
	ps := &slowPriceService{delay: 10 * time.Millisecond}
	var assets []models.Asset
	for _, s := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"} {
		assets = append(assets, assetOf(models.AssetTypeCrypto, s))
	}
	fetchMarketPrices(context.Background(), ps, assets, "IDR")
	if ps.maxAtOnce > maxPriceFetches {
		t.Errorf("max at once = %d, want at most %d", ps.maxAtOnce, maxPriceFetches)
	}
}
