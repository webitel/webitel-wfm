package service

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc/codes"

	enginepb "buf.build/gen/go/webitel/engine/protocolbuffers/go"

	"github.com/webitel/webitel-go-kit/pkg/errors"

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

func (s *versionedStore) DeleteWorkingSchedule(context.Context, *options.Read) (int64, error) {
	s.writes = append(s.writes, "delete")

	return 0, nil
}

func (s *versionedStore) UpdateWorkingSchedule(_ context.Context, _ *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error) {
	s.writes = append(s.writes, "update")

	return in, nil
}

// A save made against an older version is refused before anything else looks at
// the row: before the team/calendar branch, which deletes and recreates it
// without reaching the storage guard, and before the state check, so the client
// rereads first and then sees the true state.
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
				Team:         model.LookupItem{ID: 9},
				Calendar:     model.LookupItem{ID: 6},
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
