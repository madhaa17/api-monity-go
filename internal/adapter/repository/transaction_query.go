package repository

import (
	"context"
	"fmt"
	"strings"

	"monity/internal/core/port"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// transactionTable holds the query building shared by the incomes and expenses tables.
type transactionTable struct {
	name          string
	searchColumns []string // text columns matched by TransactionFilter.Search
}

var (
	expensesTable = transactionTable{name: "expenses", searchColumns: []string{"note"}}
	incomesTable  = transactionTable{name: "incomes", searchColumns: []string{"note", "source"}}
)

func (t transactionTable) col(column string) string {
	return t.name + "." + column
}

// withRelations selects rows joined with their category and asset, filling the read-only
// CategoryUUID, CategoryName, AssetUUID and AssetName fields. Conditions must qualify columns.
func (t transactionTable) withRelations(db *gorm.DB) *gorm.DB {
	return db.Table(t.name).
		Select(t.name + ".*, categories.uuid AS category_uuid, categories.name AS category_name, assets.uuid AS asset_uuid, assets.name AS asset_name").
		Joins("JOIN categories ON categories.id = " + t.col("category_id")).
		Joins("JOIN assets ON assets.id = " + t.col("asset_id"))
}

// filter restricts q to the user's rows matching f.
func (t transactionTable) filter(q *gorm.DB, userID int64, f port.TransactionFilter) *gorm.DB {
	q = q.Where(t.col("user_id")+" = ?", userID)
	if f.DateFrom != nil {
		q = q.Where(t.col("date")+" >= ?", *f.DateFrom)
	}
	if f.DateTo != nil {
		q = q.Where(t.col("date")+" < ?", f.DateTo.AddDate(0, 0, 1))
	}
	if len(f.CategoryUUIDs) > 0 {
		q = q.Where(t.col("category_id")+" IN (SELECT id FROM categories WHERE user_id = ? AND uuid IN ?)", userID, f.CategoryUUIDs)
	}
	if f.AssetUUID != "" {
		q = q.Where(t.col("asset_id")+" IN (SELECT id FROM assets WHERE user_id = ? AND uuid = ?)", userID, f.AssetUUID)
	}
	if search := strings.TrimSpace(f.Search); search != "" {
		pattern := "%" + escapeLike(search) + "%"
		conds := make([]string, len(t.searchColumns))
		args := make([]any, len(t.searchColumns))
		for i, c := range t.searchColumns {
			conds[i] = t.col(c) + " ILIKE ?"
			args[i] = pattern
		}
		q = q.Where("("+strings.Join(conds, " OR ")+")", args...)
	}
	if f.MinAmount != nil {
		q = q.Where(t.col("amount")+" >= ?", *f.MinAmount)
	}
	if f.MaxAmount != nil {
		q = q.Where(t.col("amount")+" <= ?", *f.MaxAmount)
	}
	return q
}

// order returns the ORDER BY clause for s, with stable tie-breakers.
func (t transactionTable) order(s port.TransactionSort) string {
	dir := "ASC"
	if s.Desc {
		dir = "DESC"
	}
	if s.Field == "amount" {
		return fmt.Sprintf("%s %s, %s DESC, %s DESC", t.col("amount"), dir, t.col("date"), t.col("id"))
	}
	return fmt.Sprintf("%s %s, %s %s, %s %s", t.col("date"), dir, t.col("created_at"), dir, t.col("id"), dir)
}

// list returns one page of matching rows (with relations) and the total match count.
func (t transactionTable) list(ctx context.Context, db *gorm.DB, userID int64, f port.TransactionFilter, s port.TransactionSort, page, limit int, dest any) (int64, error) {
	var total int64
	if err := t.filter(conn(ctx, db).Table(t.name), userID, f).Count(&total).Error; err != nil {
		return 0, fmt.Errorf("count %s: %w", t.name, err)
	}
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	err := t.filter(t.withRelations(conn(ctx, db)), userID, f).
		Order(t.order(s)).
		Offset(offset).Limit(limit).
		Find(dest).Error
	if err != nil {
		return 0, fmt.Errorf("list %s: %w", t.name, err)
	}
	return total, nil
}

func (t transactionTable) summarize(ctx context.Context, db *gorm.DB, userID int64, f port.TransactionFilter, groupBy port.SummaryGroupBy) (*port.TransactionSummary, error) {
	summary := &port.TransactionSummary{GroupBy: groupBy}

	var totals struct {
		Total decimal.Decimal
		Count int64
	}
	err := t.filter(conn(ctx, db).Table(t.name), userID, f).
		Select("COALESCE(SUM(" + t.col("amount") + "), 0) AS total, COUNT(*) AS count").
		Scan(&totals).Error
	if err != nil {
		return nil, fmt.Errorf("sum %s: %w", t.name, err)
	}
	summary.Total, summary.Count = totals.Total, totals.Count
	if totals.Count > 0 {
		summary.Average = totals.Total.Div(decimal.NewFromInt(totals.Count)).Round(2)
	}

	var byCategory []port.CategoryTotal
	err = t.filter(conn(ctx, db).Table(t.name), userID, f).
		Joins("JOIN categories ON categories.id = " + t.col("category_id")).
		Select("categories.uuid AS category_uuid, categories.name AS category, categories.color AS color, SUM(" + t.col("amount") + ") AS total, COUNT(*) AS count").
		Group("categories.id").
		Order("total DESC, categories.name").
		Scan(&byCategory).Error
	if err != nil {
		return nil, fmt.Errorf("sum %s by category: %w", t.name, err)
	}
	setPercentages(byCategory, totals.Total)
	if byCategory == nil {
		byCategory = []port.CategoryTotal{}
	}
	summary.ByCategory = byCategory

	format := "YYYY-MM-DD"
	if groupBy == port.SummaryGroupByMonth {
		format = "YYYY-MM"
	}
	var series []port.PeriodTotal
	err = t.filter(conn(ctx, db).Table(t.name), userID, f).
		Select("to_char("+t.col("date")+" AT TIME ZONE 'UTC', ?) AS period, SUM("+t.col("amount")+") AS total, COUNT(*) AS count", format).
		Group("period").
		Order("period").
		Scan(&series).Error
	if err != nil {
		return nil, fmt.Errorf("sum %s by period: %w", t.name, err)
	}
	if series == nil {
		series = []port.PeriodTotal{}
	}
	summary.Series = series
	return summary, nil
}

// setPercentages fills each item's share of grandTotal.
func setPercentages(items []port.CategoryTotal, grandTotal decimal.Decimal) {
	if grandTotal.IsZero() {
		return
	}
	hundred := decimal.NewFromInt(100)
	for i := range items {
		items[i].Percentage, _ = items[i].Total.Div(grandTotal).Mul(hundred).Round(2).Float64()
	}
}

// escapeLike escapes the LIKE wildcards in s so it is matched literally.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
