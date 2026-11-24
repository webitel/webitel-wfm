package storage

import (
	"context"
	"slices"
	"testing"
	"time"

	"google.golang.org/grpc/codes"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/internal/model"
)

// weekdays is 09:00-17:00 with a 13:00 lunch, Monday to Friday, from the 9th to the 13th of November.
func weekdays(agents ...int64) *model.WeekTemplate {
	days := map[time.Weekday][]model.ShiftTemplateTime{}
	for d := time.Monday; d <= time.Friday; d++ {
		days[d] = []model.ShiftTemplateTime{{Start: 540, End: 1020}}
	}

	return &model.WeekTemplate{
		Agents: agents,
		From:   time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC),
		To:     time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC),
		Days:   days,
		Pauses: []model.ShiftTemplatePause{{Start: 780, Duration: 30, Cause: &model.LookupItem{ID: 1}}},
	}
}

// writtenDays lists "agent date" of the shifts in schedule 1, with their pause count.
func writtenDays(t *testing.T, s *AgentWorkingSchedule) []string {
	t.Helper()

	var out []string
	if err := s.db.Primary().Select(context.Background(), &out, `SELECT wsa.agent_id || ' ' || aws.schedule_at || ' ' ||
       (SELECT count(*) FROM wfm.agent_working_schedule_pause p WHERE p.agent_working_schedule_id = aws.id)
FROM wfm.agent_working_schedule aws
         INNER JOIN wfm.working_schedule_agent wsa ON wsa.id = aws.working_schedule_agent_id
WHERE wsa.working_schedule_id = 1
ORDER BY wsa.agent_id, aws.schedule_at`); err != nil {
		t.Fatal(err)
	}

	return out
}

func TestCreateTemplateShiftsSkipsAbsenceAndLockedDaysOnPostgres(t *testing.T) {
	db, _ := newPostgres(t, shiftSeed)
	s := NewAgentWorkingSchedule(db)

	skipped, err := s.CreateTemplateShifts(context.Background(), &model.SignedInUser{ID: 1, DomainID: 1}, 1, weekdays(1, 2))
	if err != nil {
		t.Fatalf("%v %v", err, violationsOf(err))
	}

	if got, want := violationsOf(model.Refuse(ErrShiftsRefused, skipped...)), []string{
		violation(1, "2026-11-10", ReasonShiftLockedDate),
		violation(2, "2026-11-11", ReasonShiftAbsence),
	}; !slices.Equal(got, want) {
		t.Errorf("skipped %v, want %v", got, want)
	}

	if by := skipped[0].Schedule; by == nil || by.ID != 2 || by.Name == nil || *by.Name != "active" {
		t.Errorf("the locked day names %+v, want the active schedule 2", by)
	}

	want := []string{
		"1 2026-11-09 1", "1 2026-11-11 1", "1 2026-11-12 1", "1 2026-11-13 1",
		"2 2026-11-09 1", "2 2026-11-10 1", "2 2026-11-12 1", "2 2026-11-13 1",
	}
	if got := writtenDays(t, s); !slices.Equal(got, want) {
		t.Errorf("written %v, want %v", got, want)
	}
}

func TestCreateTemplateShiftsRefusesAnOverlapWholeOnPostgres(t *testing.T) {
	db, _ := newPostgres(t, shiftSeed)
	ctx := context.Background()
	user := &model.SignedInUser{ID: 1, DomainID: 1}
	s := NewAgentWorkingSchedule(db)

	existing := shiftAt(t, 1, "2026-11-12 16:00+02", "2026-11-12 20:00+02")

	ids, err := s.CreateShifts(ctx, user, 1, []*model.Shift{existing})
	if err != nil {
		t.Fatalf("%v %v", err, violationsOf(err))
	}

	_, err = s.CreateTemplateShifts(ctx, user, 1, weekdays(1, 2))
	if got, want := violationsOf(err), []string{violation(1, "2026-11-12", ReasonShiftOverlap)}; !errors.Is(err, ErrShiftsRefused) || !slices.Equal(got, want) {
		t.Fatalf("err = %v %v, want %v", err, got, want)
	}

	if v := model.ViolationsOf(err)[0]; v.ShiftID != ids[0] || !v.StartAt.Equal(existing.Start) || !v.EndAt.Equal(existing.End) {
		t.Errorf("the overlap names shift %d %v..%v, want %d %v..%v", v.ShiftID, v.StartAt, v.EndAt, ids[0], existing.Start, existing.End)
	}

	if got := writtenDays(t, s); !slices.Equal(got, []string{"1 2026-11-12 0"}) {
		t.Errorf("a refused template wrote %v", got)
	}
}

func TestCreateTemplateShiftsRefusalsOnPostgres(t *testing.T) {
	db, _ := newPostgres(t, shiftSeed)
	ctx := context.Background()
	user := &model.SignedInUser{ID: 1, DomainID: 1}
	s := NewAgentWorkingSchedule(db)

	late := weekdays(1)
	late.From, late.To = time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC), time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)

	twice := weekdays(1)
	twice.Days[time.Monday] = append(twice.Days[time.Monday], model.ShiftTemplateTime{Start: 960, End: 1200})

	stray := weekdays(1)
	stray.Pauses = append(stray.Pauses, model.ShiftTemplatePause{Start: 1080, Duration: 15})

	stranger, inactive := weekdays(3), weekdays(5)
	stranger.To, inactive.To = stranger.From, inactive.From

	for _, c := range []struct {
		name       string
		user       *model.SignedInUser
		schedule   int64
		tpl        *model.WeekTemplate
		want       error
		code       codes.Code
		violations []string
	}{
		{name: "archived", schedule: 4, tpl: weekdays(1), want: ErrWorkingScheduleArchived},
		{
			name: "an agent of another schedule", schedule: 1, tpl: stranger,
			want: ErrShiftsRefused, violations: []string{violation(3, "2026-11-09", ReasonShiftAgentNotInSchedule)},
		},
		{
			name: "a deactivated agent", schedule: 1, tpl: inactive,
			want: ErrShiftsRefused, violations: []string{violation(5, "2026-11-09", ReasonShiftAgentDeactivated)},
		},
		{
			name: "past the schedule's end", schedule: 1, tpl: late,
			want: ErrShiftsRefused, violations: []string{violation(1, "2026-12-01", ReasonShiftOutsideSchedule)},
		},
		{name: "two ranges of a day overlap", schedule: 1, tpl: twice, want: model.ErrShiftRequestOverlap},
		{name: "a pause outside every range", schedule: 1, tpl: stray, want: model.ErrShiftPauseOutside},
		{name: "another domain", user: &model.SignedInUser{ID: 2, DomainID: 2}, schedule: 1, tpl: weekdays(1), code: codes.NotFound},
		{name: "missing", schedule: 99, tpl: weekdays(1), code: codes.NotFound},
	} {
		u := user
		if c.user != nil {
			u = c.user
		}

		_, err := s.CreateTemplateShifts(ctx, u, c.schedule, c.tpl)
		if c.want != nil && !errors.Is(err, c.want) || c.want == nil && errors.Code(err) != c.code || !slices.Equal(violationsOf(err), c.violations) {
			t.Errorf("%s: %v %v, want %v %v %v", c.name, err, violationsOf(err), c.want, c.code, c.violations)
		}
	}

	if got := writtenDays(t, s); len(got) != 0 {
		t.Errorf("refused templates wrote %v", got)
	}

	// Every day skipped is not a refusal: nothing to write, the day comes back.
	absent := weekdays(2)
	absent.From, absent.To = time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC), time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC)

	skipped, err := s.CreateTemplateShifts(ctx, user, 1, absent)
	if err != nil || !slices.Equal(violationsOf(model.Refuse(ErrShiftsRefused, skipped...)), []string{violation(2, "2026-11-11", ReasonShiftAbsence)}) {
		t.Errorf("only an absent day: %v, %v", skipped, err)
	}
}
