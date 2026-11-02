package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"monity/internal/core/port"
	"monity/internal/models"

	"github.com/shopspring/decimal"
)

// dailyCloseInterval is how often the daily closes are recorded. Each run stores the past year
// again, so a missed run or a close Yahoo corrects later is picked up by the next one.
const dailyCloseInterval = 6 * time.Hour

type DailyCloseService struct {
	closeRepo    port.MarketDailyCloseRepository
	priceService port.PriceService
}

func NewDailyCloseService(closeRepo port.MarketDailyCloseRepository, priceService port.PriceService) port.DailyCloseService {
	return &DailyCloseService{closeRepo: closeRepo, priceService: priceService}
}

func (s *DailyCloseService) Run(ctx context.Context) {
	ticker := time.NewTicker(dailyCloseInterval)
	defer ticker.Stop()
	for {
		if n, err := s.RecordDailyCloses(ctx); err != nil {
			slog.Warn("daily_closes_failed", "error", err)
		} else {
			slog.Info("daily_closes_recorded", "count", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *DailyCloseService) RecordDailyCloses(ctx context.Context) (int, error) {
	symbols, err := s.closeRepo.ListPricedSymbols(ctx)
	if err != nil {
		return 0, err
	}
	stored := 0
	for _, sym := range symbols {
		if ctx.Err() != nil {
			return stored, ctx.Err()
		}
		closes, err := s.symbolCloses(ctx, sym, time.Now())
		if err != nil {
			// One symbol's outage must not stop the others.
			slog.Warn("daily_closes_symbol_failed", "type", sym.AssetType, "symbol", sym.Symbol, "error", err)
			continue
		}
		if err := s.closeRepo.Upsert(ctx, closes); err != nil {
			return stored, err
		}
		stored += len(closes)
	}
	return stored, nil
}

// symbolCloses returns the closes of the past year of one symbol, from the same cached daily
// charts the asset page shows.
func (s *DailyCloseService) symbolCloses(ctx context.Context, sym port.PricedSymbol, now time.Time) ([]models.MarketDailyClose, error) {
	switch sym.AssetType {
	case models.AssetTypeStock:
		chart, err := s.priceService.GetStockChart(ctx, sym.Symbol, "1y", "1d")
		if err != nil {
			return nil, err
		}
		return stockDailyCloses(sym.Symbol, chart, now), nil
	case models.AssetTypeCrypto:
		chart, err := s.priceService.GetCryptoChart(ctx, sym.Symbol, port.DefaultCurrency, 365)
		if err != nil {
			return nil, err
		}
		return cryptoDailyCloses(sym.Symbol, chart, now), nil
	default:
		return nil, fmt.Errorf("no daily closes for asset type %s", sym.AssetType)
	}
}

// stockDailyCloses turns a daily Yahoo chart into closes. A bar's timestamp is its session's
// open, which falls on the session's UTC date for IDX and US listings. Today's bar is skipped:
// it may still be trading, and is stored final by a run on a later day.
func stockDailyCloses(symbol string, chart *port.ChartResponse, now time.Time) []models.MarketDailyClose {
	today := utcDate(now)
	out := make([]models.MarketDailyClose, 0, len(chart.Data))
	for _, p := range chart.Data {
		date := utcDate(time.Unix(p.T, 0))
		if !date.Before(today) {
			continue
		}
		out = append(out, dailyClose(models.AssetTypeStock, symbol, chart.Currency, date, p.P, "yahoo", now))
	}
	return out
}

// cryptoDailyCloses turns a daily CoinGecko chart into closes. Its daily points sit at 00:00 UTC
// and are the close of the day before; the last point is the live price and is skipped, as is
// any other point off midnight.
func cryptoDailyCloses(symbol string, chart *port.ChartResponse, now time.Time) []models.MarketDailyClose {
	out := make([]models.MarketDailyClose, 0, len(chart.Data))
	for _, p := range chart.Data {
		t := time.Unix(p.T, 0).UTC()
		if !t.Equal(utcDate(t)) {
			continue
		}
		out = append(out, dailyClose(models.AssetTypeCrypto, symbol, chart.Currency, t.AddDate(0, 0, -1), p.P, "coingecko", now))
	}
	return out
}

func dailyClose(assetType models.AssetType, symbol, currency string, date time.Time, price float64, source string, now time.Time) models.MarketDailyClose {
	return models.MarketDailyClose{
		AssetType:  assetType,
		Symbol:     symbol,
		Currency:   currency,
		Date:       date,
		Close:      decimal.NewFromFloat(price),
		Source:     source,
		RecordedAt: now,
	}
}

// utcDate returns the start of t's UTC day.
func utcDate(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
