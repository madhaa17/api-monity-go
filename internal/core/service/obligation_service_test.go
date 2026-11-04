package service

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"monity/internal/core/port"
	"monity/internal/models"

	"github.com/shopspring/decimal"
)

// fakeObligations is an in-memory Debt/Receivable (+payment) repository. Reads return copies, so a
// service mutating a loaded row inside a failed transaction leaves the stored row untouched.
type fakeObligations struct {
	ledger             *fakeLedger
	debts              map[string]models.Debt
	debtPayments       map[int64]models.DebtPayment
	receivables        map[string]models.Receivable
	receivablePayments map[int64]models.ReceivablePayment
	nextID             int64
}

func newFakeObligations() *fakeObligations {
	return &fakeObligations{
		ledger:             newFakeLedger(),
		debts:              map[string]models.Debt{},
		debtPayments:       map[int64]models.DebtPayment{},
		receivables:        map[string]models.Receivable{},
		receivablePayments: map[int64]models.ReceivablePayment{},
	}
}

func (f *fakeObligations) id() (int64, string) {
	f.nextID++
	return f.nextID, "00000000-0000-0000-0000-" + leftPad(f.nextID)
}

func leftPad(n int64) string {
	s := strconv.FormatInt(n, 10)
	for len(s) < 12 {
		s = "0" + s
	}
	return s
}

type fakeDebtRepo struct{ *fakeObligations }

func (r fakeDebtRepo) Create(_ context.Context, d *models.Debt) error {
	d.ID, d.UUID = r.id()
	r.debts[d.UUID] = *d
	return nil
}
func (r fakeDebtRepo) GetByUUID(_ context.Context, uuid string, userID int64) (*models.Debt, error) {
	d, ok := r.debts[uuid]
	if !ok || d.UserID != userID {
		return nil, nil
	}
	return &d, nil
}
func (r fakeDebtRepo) GetByUUIDForUpdate(ctx context.Context, uuid string, userID int64) (*models.Debt, error) {
	return r.GetByUUID(ctx, uuid, userID)
}
func (r fakeDebtRepo) ListByUserID(context.Context, int64, *string, *time.Time, *time.Time, int, int) ([]models.Debt, int64, error) {
	return nil, 0, nil
}
func (r fakeDebtRepo) Update(_ context.Context, d *models.Debt) error {
	r.debts[d.UUID] = *d
	return nil
}
func (r fakeDebtRepo) Delete(_ context.Context, uuid string, _ int64) error {
	id := r.debts[uuid].ID
	delete(r.debts, uuid)
	for pid, p := range r.debtPayments {
		if p.DebtID == id {
			delete(r.debtPayments, pid)
		}
	}
	return nil
}

type fakeDebtPaymentRepo struct{ *fakeObligations }

func (r fakeDebtPaymentRepo) Create(_ context.Context, p *models.DebtPayment) error {
	p.ID, p.UUID = r.id()
	r.debtPayments[p.ID] = *p
	return nil
}
func (r fakeDebtPaymentRepo) GetByUUID(_ context.Context, uuid string, debtID int64) (*models.DebtPayment, error) {
	for _, p := range r.debtPayments {
		if p.UUID == uuid && p.DebtID == debtID {
			return &p, nil
		}
	}
	return nil, nil
}
func (r fakeDebtPaymentRepo) ListByDebtID(_ context.Context, debtID int64) ([]models.DebtPayment, error) {
	var out []models.DebtPayment
	for _, p := range r.debtPayments {
		if p.DebtID == debtID {
			out = append(out, p)
		}
	}
	return out, nil
}
func (r fakeDebtPaymentRepo) Delete(_ context.Context, id int64) error {
	delete(r.debtPayments, id)
	return nil
}

type fakeReceivableRepo struct{ *fakeObligations }

func (r fakeReceivableRepo) Create(_ context.Context, rec *models.Receivable) error {
	rec.ID, rec.UUID = r.id()
	r.receivables[rec.UUID] = *rec
	return nil
}
func (r fakeReceivableRepo) GetByUUID(_ context.Context, uuid string, userID int64) (*models.Receivable, error) {
	rec, ok := r.receivables[uuid]
	if !ok || rec.UserID != userID {
		return nil, nil
	}
	return &rec, nil
}
func (r fakeReceivableRepo) GetByUUIDForUpdate(ctx context.Context, uuid string, userID int64) (*models.Receivable, error) {
	return r.GetByUUID(ctx, uuid, userID)
}
func (r fakeReceivableRepo) ListByUserID(context.Context, int64, *string, *time.Time, *time.Time, int, int) ([]models.Receivable, int64, error) {
	return nil, 0, nil
}
func (r fakeReceivableRepo) Update(_ context.Context, rec *models.Receivable) error {
	r.receivables[rec.UUID] = *rec
	return nil
}
func (r fakeReceivableRepo) Delete(_ context.Context, uuid string, _ int64) error {
	id := r.receivables[uuid].ID
	delete(r.receivables, uuid)
	for pid, p := range r.receivablePayments {
		if p.ReceivableID == id {
			delete(r.receivablePayments, pid)
		}
	}
	return nil
}

type fakeReceivablePaymentRepo struct{ *fakeObligations }

func (r fakeReceivablePaymentRepo) Create(_ context.Context, p *models.ReceivablePayment) error {
	p.ID, p.UUID = r.id()
	r.receivablePayments[p.ID] = *p
	return nil
}
func (r fakeReceivablePaymentRepo) GetByUUID(_ context.Context, uuid string, receivableID int64) (*models.ReceivablePayment, error) {
	for _, p := range r.receivablePayments {
		if p.UUID == uuid && p.ReceivableID == receivableID {
			return &p, nil
		}
	}
	return nil, nil
}
func (r fakeReceivablePaymentRepo) ListByReceivableID(_ context.Context, receivableID int64) ([]models.ReceivablePayment, error) {
	var out []models.ReceivablePayment
	for _, p := range r.receivablePayments {
		if p.ReceivableID == receivableID {
			out = append(out, p)
		}
	}
	return out, nil
}
func (r fakeReceivablePaymentRepo) Delete(_ context.Context, id int64) error {
	delete(r.receivablePayments, id)
	return nil
}

const obligationUser int64 = 1

func newObligationServices() (*fakeObligations, port.DebtService, port.ReceivableService) {
	f := newFakeObligations()
	debts := NewDebtService(fakeDebtRepo{f}, fakeDebtPaymentRepo{f}, f.ledger, f.ledger)
	receivables := NewReceivableService(fakeReceivableRepo{f}, fakeReceivablePaymentRepo{f}, f.ledger, f.ledger)
	return f, debts, receivables
}

func strPtr(s string) *string { return &s }

func TestDebtLifecycleMovesCash(t *testing.T) {
	ctx := context.Background()
	f, svc, _ := newObligationServices()
	cash := f.ledger.addAsset(obligationUser, models.AssetTypeCash, "100")

	debt, err := svc.CreateDebt(ctx, obligationUser, port.CreateDebtRequest{
		PartyName: " Budi ", Amount: decimal.RequireFromString("500"), AssetUUID: &cash.UUID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if debt.PartyName != "Budi" || f.ledger.balance(cash.ID) != "600.00" {
		t.Fatalf("borrowing should add to cash: party=%q balance=%s", debt.PartyName, f.ledger.balance(cash.ID))
	}

	pay := func(amount string) (*models.DebtPayment, error) {
		return svc.RecordDebtPayment(ctx, obligationUser, debt.UUID, port.CreateDebtPaymentRequest{
			Amount: decimal.RequireFromString(amount), Date: time.Now(), AssetUUID: &cash.UUID,
		})
	}
	p1, err := pay("200")
	if err != nil {
		t.Fatal(err)
	}
	if got := f.debts[debt.UUID]; got.Status != models.ObligationStatusPartial || !got.PaidAmount.Equal(decimal.NewFromInt(200)) {
		t.Fatalf("after partial payment: status=%s paid=%s", got.Status, got.PaidAmount)
	}
	if f.ledger.balance(cash.ID) != "400.00" {
		t.Fatalf("repayment should take cash out, balance=%s", f.ledger.balance(cash.ID))
	}

	// Over-paying is rejected before anything is written.
	_, err = pay("300.01")
	wantValidation(t, err, "payment exceeds the remaining 300.00")
	if len(f.debtPayments) != 1 || f.ledger.balance(cash.ID) != "400.00" {
		t.Fatalf("rejected over-payment must not persist: payments=%d balance=%s", len(f.debtPayments), f.ledger.balance(cash.ID))
	}

	if _, err := pay("300"); err != nil {
		t.Fatal(err)
	}
	if f.debts[debt.UUID].Status != models.ObligationStatusPaid {
		t.Fatalf("want PAID, got %s", f.debts[debt.UUID].Status)
	}

	// Removing the first payment refunds it and reopens the debt.
	if err := svc.DeleteDebtPayment(ctx, obligationUser, debt.UUID, p1.UUID); err != nil {
		t.Fatal(err)
	}
	if got := f.debts[debt.UUID]; got.Status != models.ObligationStatusPartial || f.ledger.balance(cash.ID) != "300.00" {
		t.Fatalf("after deleting payment: status=%s balance=%s", got.Status, f.ledger.balance(cash.ID))
	}

	// Amount cannot drop below what was paid.
	_, err = svc.UpdateDebt(ctx, obligationUser, debt.UUID, port.UpdateDebtRequest{Amount: decPtr("299")})
	wantValidation(t, err, "amount cannot be less than the 300.00 already paid")

	// Deleting the debt undoes the borrowing (-500) and the remaining payment (+300).
	if err := svc.DeleteDebt(ctx, obligationUser, debt.UUID); err != nil {
		t.Fatal(err)
	}
	if f.ledger.balance(cash.ID) != "100.00" || len(f.debts) != 0 || len(f.debtPayments) != 0 {
		t.Fatalf("delete should restore cash: balance=%s debts=%d payments=%d", f.ledger.balance(cash.ID), len(f.debts), len(f.debtPayments))
	}
}

func TestDebtUpdateMovesBorrowedAmountBetweenAssets(t *testing.T) {
	ctx := context.Background()
	f, svc, _ := newObligationServices()
	wallet := f.ledger.addAsset(obligationUser, models.AssetTypeCash, "0")
	bank := f.ledger.addAsset(obligationUser, models.AssetTypeCash, "0")

	debt, err := svc.CreateDebt(ctx, obligationUser, port.CreateDebtRequest{PartyName: "Ani", Amount: decimal.NewFromInt(100), AssetUUID: &wallet.UUID})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.UpdateDebt(ctx, obligationUser, debt.UUID, port.UpdateDebtRequest{
		Amount:    decPtr("150"),
		AssetUUID: port.Optional[string]{Set: true, Value: &bank.UUID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.ledger.balance(wallet.ID) != "0.00" || f.ledger.balance(bank.ID) != "150.00" {
		t.Fatalf("wallet=%s bank=%s", f.ledger.balance(wallet.ID), f.ledger.balance(bank.ID))
	}

	// Unlinking the asset takes the borrowed amount back out; clearing the due date and note works.
	_, err = svc.UpdateDebt(ctx, obligationUser, debt.UUID, port.UpdateDebtRequest{
		AssetUUID: port.Optional[string]{Set: true},
		DueDate:   port.Optional[time.Time]{Set: true},
		Note:      port.Optional[string]{Set: true, Value: strPtr("  ")},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := f.debts[debt.UUID]
	if f.ledger.balance(bank.ID) != "0.00" || got.AssetID != nil || got.DueDate != nil || got.Note != nil {
		t.Fatalf("bank=%s asset=%v due=%v note=%v", f.ledger.balance(bank.ID), got.AssetID, got.DueDate, got.Note)
	}
}

func TestReceivableLendingNeedsCash(t *testing.T) {
	ctx := context.Background()
	f, _, svc := newObligationServices()
	cash := f.ledger.addAsset(obligationUser, models.AssetTypeCash, "100")

	_, err := svc.CreateReceivable(ctx, obligationUser, port.CreateReceivableRequest{PartyName: "Citra", Amount: decimal.NewFromInt(150), AssetUUID: &cash.UUID})
	wantValidation(t, err, "the cash asset balance is too low to lend this amount")
	if len(f.receivables) != 0 || f.ledger.balance(cash.ID) != "100.00" {
		t.Fatalf("failed lending must not persist: receivables=%d balance=%s", len(f.receivables), f.ledger.balance(cash.ID))
	}

	rec, err := svc.CreateReceivable(ctx, obligationUser, port.CreateReceivableRequest{PartyName: "Citra", Amount: decimal.NewFromInt(80), AssetUUID: &cash.UUID})
	if err != nil {
		t.Fatal(err)
	}
	if f.ledger.balance(cash.ID) != "20.00" {
		t.Fatalf("lending should take cash out, balance=%s", f.ledger.balance(cash.ID))
	}
	if _, err := svc.RecordReceivablePayment(ctx, obligationUser, rec.UUID, port.CreateReceivablePaymentRequest{
		Amount: decimal.NewFromInt(30), Date: time.Now(), AssetUUID: &cash.UUID,
	}); err != nil {
		t.Fatal(err)
	}
	if f.ledger.balance(cash.ID) != "50.00" {
		t.Fatalf("collecting should add cash, balance=%s", f.ledger.balance(cash.ID))
	}

	// A payment without an asset is only a record.
	if _, err := svc.RecordReceivablePayment(ctx, obligationUser, rec.UUID, port.CreateReceivablePaymentRequest{
		Amount: decimal.NewFromInt(50), Date: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	if f.ledger.balance(cash.ID) != "50.00" || f.receivables[rec.UUID].Status != models.ObligationStatusPaid {
		t.Fatalf("balance=%s status=%s", f.ledger.balance(cash.ID), f.receivables[rec.UUID].Status)
	}

	// Deleting reverses the lending (+80) and the cash-linked payment (-30).
	if err := svc.DeleteReceivable(ctx, obligationUser, rec.UUID); err != nil {
		t.Fatal(err)
	}
	if f.ledger.balance(cash.ID) != "100.00" {
		t.Fatalf("delete should restore cash, balance=%s", f.ledger.balance(cash.ID))
	}
}

func TestObligationRejectsNonCashAssetAndUnknownDebt(t *testing.T) {
	ctx := context.Background()
	f, svc, _ := newObligationServices()
	stock := f.ledger.addAsset(obligationUser, models.AssetTypeStock, "10")

	_, err := svc.CreateDebt(ctx, obligationUser, port.CreateDebtRequest{PartyName: "Dodi", Amount: decimal.NewFromInt(10), AssetUUID: &stock.UUID})
	wantValidation(t, err, "asset must be of type CASH")

	_, err = svc.RecordDebtPayment(ctx, obligationUser, "00000000-0000-0000-0000-999999999999", port.CreateDebtPaymentRequest{Amount: decimal.NewFromInt(1), Date: time.Now()})
	if !errors.Is(err, port.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestEffectiveObligationStatus(t *testing.T) {
	now := time.Date(2026, 11, 3, 15, 0, 0, 0, time.UTC)
	yesterday := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	today := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		stored models.ObligationStatus
		due    *time.Time
		want   models.ObligationStatus
	}{
		{models.ObligationStatusPending, &yesterday, models.ObligationStatusOverdue},
		{models.ObligationStatusPartial, &yesterday, models.ObligationStatusOverdue},
		{models.ObligationStatusPaid, &yesterday, models.ObligationStatusPaid},
		{models.ObligationStatusPending, &today, models.ObligationStatusPending},
		{models.ObligationStatusPending, nil, models.ObligationStatusPending},
	}
	for _, c := range cases {
		if got := models.EffectiveObligationStatus(c.stored, c.due, now); got != c.want {
			t.Errorf("stored=%s due=%v: got %s want %s", c.stored, c.due, got, c.want)
		}
	}
}
