package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"monity/internal/core/port"
	"monity/internal/models"

	"github.com/shopspring/decimal"
)

// fakeLedger is an in-memory AssetRepository and Transactor; fakeExpenseRepo, fakeIncomeRepo and
// fakeCategoryRepo share its state.
// WithinTransaction snapshots state and restores it when fn fails, mimicking a rollback.
type fakeLedger struct {
	assets     map[int64]*models.Asset
	expenses   map[string]*models.Expense
	incomes    map[string]*models.Income
	categories map[int64]*models.Category
	nextID     int64
}

func newFakeLedger() *fakeLedger {
	return &fakeLedger{
		assets:     map[int64]*models.Asset{},
		expenses:   map[string]*models.Expense{},
		incomes:    map[string]*models.Income{},
		categories: map[int64]*models.Category{},
	}
}

func (f *fakeLedger) addCategory(userID int64, typ models.CategoryType, name string) *models.Category {
	f.nextID++
	c := &models.Category{ID: f.nextID, UUID: "category-" + name + "-" + string(typ), UserID: userID, Type: typ, Name: name}
	f.categories[c.ID] = c
	return c
}

func (f *fakeLedger) addDefaultCategories(userID int64) {
	for typ, names := range models.DefaultCategories {
		for _, name := range names {
			f.addCategory(userID, typ, name)
		}
	}
}

func (f *fakeLedger) category(typ models.CategoryType, name string) *models.Category {
	for _, c := range f.categories {
		if c.Type == typ && c.Name == name {
			return c
		}
	}
	return nil
}

func (f *fakeLedger) addAsset(userID int64, typ models.AssetType, balance string) *models.Asset {
	f.nextID++
	a := &models.Asset{ID: f.nextID, UUID: "asset-" + string(rune('a'+f.nextID)), UserID: userID, Type: typ, Quantity: decimal.RequireFromString(balance)}
	f.assets[a.ID] = a
	return a
}

func (f *fakeLedger) balance(id int64) string {
	return f.assets[id].Quantity.StringFixed(2)
}

func (f *fakeLedger) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	assets := map[int64]models.Asset{}
	for id, a := range f.assets {
		assets[id] = *a
	}
	expenses := map[string]models.Expense{}
	for k, e := range f.expenses {
		expenses[k] = *e
	}
	incomes := map[string]models.Income{}
	for k, i := range f.incomes {
		incomes[k] = *i
	}
	err := fn(ctx)
	if err != nil {
		for id, a := range assets {
			*f.assets[id] = a
		}
		f.expenses = map[string]*models.Expense{}
		for k, e := range expenses {
			f.expenses[k] = &e
		}
		f.incomes = map[string]*models.Income{}
		for k, i := range incomes {
			f.incomes[k] = &i
		}
	}
	return err
}

// AssetRepository

func (f *fakeLedger) Create(ctx context.Context, asset *models.Asset) error { return nil }
func (f *fakeLedger) GetByID(ctx context.Context, id int64) (*models.Asset, error) {
	if a, ok := f.assets[id]; ok {
		c := *a
		return &c, nil
	}
	return nil, nil
}
func (f *fakeLedger) GetByUUID(ctx context.Context, uuid string, userID int64) (*models.Asset, error) {
	for _, a := range f.assets {
		if a.UUID == uuid && a.UserID == userID {
			c := *a
			return &c, nil
		}
	}
	return nil, nil
}
func (f *fakeLedger) ListByUserID(ctx context.Context, userID int64, page, limit int) ([]models.Asset, int64, error) {
	return nil, 0, nil
}
func (f *fakeLedger) Update(ctx context.Context, asset *models.Asset) error { return nil }
func (f *fakeLedger) AdjustQuantity(ctx context.Context, assetID int64, delta decimal.Decimal) error {
	a, ok := f.assets[assetID]
	if !ok {
		return port.ErrNotFound
	}
	next := a.Quantity.Add(delta)
	if next.IsNegative() {
		return port.ErrInsufficientBalance
	}
	a.Quantity = next
	return nil
}
func (f *fakeLedger) Delete(ctx context.Context, uuid string, userID int64) error { return nil }

// ExpenseRepository

type fakeExpenseRepo struct{ *fakeLedger }

func (r fakeExpenseRepo) Create(ctx context.Context, e *models.Expense) error {
	r.nextID++
	e.ID = r.nextID
	e.UUID = "expense-" + string(rune('a'+r.nextID))
	c := *e
	r.expenses[e.UUID] = &c
	return nil
}
func (r fakeExpenseRepo) GetByUUID(ctx context.Context, uuid string, userID int64) (*models.Expense, error) {
	if e, ok := r.expenses[uuid]; ok && e.UserID == userID {
		c := *e
		return &c, nil
	}
	return nil, nil
}
func (r fakeExpenseRepo) GetByUUIDForUpdate(ctx context.Context, uuid string, userID int64) (*models.Expense, error) {
	return r.GetByUUID(ctx, uuid, userID)
}
func (r fakeExpenseRepo) ListByUserID(ctx context.Context, userID int64, dateFrom, dateTo *time.Time, page, limit int) ([]models.Expense, int64, error) {
	return nil, 0, nil
}
func (r fakeExpenseRepo) Update(ctx context.Context, e *models.Expense) error {
	c := *e
	r.expenses[e.UUID] = &c
	return nil
}
func (r fakeExpenseRepo) Delete(ctx context.Context, uuid string, userID int64) error {
	delete(r.expenses, uuid)
	return nil
}

// IncomeRepository

type fakeIncomeRepo struct{ *fakeLedger }

func (r fakeIncomeRepo) Create(ctx context.Context, i *models.Income) error {
	r.nextID++
	i.ID = r.nextID
	i.UUID = "income-" + string(rune('a'+r.nextID))
	c := *i
	r.incomes[i.UUID] = &c
	return nil
}
func (r fakeIncomeRepo) GetByUUID(ctx context.Context, uuid string, userID int64) (*models.Income, error) {
	if i, ok := r.incomes[uuid]; ok && i.UserID == userID {
		c := *i
		return &c, nil
	}
	return nil, nil
}
func (r fakeIncomeRepo) GetByUUIDForUpdate(ctx context.Context, uuid string, userID int64) (*models.Income, error) {
	return r.GetByUUID(ctx, uuid, userID)
}
func (r fakeIncomeRepo) ListByUserID(ctx context.Context, userID int64, dateFrom, dateTo *time.Time, page, limit int) ([]models.Income, int64, error) {
	return nil, 0, nil
}
func (r fakeIncomeRepo) Update(ctx context.Context, i *models.Income) error {
	c := *i
	r.incomes[i.UUID] = &c
	return nil
}
func (r fakeIncomeRepo) Delete(ctx context.Context, uuid string, userID int64) error {
	delete(r.incomes, uuid)
	return nil
}

// CategoryRepository

type fakeCategoryRepo struct{ *fakeLedger }

func (r fakeCategoryRepo) Create(ctx context.Context, c *models.Category) error {
	if existing, _ := r.GetByName(ctx, c.UserID, c.Type, c.Name); existing != nil {
		return port.ErrConflict
	}
	r.nextID++
	c.ID = r.nextID
	c.UUID = "category-" + c.Name + "-" + string(c.Type)
	copied := *c
	r.categories[c.ID] = &copied
	return nil
}
func (r fakeCategoryRepo) CreateDefaults(ctx context.Context, userID int64) error {
	r.addDefaultCategories(userID)
	return nil
}
func (r fakeCategoryRepo) GetByUUID(ctx context.Context, uuid string, userID int64) (*models.Category, error) {
	for _, c := range r.categories {
		if c.UUID == uuid && c.UserID == userID {
			copied := *c
			return &copied, nil
		}
	}
	return nil, nil
}
func (r fakeCategoryRepo) GetByName(ctx context.Context, userID int64, typ models.CategoryType, name string) (*models.Category, error) {
	for _, c := range r.categories {
		if c.UserID == userID && c.Type == typ && strings.EqualFold(c.Name, name) {
			copied := *c
			return &copied, nil
		}
	}
	return nil, nil
}
func (r fakeCategoryRepo) ListByUserID(ctx context.Context, userID int64, typ *models.CategoryType) ([]models.Category, error) {
	return nil, nil
}
func (r fakeCategoryRepo) Update(ctx context.Context, c *models.Category) error {
	for _, other := range r.categories {
		if other.ID != c.ID && other.UserID == c.UserID && other.Type == c.Type && strings.EqualFold(other.Name, c.Name) {
			return port.ErrConflict
		}
	}
	copied := *c
	r.categories[c.ID] = &copied
	return nil
}
func (r fakeCategoryRepo) Delete(ctx context.Context, id int64) error {
	delete(r.categories, id)
	return nil
}
func (r fakeCategoryRepo) CountUsage(ctx context.Context, id int64) (int64, error) {
	var n int64
	for _, e := range r.expenses {
		if e.CategoryID == id {
			n++
		}
	}
	for _, i := range r.incomes {
		if i.CategoryID == id {
			n++
		}
	}
	return n, nil
}
func (r fakeCategoryRepo) Reassign(ctx context.Context, fromID, toID int64) error {
	for _, e := range r.expenses {
		if e.CategoryID == fromID {
			e.CategoryID = toID
		}
	}
	for _, i := range r.incomes {
		if i.CategoryID == fromID {
			i.CategoryID = toID
		}
	}
	return nil
}

const testUser int64 = 1

var testDate = time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func decPtr(s string) *decimal.Decimal { d := dec(s); return &d }

func wantValidation(t *testing.T, err error, msg string) {
	t.Helper()
	var verr *port.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("want ValidationError %q, got %v", msg, err)
	}
	if verr.Message != msg {
		t.Fatalf("want message %q, got %q", msg, verr.Message)
	}
}

func Test_validateCashAmount(t *testing.T) {
	tests := []struct {
		in      string
		wantMsg string
	}{
		{"0.01", ""},
		{"1500000", ""},
		{"999999999999999999.99", ""},
		{"0", "amount must be positive"},
		{"-5", "amount must be positive"},
		{"1.005", "amount must have at most 2 decimal places"},
		{"1000000000000000000", "amount is too large"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			err := validateCashAmount(dec(tt.in))
			if tt.wantMsg == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			wantValidation(t, err, tt.wantMsg)
		})
	}
}

func TestExpenseService(t *testing.T) {
	ctx := context.Background()
	setup := func() (*fakeLedger, port.ExpenseService, *models.Asset, *models.Asset) {
		f := newFakeLedger()
		wallet := f.addAsset(testUser, models.AssetTypeCash, "100.00")
		bank := f.addAsset(testUser, models.AssetTypeCash, "50.00")
		f.addDefaultCategories(testUser)
		return f, NewExpenseService(fakeExpenseRepo{f}, f, fakeCategoryRepo{f}, f), wallet, bank
	}
	create := func(svc port.ExpenseService, asset *models.Asset, amount string) (*models.Expense, error) {
		return svc.CreateExpense(ctx, testUser, port.CreateExpenseRequest{
			AssetUUID: asset.UUID, Amount: dec(amount), Category: "FOOD", Date: testDate,
		})
	}

	t.Run("create deducts balance", func(t *testing.T) {
		f, svc, wallet, _ := setup()
		if _, err := create(svc, wallet, "30.50"); err != nil {
			t.Fatal(err)
		}
		if got := f.balance(wallet.ID); got != "69.50" {
			t.Fatalf("balance = %s, want 69.50", got)
		}
	})

	t.Run("create over balance is rejected and nothing is saved", func(t *testing.T) {
		f, svc, wallet, _ := setup()
		_, err := create(svc, wallet, "100.01")
		wantValidation(t, err, expenseInsufficientMsg)
		if got := f.balance(wallet.ID); got != "100.00" {
			t.Fatalf("balance = %s, want 100.00", got)
		}
		if len(f.expenses) != 0 {
			t.Fatalf("expense saved despite error")
		}
	})

	t.Run("create requires date and CASH asset", func(t *testing.T) {
		f, svc, wallet, _ := setup()
		_, err := svc.CreateExpense(ctx, testUser, port.CreateExpenseRequest{AssetUUID: wallet.UUID, Amount: dec("1"), Category: "FOOD"})
		wantValidation(t, err, "date is required")

		stock := f.addAsset(testUser, models.AssetTypeStock, "10")
		_, err = create(svc, stock, "1")
		wantValidation(t, err, "asset must be of type CASH")
	})

	t.Run("update amount adjusts by the difference", func(t *testing.T) {
		f, svc, wallet, _ := setup()
		e, _ := create(svc, wallet, "30")
		if _, err := svc.UpdateExpense(ctx, testUser, e.UUID, port.UpdateExpenseRequest{Amount: decPtr("50")}); err != nil {
			t.Fatal(err)
		}
		if got := f.balance(wallet.ID); got != "50.00" {
			t.Fatalf("balance = %s, want 50.00", got)
		}
		if _, err := svc.UpdateExpense(ctx, testUser, e.UUID, port.UpdateExpenseRequest{Amount: decPtr("10")}); err != nil {
			t.Fatal(err)
		}
		if got := f.balance(wallet.ID); got != "90.00" {
			t.Fatalf("balance = %s, want 90.00", got)
		}
	})

	t.Run("update moving to another asset refunds old and charges new", func(t *testing.T) {
		f, svc, wallet, bank := setup()
		e, _ := create(svc, wallet, "30")
		if _, err := svc.UpdateExpense(ctx, testUser, e.UUID, port.UpdateExpenseRequest{AssetUUID: &bank.UUID, Amount: decPtr("40")}); err != nil {
			t.Fatal(err)
		}
		if got := f.balance(wallet.ID); got != "100.00" {
			t.Fatalf("wallet = %s, want 100.00", got)
		}
		if got := f.balance(bank.ID); got != "10.00" {
			t.Fatalf("bank = %s, want 10.00", got)
		}
	})

	t.Run("update moving to asset without enough balance rolls back the refund", func(t *testing.T) {
		f, svc, wallet, bank := setup()
		e, _ := create(svc, wallet, "80")
		_, err := svc.UpdateExpense(ctx, testUser, e.UUID, port.UpdateExpenseRequest{AssetUUID: &bank.UUID})
		wantValidation(t, err, expenseInsufficientMsg)
		if got := f.balance(wallet.ID); got != "20.00" {
			t.Fatalf("wallet = %s, want 20.00 (refund must roll back)", got)
		}
		if got := f.balance(bank.ID); got != "50.00" {
			t.Fatalf("bank = %s, want 50.00", got)
		}
	})

	t.Run("delete refunds balance", func(t *testing.T) {
		f, svc, wallet, _ := setup()
		e, _ := create(svc, wallet, "30")
		if err := svc.DeleteExpense(ctx, testUser, e.UUID); err != nil {
			t.Fatal(err)
		}
		if got := f.balance(wallet.ID); got != "100.00" {
			t.Fatalf("balance = %s, want 100.00", got)
		}
	})

	t.Run("missing expense is ErrNotFound", func(t *testing.T) {
		_, svc, _, _ := setup()
		if _, err := svc.UpdateExpense(ctx, testUser, "nope", port.UpdateExpenseRequest{}); !errors.Is(err, port.ErrNotFound) {
			t.Fatalf("update: want ErrNotFound, got %v", err)
		}
		if err := svc.DeleteExpense(ctx, testUser, "nope"); !errors.Is(err, port.ErrNotFound) {
			t.Fatalf("delete: want ErrNotFound, got %v", err)
		}
	})
}

func TestIncomeService(t *testing.T) {
	ctx := context.Background()
	setup := func() (*fakeLedger, port.IncomeService, port.ExpenseService, *models.Asset) {
		f := newFakeLedger()
		wallet := f.addAsset(testUser, models.AssetTypeCash, "0")
		f.addDefaultCategories(testUser)
		return f, NewIncomeService(fakeIncomeRepo{f}, f, fakeCategoryRepo{f}, f), NewExpenseService(fakeExpenseRepo{f}, f, fakeCategoryRepo{f}, f), wallet
	}

	t.Run("create adds balance", func(t *testing.T) {
		f, svc, _, wallet := setup()
		if _, err := svc.CreateIncome(ctx, testUser, port.CreateIncomeRequest{AssetUUID: wallet.UUID, Amount: dec("250.25"), Source: "Salary", Date: testDate}); err != nil {
			t.Fatal(err)
		}
		if got := f.balance(wallet.ID); got != "250.25" {
			t.Fatalf("balance = %s, want 250.25", got)
		}
	})

	t.Run("deleting an already-spent income is rejected", func(t *testing.T) {
		f, svc, expenses, wallet := setup()
		inc, _ := svc.CreateIncome(ctx, testUser, port.CreateIncomeRequest{AssetUUID: wallet.UUID, Amount: dec("100"), Source: "Salary", Date: testDate})
		if _, err := expenses.CreateExpense(ctx, testUser, port.CreateExpenseRequest{AssetUUID: wallet.UUID, Amount: dec("70"), Category: "FOOD", Date: testDate}); err != nil {
			t.Fatal(err)
		}
		wantValidation(t, svc.DeleteIncome(ctx, testUser, inc.UUID), incomeInsufficientMsg)
		_, err := svc.UpdateIncome(ctx, testUser, inc.UUID, port.UpdateIncomeRequest{Amount: decPtr("20")})
		wantValidation(t, err, incomeInsufficientMsg)
		if got := f.balance(wallet.ID); got != "30.00" {
			t.Fatalf("balance = %s, want 30.00", got)
		}
		if _, ok := f.incomes[inc.UUID]; !ok {
			t.Fatalf("income deleted despite error")
		}
	})

	t.Run("source is required", func(t *testing.T) {
		_, svc, _, wallet := setup()
		_, err := svc.CreateIncome(ctx, testUser, port.CreateIncomeRequest{AssetUUID: wallet.UUID, Amount: dec("1"), Source: "  ", Date: testDate})
		wantValidation(t, err, "source is required")
	})
}
