package storage

import (
	"regexp"
	"testing"
	"time"

	"github.com/webitel/webitel-wfm/internal/model"
)

// The period narrows the cells aggregated for each agent; it must not drop an
// agent that has nothing in the period.
func TestSearchAgentWorkingScheduleAppliesPeriod(t *testing.T) {
	db, sql, args := newSQLStore()
	a := &AgentWorkingSchedule{db: db}

	from := model.NewTimestamp(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC).Unix())
	to := model.NewTimestamp(time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC).Unix())
	search := &model.AgentWorkingScheduleSearch{
		WorkingScheduleId: 7,
		SearchItem:        model.SearchItem{Date: &model.FilterBetween{From: from, To: to}},
	}

	if _, err := a.SearchAgentWorkingSchedule(userCtx(2, 1), &model.SignedInUser{Id: 1, DomainId: 2}, search); err != nil {
		t.Fatal(err)
	}

	query, queryArgs := (*sql)[0], (*args)[0]

	cells := regexp.MustCompile(`FILTER \(WHERE \(date IS NOT NULL AND date >= (\$\d+) AND date <= (\$\d+)\)\)`)

	m := cells.FindStringSubmatch(query)
	if m == nil {
		t.Fatalf("period is not applied to the aggregated cells:\n%s", query)
	}

	if got := argAt(t, queryArgs, m[1]); got != from {
		t.Errorf("from = %v, want %v", got, from)
	}

	if got := argAt(t, queryArgs, m[2]); got != to {
		t.Errorf("to = %v, want %v", got, to)
	}

	if where := query[regexp.MustCompile(`WHERE domain_id`).FindStringIndex(query)[0]:]; regexp.MustCompile(`\bdate\b`).MatchString(where) {
		t.Errorf("period must not filter the agent rows:\n%s", where)
	}
}
