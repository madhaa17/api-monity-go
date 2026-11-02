package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"monity/internal/core/port"
	"monity/internal/models"
)

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func closeDates(closes []models.MarketDailyClose) []string {
	out := make([]string, len(closes))
	for i, c := range closes {
		out[i] = c.Date.Format("2006-01-02") + "=" + c.Close.String()
	}
	return out
}

func TestStockDailyCloses_SkipsToday(t *testing.T) {
	now := time.Date(2026, 10, 30, 10, 0, 0, 0, time.UTC) // IDX closed for the day, bar not yet stored
	chart := &port.ChartResponse{Currency: "IDR", Data: []port.ChartDataPoint{
		{T: time.Date(2026, 10, 28, 2, 0, 0, 0, time.UTC).Unix(), P: 4900}, // 09:00 WIB session opens
		{T: time.Date(2026, 10, 29, 2, 0, 0, 0, time.UTC).Unix(), P: 4940},
		{T: time.Date(2026, 10, 30, 2, 0, 0, 0, time.UTC).Unix(), P: 4980},
	}}
	got := closeDates(stockDailyCloses("BBCA.JK", chart, now))
	want := []string{"2026-10-28=4900", "2026-10-29=4940"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("closes = %v, want %v", got, want)
	}
}

func TestCryptoDailyCloses_MidnightIsPreviousDayAndLivePointSkipped(t *testing.T) {
	now := time.Date(2026, 10, 31, 0, 21, 10, 0, time.UTC)
	chart := &port.ChartResponse{Currency: "IDR", Data: []port.ChartDataPoint{
		{T: day("2026-10-30").Unix(), P: 1410},
		{T: day("2026-10-31").Unix(), P: 1400},
		{T: now.Unix(), P: 1402}, // live price
	}}
	closes := cryptoDailyCloses("BTC", chart, now)
	got := closeDates(closes)
	want := []string{"2026-10-29=1410", "2026-10-30=1400"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("closes = %v, want %v", got, want)
	}
	if closes[0].Currency != "IDR" || closes[0].Source != "coingecko" || closes[0].AssetType != models.AssetTypeCrypto {
		t.Errorf("close = %+v, want an IDR coingecko crypto close", closes[0])
	}
}

type fakeCloseRepo struct {
	symbols []port.PricedSymbol
	stored  []models.MarketDailyClose
}

func (f *fakeCloseRepo) ListPricedSymbols(context.Context) ([]port.PricedSymbol, error) {
	return f.symbols, nil
}

func (f *fakeCloseRepo) Upsert(_ context.Context, closes []models.MarketDailyClose) error {
	f.stored = append(f.stored, closes...)
	return nil
}

// chartPriceService serves a one-day-old daily chart for every symbol except DOWN.JK.
type chartPriceService struct{ port.PriceService }

func (chartPriceService) GetStockChart(_ context.Context, symbol, rangeParam, interval string) (*port.ChartResponse, error) {
	if symbol == "DOWN.JK" {
		return nil, errors.New("yahoo down")
	}
	if rangeParam != "1y" || interval != "1d" {
		return nil, errors.New("want the 1y daily chart")
	}
	yesterday := utcDate(time.Now()).AddDate(0, 0, -1).Add(2 * time.Hour)
	return &port.ChartResponse{Symbol: symbol, Currency: "IDR", Data: []port.ChartDataPoint{{T: yesterday.Unix(), P: 100}}}, nil
}

func (chartPriceService) GetCryptoChart(_ context.Context, symbol, currency string, days int) (*port.ChartResponse, error) {
	return &port.ChartResponse{Symbol: symbol, Currency: currency, Data: []port.ChartDataPoint{{T: utcDate(time.Now()).Unix(), P: 200}}}, nil
}

func TestRecordDailyCloses_OneSymbolFailingDoesNotStopOthers(t *testing.T) {
	repo := &fakeCloseRepo{symbols: []port.PricedSymbol{
		{AssetType: models.AssetTypeCrypto, Symbol: "BTC"},
		{AssetType: models.AssetTypeStock, Symbol: "DOWN.JK"},
		{AssetType: models.AssetTypeStock, Symbol: "BBCA.JK"},
	}}
	svc := NewDailyCloseService(repo, chartPriceService{})

	n, err := svc.RecordDailyCloses(context.Background())
	if err != nil {
		t.Fatalf("RecordDailyCloses: %v", err)
	}
	if n != 2 || len(repo.stored) != 2 {
		t.Fatalf("stored %d (%+v), want BTC and BBCA.JK", n, repo.stored)
	}
	if repo.stored[0].Symbol != "BTC" || repo.stored[1].Symbol != "BBCA.JK" {
		t.Errorf("stored %+v, want BTC then BBCA.JK", repo.stored)
	}
}
