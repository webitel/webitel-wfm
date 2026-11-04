package service

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc/codes"

	enginepb "buf.build/gen/go/webitel/engine/protocolbuffers/go"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/internal/storage"
)

// forecastStore returns the rows a forecast procedure produced.
type forecastStore struct {
	storage.ForecastCalculationManager

	rows []*model.ForecastCalculationResult
}

func (f forecastStore) ExecuteForecastCalculation(context.Context, *model.SignedInUser, int64, int64, *model.FilterBetween) ([]*model.ForecastCalculationResult, error) {
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
