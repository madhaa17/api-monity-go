package service

import (
	"context"
	"errors"
	"testing"

	"monity/internal/core/port"
	"monity/internal/models"
)

func TestCreateAssetRejectsBadInputAsValidation(t *testing.T) {
	f := newFakeLedger()
	svc := NewAssetService(f)
	base := func() port.CreateAssetRequest {
		return port.CreateAssetRequest{Name: "Wallet", Type: models.AssetTypeCash, Quantity: 10, PurchaseDate: "2026-01-01T00:00:00Z"}
	}
	strp := func(s string) *string { return &s }
	cases := map[string]func(*port.CreateAssetRequest){
		"blank name":       func(r *port.CreateAssetRequest) { r.Name = "  " },
		"bad type":         func(r *port.CreateAssetRequest) { r.Type = "GOLD" },
		"bad status":       func(r *port.CreateAssetRequest) { s := models.AssetStatus("LOST"); r.Status = &s },
		"bad date":         func(r *port.CreateAssetRequest) { r.PurchaseDate = "31/01/2026" },
		"bad target date":  func(r *port.CreateAssetRequest) { r.TargetDate = strp("soon") },
		"negative qty":     func(r *port.CreateAssetRequest) { r.Quantity = -1 },
		"too large amount": func(r *port.CreateAssetRequest) { r.TotalCost = 1e15 },
	}
	for name, mutate := range cases {
		req := base()
		mutate(&req)
		_, err := svc.CreateAsset(context.Background(), 1, req)
		var v *port.ValidationError
		if !errors.As(err, &v) {
			t.Errorf("%s: want ValidationError, got %v", name, err)
		}
	}

	// A plain date is accepted.
	req := base()
	req.PurchaseDate = "2026-01-31"
	asset, err := svc.CreateAsset(context.Background(), 1, req)
	if err != nil {
		t.Fatalf("plain date: %v", err)
	}
	if got := asset.PurchaseDate.Format("2006-01-02"); got != "2026-01-31" {
		t.Fatalf("purchase date = %s", got)
	}
}

func TestUpdateAssetValidatesBeforeApplying(t *testing.T) {
	f := newFakeLedger()
	svc := NewAssetService(f)
	a := f.addAsset(1, models.AssetTypeCash, "5")
	bad := "not-a-date"
	_, err := svc.UpdateAsset(context.Background(), 1, a.UUID, port.UpdateAssetRequest{PurchaseDate: &bad})
	var v *port.ValidationError
	if !errors.As(err, &v) {
		t.Fatalf("want ValidationError, got %v", err)
	}
	_, err = svc.GetAsset(context.Background(), 1, "missing")
	if !errors.Is(err, port.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}
