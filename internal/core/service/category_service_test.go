package service

import (
	"context"
	"testing"

	"monity/internal/core/port"
	"monity/internal/models"
)

func TestExpenseCategoryResolution(t *testing.T) {
	ctx := context.Background()
	f := newFakeLedger()
	wallet := f.addAsset(testUser, models.AssetTypeCash, "1000")
	f.addDefaultCategories(testUser)
	svc := NewExpenseService(fakeExpenseRepo{f}, f, fakeCategoryRepo{f}, f)
	food := f.category(models.CategoryTypeExpense, "Food")
	transport := f.category(models.CategoryTypeExpense, "Transport")
	salary := f.category(models.CategoryTypeIncome, "Salary")

	req := func(categoryUUID, category string) port.CreateExpenseRequest {
		return port.CreateExpenseRequest{AssetUUID: wallet.UUID, Amount: dec("1"), CategoryUUID: categoryUUID, Category: category, Date: testDate}
	}

	t.Run("by uuid", func(t *testing.T) {
		e, err := svc.CreateExpense(ctx, testUser, req(food.UUID, ""))
		if err != nil {
			t.Fatal(err)
		}
		if e.CategoryID != food.ID || e.CategoryName != "Food" || e.CategoryUUID != food.UUID {
			t.Fatalf("got category %d %q %q, want Food", e.CategoryID, e.CategoryName, e.CategoryUUID)
		}
	})

	t.Run("legacy enum code matches default name", func(t *testing.T) {
		e, err := svc.CreateExpense(ctx, testUser, req("", "TRANSPORT"))
		if err != nil {
			t.Fatal(err)
		}
		if e.CategoryID != transport.ID {
			t.Fatalf("got category %d, want Transport %d", e.CategoryID, transport.ID)
		}
	})

	t.Run("errors", func(t *testing.T) {
		_, err := svc.CreateExpense(ctx, testUser, req("", ""))
		wantValidation(t, err, "categoryUuid is required")
		_, err = svc.CreateExpense(ctx, testUser, req("nope", ""))
		wantValidation(t, err, "category not found")
		_, err = svc.CreateExpense(ctx, testUser, req(salary.UUID, ""))
		wantValidation(t, err, "category must be an EXPENSE category")
	})

	t.Run("another user's category is not found", func(t *testing.T) {
		other := f.addCategory(testUser+1, models.CategoryTypeExpense, "Theirs")
		_, err := svc.CreateExpense(ctx, testUser, req(other.UUID, ""))
		wantValidation(t, err, "category not found")
	})

	t.Run("update changes category", func(t *testing.T) {
		e, _ := svc.CreateExpense(ctx, testUser, req(food.UUID, ""))
		updated, err := svc.UpdateExpense(ctx, testUser, e.UUID, port.UpdateExpenseRequest{CategoryUUID: &transport.UUID})
		if err != nil {
			t.Fatal(err)
		}
		if updated.CategoryName != "Transport" || f.expenses[e.UUID].CategoryID != transport.ID {
			t.Fatalf("category not updated: %q", updated.CategoryName)
		}
	})
}

func TestIncomeDefaultsToOtherCategory(t *testing.T) {
	ctx := context.Background()
	f := newFakeLedger()
	wallet := f.addAsset(testUser, models.AssetTypeCash, "0")
	f.addDefaultCategories(testUser)
	svc := NewIncomeService(fakeIncomeRepo{f}, f, fakeCategoryRepo{f}, f)

	inc, err := svc.CreateIncome(ctx, testUser, port.CreateIncomeRequest{AssetUUID: wallet.UUID, Amount: dec("10"), Source: "Client", Date: testDate})
	if err != nil {
		t.Fatal(err)
	}
	if inc.CategoryID != f.category(models.CategoryTypeIncome, models.CategoryOther).ID {
		t.Fatalf("income category = %q, want Other", inc.CategoryName)
	}

	food := f.category(models.CategoryTypeExpense, "Food")
	_, err = svc.UpdateIncome(ctx, testUser, inc.UUID, port.UpdateIncomeRequest{CategoryUUID: &food.UUID})
	wantValidation(t, err, "category must be an INCOME category")
}

func TestCategoryService(t *testing.T) {
	ctx := context.Background()
	setup := func() (*fakeLedger, port.CategoryService) {
		f := newFakeLedger()
		f.addDefaultCategories(testUser)
		return f, NewCategoryService(fakeCategoryRepo{f}, f)
	}
	strPtr := func(s string) *string { return &s }

	t.Run("create validates input", func(t *testing.T) {
		_, svc := setup()
		c, err := svc.CreateCategory(ctx, testUser, port.CreateCategoryRequest{Type: models.CategoryTypeExpense, Name: "  Coffee  ", Color: strPtr("#A0522D")})
		if err != nil {
			t.Fatal(err)
		}
		if c.Name != "Coffee" || *c.Color != "#A0522D" {
			t.Fatalf("got %q %v", c.Name, c.Color)
		}

		_, err = svc.CreateCategory(ctx, testUser, port.CreateCategoryRequest{Type: "SAVING", Name: "X"})
		wantValidation(t, err, "type must be INCOME or EXPENSE")
		_, err = svc.CreateCategory(ctx, testUser, port.CreateCategoryRequest{Type: models.CategoryTypeExpense, Name: " "})
		wantValidation(t, err, "name is required")
		_, err = svc.CreateCategory(ctx, testUser, port.CreateCategoryRequest{Type: models.CategoryTypeExpense, Name: "Y", Color: strPtr("red")})
		wantValidation(t, err, "color must be a hex color like #22C55E")
	})

	t.Run("duplicate names are rejected per type, case-insensitively", func(t *testing.T) {
		_, svc := setup()
		_, err := svc.CreateCategory(ctx, testUser, port.CreateCategoryRequest{Type: models.CategoryTypeExpense, Name: "food"})
		wantValidation(t, err, "a category with this name already exists")
		// Uniqueness is per type, so "Food" is still free as an INCOME category.
		if _, err := svc.CreateCategory(ctx, testUser, port.CreateCategoryRequest{Type: models.CategoryTypeIncome, Name: "Food"}); err != nil {
			t.Fatalf("same name under another type: %v", err)
		}
	})

	t.Run("update renames and clears color", func(t *testing.T) {
		f, svc := setup()
		food := f.category(models.CategoryTypeExpense, "Food")
		food.Color = strPtr("#000000")
		c, err := svc.UpdateCategory(ctx, testUser, food.UUID, port.UpdateCategoryRequest{Name: strPtr("Groceries"), Color: strPtr("")})
		if err != nil {
			t.Fatal(err)
		}
		if c.Name != "Groceries" || c.Color != nil {
			t.Fatalf("got %q %v", c.Name, c.Color)
		}
		_, err = svc.UpdateCategory(ctx, testUser, food.UUID, port.UpdateCategoryRequest{Name: strPtr("TRANSPORT")})
		wantValidation(t, err, "a category with this name already exists")
	})

	t.Run("delete in-use category requires reassign_to", func(t *testing.T) {
		f, svc := setup()
		food := f.category(models.CategoryTypeExpense, "Food")
		other := f.category(models.CategoryTypeExpense, models.CategoryOther)
		salary := f.category(models.CategoryTypeIncome, "Salary")
		f.expenses["e1"] = &models.Expense{UUID: "e1", UserID: testUser, CategoryID: food.ID}

		err := svc.DeleteCategory(ctx, testUser, food.UUID, "")
		wantValidation(t, err, "category is used by 1 transaction(s); pass reassign_to with another category to move them")
		wantValidation(t, svc.DeleteCategory(ctx, testUser, food.UUID, salary.UUID), "reassign_to must be a category of the same type")
		wantValidation(t, svc.DeleteCategory(ctx, testUser, food.UUID, food.UUID), "reassign_to must be a different category")

		if err := svc.DeleteCategory(ctx, testUser, food.UUID, other.UUID); err != nil {
			t.Fatal(err)
		}
		if f.expenses["e1"].CategoryID != other.ID {
			t.Fatalf("expense not moved to Other")
		}
		if _, ok := f.categories[food.ID]; ok {
			t.Fatalf("category not deleted")
		}
	})

	t.Run("delete unused category", func(t *testing.T) {
		f, svc := setup()
		gift := f.category(models.CategoryTypeIncome, "Gift")
		if err := svc.DeleteCategory(ctx, testUser, gift.UUID, ""); err != nil {
			t.Fatal(err)
		}
		if err := svc.DeleteCategory(ctx, testUser, gift.UUID, ""); err == nil {
			t.Fatal("deleting twice should fail")
		}
	})
}
