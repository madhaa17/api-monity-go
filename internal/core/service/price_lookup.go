package service

import (
	"context"
	"errors"
	"strings"
	"sync"

	"monity/internal/core/port"
	"monity/internal/models"

	"golang.org/x/sync/errgroup"
)

// maxPriceFetches bounds how many market prices are looked up at once, to stay friendly with the
// free Yahoo and CoinGecko APIs.
const maxPriceFetches = 8

// priceKey identifies one market price lookup.
type priceKey struct {
	assetType models.AssetType
	symbol    string
}

type priceResult struct {
	data *port.PriceData
	err  error
}

var errPriceNotFetched = errors.New("price not fetched")

// priceKeyOf returns the lookup key of a crypto or stock asset with a symbol, and false for any
// other asset.
func priceKeyOf(asset *models.Asset) (priceKey, bool) {
	if asset.Symbol == nil || *asset.Symbol == "" {
		return priceKey{}, false
	}
	if asset.Type != models.AssetTypeCrypto && asset.Type != models.AssetTypeStock {
		return priceKey{}, false
	}
	return priceKey{asset.Type, strings.ToUpper(*asset.Symbol)}, true
}

// marketPrices holds the market prices of a set of assets, looked up together.
type marketPrices map[priceKey]priceResult

// get returns the price looked up for asset.
func (m marketPrices) get(asset *models.Asset) (*port.PriceData, error) {
	key, ok := priceKeyOf(asset)
	if !ok {
		return nil, errPriceNotFetched
	}
	r, ok := m[key]
	if !ok {
		return nil, errPriceNotFetched
	}
	return r.data, r.err
}

// fetchMarketPrices looks up the current prices of the crypto and stock assets in currency
// concurrently, each distinct symbol once, so a portfolio's prices take about as long as its
// slowest one instead of the sum of all.
func fetchMarketPrices(ctx context.Context, ps port.PriceService, assets []models.Asset, currency string) marketPrices {
	out := make(marketPrices)
	var mu sync.Mutex
	var g errgroup.Group
	g.SetLimit(maxPriceFetches)
	seen := make(map[priceKey]bool)
	for i := range assets {
		key, ok := priceKeyOf(&assets[i])
		if !ok || seen[key] {
			continue
		}
		seen[key] = true
		g.Go(func() error {
			var r priceResult
			if key.assetType == models.AssetTypeCrypto {
				r.data, r.err = ps.GetCryptoPriceWithCurrency(ctx, key.symbol, currency)
			} else {
				r.data, r.err = ps.GetStockPriceWithCurrency(ctx, key.symbol, currency)
			}
			mu.Lock()
			out[key] = r
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait()
	return out
}
