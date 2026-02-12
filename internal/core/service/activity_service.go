package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"monity/internal/core/port"
)

const (
	groupByDay   = "day"
	groupByMonth = "month"
	groupByYear  = "year"
)

type ActivityService struct {
	expenseRepo port.ExpenseRepository
	incomeRepo  port.IncomeRepository
}

func NewActivityService(expenseRepo port.ExpenseRepository, incomeRepo port.IncomeRepository) port.ActivityService {
	return &ActivityService{
		expenseRepo: expenseRepo,
		incomeRepo:  incomeRepo,
	}
}

func (s *ActivityService) ListActivities(ctx context.Context, userID int64, groupBy string) (*port.ActivityResponse, error) {
	groupBy = normalizeGroupBy(groupBy)

	expenses, err := s.expenseRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list expenses: %w", err)
	}
	incomes, err := s.incomeRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list incomes: %w", err)
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

func normalizeGroupBy(g string) string {
	switch strings.ToLower(strings.TrimSpace(g)) {
	case groupByMonth, groupByYear:
		return strings.ToLower(strings.TrimSpace(g))
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
