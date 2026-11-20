package service

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc/codes"

	enginepb "buf.build/gen/go/webitel/engine/protocolbuffers/go"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/infra/webitel/engine"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/internal/storage"
)

// versionedStore holds one schedule at version 2 and records every write.
type versionedStore struct {
	storage.WorkingScheduleManager

	state  model.WorkingScheduleState
	writes []string
}

func (s *versionedStore) ReadWorkingSchedule(_ context.Context, read *options.Read) (*model.WorkingSchedule, error) {
	return &model.WorkingSchedule{
		DomainRecord: model.DomainRecord{ID: read.ID()},
		State:        s.state,
		Team:         model.LookupItem{ID: 5},
		Calendar:     model.LookupItem{ID: 6},
		Ver:          2,
	}, nil
}

func (s *versionedStore) UpdateWorkingSchedule(_ context.Context, _ *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error) {
	s.writes = append(s.writes, "update")

	return in, nil
}

// A save made against an older version is refused before anything else looks at
// the row: before engine is asked for a new team's agents, and before the state
// check, so the client rereads first and then sees the true state.
func TestUpdateWorkingScheduleRefusesAStaleVersion(t *testing.T) {
	for name, state := range map[string]model.WorkingScheduleState{
		"draft with a changed team": model.WorkingScheduleStateDraft,
		"no longer a draft":         model.WorkingScheduleStateActive,
	} {
		t.Run(name, func(t *testing.T) {
			ctx, user := signedIn()
			store := &versionedStore{state: state}

			in := &model.WorkingSchedule{
				DomainRecord: model.DomainRecord{ID: 7},
				Name:         "schedule",
				Team:         model.LookupItem{ID: 9},
				Calendar:     model.LookupItem{ID: 6},
				StartDateAt:  model.DateFromMillis(1),
				EndDateAt:    model.DateFromMillis(1),
				Ver:          1,
			}

			_, err := NewWorkingSchedule(store, nil, nil).UpdateWorkingSchedule(ctx, user, in)
			if !errors.Is(err, storage.ErrWorkingScheduleVersionConflict) {
				t.Errorf("err = %v, want %v", err, storage.ErrWorkingScheduleVersionConflict)
			}

			if len(store.writes) > 0 {
				t.Errorf("a stale save wrote: %v", store.writes)
			}
		})
	}
}

// forecastStore returns the rows a forecast procedure produced.
type forecastStore struct {
	storage.ForecastCalculationManager

	rows   []*model.ForecastCalculationResult
	period *model.FilterBetween
}

func (f forecastStore) ExecuteForecastCalculation(_ context.Context, _ *model.SignedInUser, _, _ int64, period *model.FilterBetween) ([]*model.ForecastCalculationResult, error) {
	if f.period != nil {
		*f.period = *period
	}

	return f.rows, nil
}

func forecastRow(agents *int64) *model.ForecastCalculationResult {
	return &model.ForecastCalculationResult{Timestamp: pgtype.Timestamp{Time: periodStart, Valid: true}, Agents: agents}
}

func readWorkingScheduleForecast(t *testing.T, rows ...*model.ForecastCalculationResult) ([]*model.ForecastCalculationResult, error) {
	t.Helper()

	ctx, _ := signedIn()

	read, err := options.NewRead(ctx, options.WithID(7))
	if err != nil {
		t.Fatal(err)
	}

	eng := newEngine(t, &fakeEngine{team: &enginepb.AgentTeam{Id: 5, Name: "team", ForecastCalculation: &enginepb.Lookup{Id: 3}}})
	svc := NewWorkingSchedule(scheduleStore{}, eng, NewForecastCalculation(forecastStore{rows: rows}))

	return svc.ReadWorkingScheduleForecast(ctx, read, &model.FilterBetween{})
}

// The schedule forecast runs the team's forecast procedure, which needs the
// forecast service wired into the schedule service.
func (s *versionedStore) DeactivatedAgents(context.Context, int64, []int64) ([]int64, error) {
	return []int64{7}, nil
}

// A header without a team, a calendar or dates is refused before engine or the storage is asked.
func TestWorkingScheduleHeaderIsValidatedFirst(t *testing.T) {
	ctx, user := signedIn()
	svc := NewWorkingSchedule(&versionedStore{state: model.WorkingScheduleStateDraft}, nil, nil)
	in := &model.WorkingSchedule{DomainRecord: model.DomainRecord{ID: 7}, Calendar: model.LookupItem{ID: 6}, Ver: 2}

	if _, err := svc.CreateWorkingSchedule(ctx, user, in); !errors.Is(err, model.ErrWorkingScheduleHeader) {
		t.Errorf("create: %v, want %v", err, model.ErrWorkingScheduleHeader)
	}

	if _, err := svc.UpdateWorkingSchedule(ctx, user, in); !errors.Is(err, model.ErrWorkingScheduleHeader) {
		t.Errorf("update: %v, want %v", err, model.ErrWorkingScheduleHeader)
	}
}

// Only a new team asks engine for agents; the storage takes them in place of the old ones.
func TestUpdateWorkingScheduleTakesTheNewTeamsAgents(t *testing.T) {
	ctx, user := signedIn()
	day := model.DateFromMillis(86400000)

	for name, c := range map[string]struct {
		team   int64
		engine *fakeEngine
		want   []int64
	}{
		"same team, engine is not asked":      {team: 5, want: []int64{1}},
		"new team, without deactivated users": {team: 9, engine: &fakeEngine{agents: []int64{3, 7}}, want: []int64{3}},
	} {
		t.Run(name, func(t *testing.T) {
			var eng *engine.Client
			if c.engine != nil {
				eng = newEngine(t, c.engine)
			}

			in := &model.WorkingSchedule{
				DomainRecord: model.DomainRecord{ID: 7},
				Name:         "schedule",
				Team:         model.LookupItem{ID: c.team},
				Calendar:     model.LookupItem{ID: 6},
				StartDateAt:  day,
				EndDateAt:    day,
				Agents:       []*model.LookupItem{{ID: 1}},
				Ver:          2,
			}

			out, err := NewWorkingSchedule(&versionedStore{state: model.WorkingScheduleStateDraft}, eng, nil).UpdateWorkingSchedule(ctx, user, in)
			if err != nil {
				t.Fatal(err)
			}

			got := make([]int64, 0, len(out.Agents))
			for _, a := range out.Agents {
				got = append(got, a.ID)
			}

			if !slices.Equal(got, c.want) {
				t.Errorf("agents %v, want %v", got, c.want)
			}
		})
	}
}

func TestReadWorkingScheduleForecast(t *testing.T) {
	agents := int64(4)

	out, err := readWorkingScheduleForecast(t, forecastRow(&agents))
	if err != nil {
		t.Fatal(err)
	}

	if len(out) != 1 || *out[0].Agents != agents {
		t.Errorf("forecast = %v, want one hour with %d agents", out, agents)
	}
}

// Without a period the forecast covers the schedule's days, sent as UTC midnights: pgx
// sends the wall clock, so a bound in the server's zone moves the day west of UTC.
func TestReadWorkingScheduleForecastDefaultPeriod(t *testing.T) {
	ctx, _ := signedIn()

	read, err := options.NewRead(ctx, options.WithID(7))
	if err != nil {
		t.Fatal(err)
	}

	var got model.FilterBetween

	eng := newEngine(t, &fakeEngine{team: &enginepb.AgentTeam{Id: 5, Name: "team", ForecastCalculation: &enginepb.Lookup{Id: 3}}})
	svc := NewWorkingSchedule(scheduleStore{}, eng, NewForecastCalculation(forecastStore{period: &got}))

	if _, err := svc.ReadWorkingScheduleForecast(ctx, read, &model.FilterBetween{}); err != nil {
		t.Fatal(err)
	}

	if from, to := got.From.Time.Format(time.DateTime), got.To.Time.Format(time.DateTime); got.From.Time.Location() != time.UTC ||
		got.To.Time.Location() != time.UTC || from != "2026-10-01 00:00:00" || to != "2026-10-31 00:00:00" {
		t.Errorf("period sent as %s..%s, want 2026-10-01 00:00:00..2026-10-31 00:00:00", from, to)
	}
}

// A forecast with an hour that has no number of agents is rejected whole, with
// an explanation, not taken in part (Core design §3.4).
func TestForecastWithEmptyAgentsIsRejected(t *testing.T) {
	agents := int64(4)
	rows := []*model.ForecastCalculationResult{forecastRow(&agents), forecastRow(nil)}

	t.Run("forecast calculation", func(t *testing.T) {
		ctx, user := signedIn()

		_, err := NewForecastCalculation(forecastStore{rows: rows}).ExecuteForecastCalculation(ctx, user, 3, 5, &model.FilterBetween{})
		if !errors.Is(err, ErrForecastCalculationEmptyAgents) {
			t.Errorf("err = %v, want %v", err, ErrForecastCalculationEmptyAgents)
		}

		// The team's procedure is at fault, not the request.
		if got := errors.Code(err); got != codes.FailedPrecondition {
			t.Errorf("code = %s, want %s", got, codes.FailedPrecondition)
		}
	})

	t.Run("working schedule forecast", func(t *testing.T) {
		if _, err := readWorkingScheduleForecast(t, rows...); !errors.Is(err, ErrForecastCalculationEmptyAgents) {
			t.Errorf("err = %v, want %v", err, ErrForecastCalculationEmptyAgents)
		}
	})
}
