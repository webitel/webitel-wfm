package handler

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/internal/service"
)

type forecastRecorder struct {
	service.WorkingScheduleManager
	service.ForecastCalculationManager

	period *model.FilterBetween
	rows   []*model.ForecastCalculationResult
}

func (f *forecastRecorder) ReadWorkingScheduleForecast(_ context.Context, _ *options.Read, date *model.FilterBetween) ([]*model.ForecastCalculationResult, error) {
	f.period = date

	return f.rows, nil
}

func (f *forecastRecorder) ExecuteForecastCalculation(_ context.Context, _ *model.SignedInUser, _, _ int64, period *model.FilterBetween) ([]*model.ForecastCalculationResult, error) {
	f.period = period

	return f.rows, nil
}

// The forecast takes its period in epoch milliseconds, like every moment on the platform,
// and keys the days of its answer by their UTC midnight in milliseconds.
func TestForecastPeriodIsInMillis(t *testing.T) {
	from := time.Date(2026, 11, 1, 22, 0, 0, 0, time.UTC)
	to := time.Date(2026, 11, 2, 21, 59, 59, 999e6, time.UTC)
	agents := int64(3)
	hour := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	svc := &forecastRecorder{rows: []*model.ForecastCalculationResult{{Timestamp: pgtype.Timestamp{Time: hour, Valid: true}, Agents: &agents}}}
	period := &pb.FilterBetween{From: from.UnixMilli(), To: to.UnixMilli()}

	got := func(name string) {
		t.Helper()

		// In UTC: pgx sends a timestamp by its wall clock.
		if !svc.period.From.Time.Equal(from) || !svc.period.To.Time.Equal(to) || svc.period.From.Time.Location() != time.UTC {
			t.Errorf("%s: period = %s..%s, want %s..%s", name, svc.period.From.Time, svc.period.To.Time, from, to)
		}
	}

	out, err := NewWorkingSchedule(svc).ReadWorkingScheduleForecast(searchCtx(), &pb.ReadWorkingScheduleForecastRequest{Id: 7, Date: period})
	if err != nil {
		t.Fatal(err)
	}

	got("schedule forecast")

	day := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC).UnixMilli()
	if f := out.GetItems()[day].GetForecast(); len(f) != 1 || f[0].GetHour() != 9 {
		t.Errorf("forecast = %v, want hour 9 under %d", out.GetItems(), day)
	}

	if _, err := NewForecastCalculation(svc).ExecuteForecastCalculation(searchCtx(), &pb.ExecuteForecastCalculationRequest{Id: 3, TeamId: 5, ForecastData: period}); err != nil {
		t.Fatal(err)
	}

	got("forecast calculation")
}
