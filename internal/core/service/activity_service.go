package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"monity/internal/core/port"
	"monity/internal/models"
)

const (
	groupByDay   = "day"
	groupByMonth = "month"
	groupByYear  = "year"
)

// ActivityService implements business logic for listing and grouping user activities (incomes and expenses).
type ActivityService struct {
	expenseRepo port.ExpenseRepository
	incomeRepo  port.IncomeRepository
}

// NewActivityService returns a new ActivityService with the given repositories.
func NewActivityService(expenseRepo port.ExpenseRepository, incomeRepo port.IncomeRepository) port.ActivityService {
	return &ActivityService{
		expenseRepo: expenseRepo,
		incomeRepo:  incomeRepo,
	}
}

// ListActivities returns activities for the user, grouped by day, month, or year, and optionally filtered by date and timezone.
func (s *ActivityService) ListActivities(ctx context.Context, userID int64, groupBy string, dateFilter string, timezone string) (*port.ActivityResponse, error) {
	groupBy = normalizeGroupBy(groupBy)

	expenses, _, err := s.expenseRepo.ListByUserID(ctx, userID, nil, nil, 1, 10000)
	if err != nil {
		return nil, fmt.Errorf("list expenses: %w", err)
	}
	incomes, _, err := s.incomeRepo.ListByUserID(ctx, userID, nil, nil, 1, 10000)
	if err != nil {
		return nil, fmt.Errorf("list incomes: %w", err)
	}

	var loc *time.Location
	if timezone != "" {
		if l, err := time.LoadLocation(timezone); err == nil {
			loc = l
		}
	}

	if dateFilter != "" {
		incomes = filterIncomesByDate(incomes, dateFilter, loc)
		expenses = filterExpensesByDate(expenses, dateFilter, loc)
	}

	// group key -> slice of items (will merge and sort per group)
	groupsMap := make(map[string][]port.ActivityItem)

	for i := range incomes {
		key := groupKey(incomes[i].Date, groupBy)
		item := port.ActivityItem{
			Type:      "income",
			UUID:      incomes[i].UUID,
			Amount:    incomes[i].Amount,
			Date:      incomes[i].Date,
			CreatedAt: incomes[i].CreatedAt,
			Note:      incomes[i].Note,
			Source:    incomes[i].Source,
		}
		groupsMap[key] = append(groupsMap[key], item)
	}
	for i := range expenses {
		key := groupKey(expenses[i].Date, groupBy)
		item := port.ActivityItem{
			Type:      "expense",
			UUID:      expenses[i].UUID,
			Amount:    expenses[i].Amount,
			Date:      expenses[i].Date,
			CreatedAt: expenses[i].CreatedAt,
			Note:      expenses[i].Note,
			Category:  string(expenses[i].Category),
		}
		groupsMap[key] = append(groupsMap[key], item)
	}

	// sort group keys descending (newest first)
	keys := make([]string, 0, len(groupsMap))
	for k := range groupsMap {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))

	groups := make([]port.ActivityGroup, 0, len(keys))
	for _, k := range keys {
		items := groupsMap[k]
		// sort items by Date then CreatedAt ascending (chronological within group)
		sort.Slice(items, func(i, j int) bool {
			if !items[i].Date.Equal(items[j].Date) {
				return items[i].Date.Before(items[j].Date)
			}
			return items[i].CreatedAt.Before(items[j].CreatedAt)
		})
		groups = append(groups, port.ActivityGroup{Key: k, Items: items})
	}

	return &port.ActivityResponse{Groups: groups}, nil
}

func dateMatches(t time.Time, dateFilter string, loc *time.Location) bool {
	if loc == nil {
		loc = time.Local
	}
	return t.In(loc).Format("2006-01-02") == dateFilter
}

func filterIncomesByDate(incomes []models.Income, dateFilter string, loc *time.Location) []models.Income {
	if dateFilter == "" {
		return incomes
	}
	out := make([]models.Income, 0, len(incomes))
	for i := range incomes {
		if dateMatches(incomes[i].Date, dateFilter, loc) {
			out = append(out, incomes[i])
		}
	}
	return out
}

func filterExpensesByDate(expenses []models.Expense, dateFilter string, loc *time.Location) []models.Expense {
	if dateFilter == "" {
		return expenses
	}
	out := make([]models.Expense, 0, len(expenses))
	for i := range expenses {
		if dateMatches(expenses[i].Date, dateFilter, loc) {
			out = append(out, expenses[i])
		}
	}
	return out
}

func normalizeGroupBy(g string) string {
	normalized := strings.ToLower(strings.TrimSpace(g))
	switch normalized {
	case groupByMonth, groupByYear:
		return normalized
	default:
		return groupByDay
	}
}

func groupKey(t time.Time, groupBy string) string {
	switch groupBy {
	case groupByMonth:
		return t.Format("2006-01")
	case groupByYear:
		return t.Format("2006")
	default:
		return t.Format("2006-01-02")
	}
}
