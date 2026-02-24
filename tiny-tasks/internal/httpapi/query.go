package httpapi

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"tiny-tasks/internal/model"
)

type listFilters struct {
	hasCompleted bool
	completed    bool
	start        *time.Time
	end          *time.Time
}

func parseListFilters(q url.Values) (listFilters, error) {
	var filters listFilters

	if v := q.Get("completed"); v != "" {
		parsed, err := parseBoolStrict(v)
		if err != nil {
			return listFilters{}, errors.New("completed must be true or false")
		}
		filters.hasCompleted = true
		filters.completed = parsed
	}

	if day := q.Get("completed_on"); day != "" {
		start, end, err := parseUTCDayRange(day)
		if err != nil {
			return listFilters{}, errors.New("completed_on must be YYYY-MM-DD")
		}
		filters.start = &start
		filters.end = &end
	}

	return filters, nil
}

func filterTasks(tasks []model.Task, filters listFilters) []model.Task {
	if !filters.hasCompleted && filters.start == nil {
		return tasks
	}

	out := make([]model.Task, 0, len(tasks))
	for _, t := range tasks {
		if filters.hasCompleted {
			isCompleted := t.CompletedAt != nil
			if isCompleted != filters.completed {
				continue
			}
		}

		if filters.start != nil {
			if t.CompletedAt == nil {
				continue
			}
			ct := *t.CompletedAt
			if ct.Before(*filters.start) || !ct.Before(*filters.end) {
				continue
			}
		}

		out = append(out, t)
	}

	return out
}

func parseBoolStrict(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, errors.New("not a bool")
	}
}

func parseUTCDayRange(day string) (time.Time, time.Time, error) {
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	return start, end, nil
}
