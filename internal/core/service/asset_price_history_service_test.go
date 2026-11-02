package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"monity/internal/core/port"
	"monity/internal/models"
)

type oneAssetRepo struct {
	port.AssetRepository
	asset *models.Asset
}

func (r oneAssetRepo) GetByUUID(_ context.Context, uuid string, _ int64) (*models.Asset, error) {
	if r.asset == nil || r.asset.UUID != uuid {
		return nil, nil
	}
	return r.asset, nil
}

type recordingHistoryRepo struct {
	port.AssetPriceHistoryRepository
	created []models.AssetPriceHistory
}

func (r *recordingHistoryRepo) Create(_ context.Context, h *models.AssetPriceHistory) error {
	r.created = append(r.created, *h)
	return nil
}

// fixedPriceService returns price for every lookup and records the currency asked for.
type fixedPriceService struct {
	port.PriceService
	price     port.PriceData
	askedCurr string
}

func (f *fixedPriceService) GetPriceWithCurrency(_ context.Context, _, symbol, currency string) (*port.PriceData, error) {
	f.askedCurr = currency
	p := f.price
	p.Symbol = symbol
	return &p, nil
}

func stockAsset(currency string) *models.Asset {
	symbol := "AAPL"
	return &models.Asset{ID: 7, UUID: "a-1", Type: models.AssetTypeStock, Symbol: &symbol, PurchaseCurrency: currency}
}

func TestFetchAndRecordPrice_RecordsInPurchaseCurrency(t *testing.T) {
	history := &recordingHistoryRepo{}
	ps := &fixedPriceService{price: port.PriceData{Price: 180, Currency: "USD", Source: "yahoo", FetchedAt: time.Now()}}
	svc := NewAssetPriceHistoryService(history, oneAssetRepo{asset: stockAsset("USD")}, ps)

	h, err := svc.FetchAndRecordPrice(context.Background(), 1, "a-1")
	if err != nil {
		t.Fatalf("FetchAndRecordPrice: %v", err)
	}
	if ps.askedCurr != "USD" || h.Price.String() != "180" || len(history.created) != 1 {
		t.Errorf("asked %s, recorded %+v (%d rows); want 180 fetched in USD", ps.askedCurr, h, len(history.created))
	}
}

func TestFetchAndRecordPrice_RefusesUnusablePrices(t *testing.T) {
	tests := map[string]port.PriceData{
		"stale":       {Price: 2_900_000, Currency: "IDR", Stale: true, FetchedAt: time.Now().Add(-48 * time.Hour)},
		"unconverted": {Price: 180, Currency: "USD", FetchedAt: time.Now()},
	}
	for name, price := range tests {
		t.Run(name, func(t *testing.T) {
			history := &recordingHistoryRepo{}
			svc := NewAssetPriceHistoryService(history, oneAssetRepo{asset: stockAsset("IDR")}, &fixedPriceService{price: price})

			_, err := svc.FetchAndRecordPrice(context.Background(), 1, "a-1")
			if !errors.Is(err, port.ErrPriceUnavailable) {
				t.Errorf("err = %v, want ErrPriceUnavailable", err)
			}
			if len(history.created) != 0 {
				t.Errorf("recorded %+v, want nothing", history.created)
			}
		})
	}
}

func TestFetchAndRecordPrice_TypedErrors(t *testing.T) {
	cash := &models.Asset{ID: 8, UUID: "c-1", Type: models.AssetTypeCash}
	svc := NewAssetPriceHistoryService(&recordingHistoryRepo{}, oneAssetRepo{asset: cash}, &fixedPriceService{})

	var validationErr *port.ValidationError
	if _, err := svc.FetchAndRecordPrice(context.Background(), 1, "c-1"); !errors.As(err, &validationErr) {
		t.Errorf("cash asset: err = %v, want a validation error", err)
	}
	if _, err := svc.FetchAndRecordPrice(context.Background(), 1, "missing"); !errors.Is(err, port.ErrNotFound) {
		t.Errorf("missing asset: err = %v, want ErrNotFound", err)
	}
}

func TestRecordPrice_RefusesFutureTime(t *testing.T) {
	history := &recordingHistoryRepo{}
	svc := NewAssetPriceHistoryService(history, oneAssetRepo{asset: stockAsset("IDR")}, &fixedPriceService{})

	req := port.RecordPriceRequest{Price: 100, Source: "manual", RecordedAt: time.Now().Add(24 * time.Hour)}
	var validationErr *port.ValidationError
	if _, err := svc.RecordPrice(context.Background(), 1, "a-1", req); !errors.As(err, &validationErr) || len(history.created) != 0 {
		t.Errorf("err = %v, recorded %d; want a validation error and nothing recorded", err, len(history.created))
	}

	req.RecordedAt = time.Now().Add(time.Minute) // within clock skew
	if _, err := svc.RecordPrice(context.Background(), 1, "a-1", req); err != nil {
		t.Errorf("a minute ahead: err = %v, want it recorded", err)
	}
}
