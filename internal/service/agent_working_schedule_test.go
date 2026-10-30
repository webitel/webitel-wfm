package service

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/internal/storage"
)

var (
	periodStart = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	periodEnd   = time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
)

// scheduleStore serves one working schedule covering October 2026.
type scheduleStore struct {
	storage.WorkingScheduleManager
}

func (scheduleStore) ReadWorkingSchedule(_ context.Context, read *options.Read) (*model.WorkingSchedule, error) {
	return &model.WorkingSchedule{
		DomainRecord: model.DomainRecord{Id: read.ID()},
		Team:         model.LookupItem{Id: 5},
		StartDateAt:  pgtype.Date{Time: periodStart, Valid: true},
		EndDateAt:    pgtype.Date{Time: periodEnd, Valid: true},
	}, nil
}

// shiftStore lists every agent it holds whatever the filter, as the storage
// does when it receives no agent ids, and records the search it was given.
type shiftStore struct {
	storage.AgentWorkingScheduleManager

	agents   []*model.AgentWorkingSchedule
	holidays []*model.Holiday
	searched *model.AgentWorkingScheduleSearch
}

func (s *shiftStore) SearchAgentWorkingSchedule(_ context.Context, _ *model.SignedInUser, search *model.AgentWorkingScheduleSearch) ([]*model.AgentWorkingSchedule, error) {
	s.searched = search

	return s.agents, nil
}

func (s *shiftStore) Holidays(context.Context, *model.SignedInUser, *model.AgentWorkingScheduleSearch) ([]*model.Holiday, error) {
	return s.holidays, nil
}

func signedIn() (context.Context, *model.SignedInUser) {
	user := &model.SignedInUser{Id: 1, DomainId: 2, Token: "token"}

	return grpccontext.SetUser(context.Background(), user), user
}

// A team, skill or supervisor filter that matches no agent must list no
// agents. Left as an empty id list it would read as "no filter" and list all.
func TestSearchAgentsWorkingScheduleFilterMatchingNoAgent(t *testing.T) {
	ctx, user := signedIn()
	shifts := &shiftStore{
		agents:   []*model.AgentWorkingSchedule{{Agent: model.LookupItem{Id: 10}}, {Agent: model.LookupItem{Id: 11}}},
		holidays: []*model.Holiday{{Name: "holiday"}},
	}

	svc := NewAgentWorkingSchedule(shifts, scheduleStore{}, newEngine(t, &fakeEngine{}))

	items, holidays, err := svc.SearchAgentsWorkingSchedule(ctx, user, &model.AgentWorkingScheduleSearch{
		WorkingScheduleId: 7,
		TeamIds:           []int64{99},
		SearchItem:        model.SearchItem{Date: &model.FilterBetween{}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 0 {
		t.Errorf("listed %d agents, want none", len(items))
	}

	if len(holidays) != 1 {
		t.Errorf("holidays = %d, want 1: the calendar does not depend on the agent filter", len(holidays))
	}
}

// The shifts and the holidays are read in parallel. Run with -race: both reads
// used to write one shared error variable.
func TestSearchAgentsWorkingScheduleParallelReads(t *testing.T) {
	ctx, user := signedIn()
	shifts := &shiftStore{agents: []*model.AgentWorkingSchedule{{}}, holidays: []*model.Holiday{{}}}

	svc := NewAgentWorkingSchedule(shifts, scheduleStore{}, nil)
	for range 20 {
		if _, _, err := svc.SearchAgentsWorkingSchedule(ctx, user, &model.AgentWorkingScheduleSearch{
			WorkingScheduleId: 7,
			SearchItem:        model.SearchItem{Date: &model.FilterBetween{}},
		}); err != nil {
			t.Fatal(err)
		}
	}
}

// A search without a period, or with one bound missing, covers the
// schedule's own period — as the schedule forecast does.
func TestSearchAgentsWorkingScheduleWithoutPeriod(t *testing.T) {
	from := model.NewTimestamp(time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC).Unix())

	for name, tc := range map[string]struct {
		date     *model.FilterBetween
		wantFrom time.Time
	}{
		"no period":    {date: nil, wantFrom: periodStart},
		"unset period": {date: &model.FilterBetween{}, wantFrom: periodStart},
		"only from":    {date: &model.FilterBetween{From: from}, wantFrom: from.Time},
	} {
		t.Run(name, func(t *testing.T) {
			ctx, user := signedIn()
			shifts := &shiftStore{}

			svc := NewAgentWorkingSchedule(shifts, scheduleStore{}, nil)
			if _, _, err := svc.SearchAgentsWorkingSchedule(ctx, user, &model.AgentWorkingScheduleSearch{
				WorkingScheduleId: 7,
				SearchItem:        model.SearchItem{Date: tc.date},
			}); err != nil {
				t.Fatal(err)
			}

			got := shifts.searched.SearchItem.Date
			if !got.From.Time.Equal(tc.wantFrom) || !got.To.Time.Equal(periodEnd) {
				t.Errorf("period = %s..%s, want %s..%s", got.From.Time, got.To.Time, tc.wantFrom, periodEnd)
			}
		})
	}
}
