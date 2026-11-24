package model

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

func shiftAt(t *testing.T, agent int64, start, end string, pauses ...string) *Shift {
	t.Helper()

	at := func(s string) time.Time {
		ts, err := time.Parse("2006-01-02 15:04-07", s)
		if err != nil {
			t.Fatal(err)
		}

		return ts
	}

	shift := &Shift{AgentID: agent, Start: at(start), End: at(end)}
	for i := 0; i+1 < len(pauses); i += 2 {
		shift.Pauses = append(shift.Pauses, &ShiftPause{Start: at(pauses[i]), End: at(pauses[i+1])})
	}

	return shift
}

func TestValidateShifts(t *testing.T) {
	cases := []struct {
		name string
		in   []*Shift
		want error
	}{
		{
			name: "a day shift with pauses back to back, the later one first",
			in: []*Shift{shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02",
				"2026-11-02 13:00+02", "2026-11-02 13:30+02", "2026-11-02 12:00+02", "2026-11-02 13:00+02")},
		},
		{
			name: "a shift of 24 hours",
			in:   []*Shift{shiftAt(t, 1, "2026-11-02 08:00+02", "2026-11-03 08:00+02")},
		},
		{
			name: "a shift that ends as it starts",
			in:   []*Shift{shiftAt(t, 1, "2026-11-02 08:00+02", "2026-11-02 08:00+02")},
			want: ErrShiftDuration,
		},
		{
			name: "a shift over the night the clocks go back",
			in:   []*Shift{shiftAt(t, 1, "2026-10-24 22:00+03", "2026-10-25 22:00+02")},
			want: ErrShiftDuration,
		},
		{
			name: "a pause past the end of the shift",
			in:   []*Shift{shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02", "2026-11-02 16:30+02", "2026-11-02 17:30+02")},
			want: ErrShiftPauseOutside,
		},
		{
			name: "a pause before the start of the shift",
			in:   []*Shift{shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02", "2026-11-02 08:30+02", "2026-11-02 09:30+02")},
			want: ErrShiftPauseOutside,
		},
		{
			name: "a pause that ends as it starts",
			in:   []*Shift{shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02", "2026-11-02 12:00+02", "2026-11-02 12:00+02")},
			want: ErrShiftPauseOutside,
		},
		{
			name: "pauses over each other, the later one first",
			in: []*Shift{shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02",
				"2026-11-02 12:30+02", "2026-11-02 13:30+02", "2026-11-02 12:00+02", "2026-11-02 13:00+02")},
			want: ErrShiftPauseOverlap,
		},
		{
			name: "shifts of an agent over each other, the later one first",
			in: []*Shift{
				shiftAt(t, 1, "2026-11-02 16:00+02", "2026-11-02 20:00+02"),
				shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02"),
			},
			want: ErrShiftRequestOverlap,
		},
		{
			name: "shifts of an agent over each other, another agent's in between",
			in: []*Shift{
				shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02"),
				shiftAt(t, 2, "2026-11-02 10:00+02", "2026-11-02 11:00+02"),
				shiftAt(t, 1, "2026-11-02 16:00+02", "2026-11-02 20:00+02"),
			},
			want: ErrShiftRequestOverlap,
		},
		{
			name: "shifts of an agent back to back, the later one first, another agent at the same time",
			in: []*Shift{
				shiftAt(t, 1, "2026-11-02 17:00+02", "2026-11-02 21:00+02"),
				shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02"),
				shiftAt(t, 2, "2026-11-02 09:00+02", "2026-11-02 17:00+02"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateShifts(tc.in); !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestWeekTemplateShifts(t *testing.T) {
	kyiv, err := time.LoadLocation("Europe/Kyiv")
	if err != nil {
		t.Fatal(err)
	}

	date := func(s string) time.Time {
		d, err := time.Parse(time.DateOnly, s)
		if err != nil {
			t.Fatal(err)
		}

		return d
	}

	// One line a shift: agent, start and end in Kyiv, then its pauses.
	render := func(in []*Shift) []string {
		out := make([]string, 0, len(in))

		for _, s := range in {
			line := fmt.Sprintf("%d %s..%s", s.AgentID, s.Start.In(kyiv).Format("01-02 15:04-07"), s.End.In(kyiv).Format("01-02 15:04-07"))
			for _, p := range s.Pauses {
				line += fmt.Sprintf(" p%s..%s", p.Start.In(kyiv).Format("15:04"), p.End.In(kyiv).Format("15:04"))
			}

			out = append(out, line)
		}

		return out
	}

	lunch := []ShiftTemplatePause{{Start: 780, Duration: 30}}

	for _, c := range []struct {
		name    string
		tpl     WeekTemplate
		want    []string
		wantErr error
	}{
		{
			name: "weekdays for two agents, the lunch in each shift, the weekend off",
			tpl: WeekTemplate{
				Agents: []int64{1, 2}, From: date("2026-11-13"), To: date("2026-11-16"),
				Days:   map[time.Weekday][]ShiftTemplateTime{time.Monday: {{540, 1020}}, time.Friday: {{540, 1020}}},
				Pauses: lunch,
			},
			want: []string{
				"1 11-13 09:00+02..11-13 17:00+02 p13:00..13:30", "2 11-13 09:00+02..11-13 17:00+02 p13:00..13:30",
				"1 11-16 09:00+02..11-16 17:00+02 p13:00..13:30", "2 11-16 09:00+02..11-16 17:00+02 p13:00..13:30",
			},
		},
		{
			name: "a night range keeps wall-clock hours across the end of summer time, its pause after midnight",
			tpl: WeekTemplate{
				Agents: []int64{1}, From: date("2026-10-24"), To: date("2026-10-24"),
				Days:   map[time.Weekday][]ShiftTemplateTime{time.Saturday: {{1320, 360}}},
				Pauses: []ShiftTemplatePause{{Start: 120, Duration: 30}},
			},
			want: []string{"1 10-24 22:00+03..10-25 06:00+02 p02:00..02:30"},
		},
		{
			name: "a range to 1440 ends at the next midnight, two ranges a day",
			tpl: WeekTemplate{
				Agents: []int64{1}, From: date("2026-11-15"), To: date("2026-11-15"),
				Days:   map[time.Weekday][]ShiftTemplateTime{time.Sunday: {{480, 720}, {960, 1440}}},
				Pauses: []ShiftTemplatePause{{Start: 1200, Duration: 15}},
			},
			want: []string{"1 11-15 08:00+02..11-15 12:00+02", "1 11-15 16:00+02..11-16 00:00+02 p20:00..20:15"},
		},
		{
			name: "a pause outside every range of one day refuses",
			tpl: WeekTemplate{
				Agents: []int64{1}, From: date("2026-11-13"), To: date("2026-11-14"),
				Days:   map[time.Weekday][]ShiftTemplateTime{time.Friday: {{540, 1020}}, time.Saturday: {{600, 780}}},
				Pauses: lunch,
			},
			wantErr: ErrShiftPauseOutside,
		},
	} {
		got, err := c.tpl.Shifts(kyiv)
		if !errors.Is(err, c.wantErr) || !slices.Equal(render(got), c.want) {
			t.Errorf("%s:\n got %v, %q\nwant %v, %q", c.name, err, render(got), c.wantErr, c.want)
		}
	}

	skills := []*ShiftSkill{{Skill: LookupItem{ID: 9}, Enabled: true}}
	tpl := WeekTemplate{Agents: []int64{1}, From: date("2026-11-16"), To: date("2026-11-16"), Days: map[time.Weekday][]ShiftTemplateTime{time.Monday: {{540, 1020}}}, Skills: skills}

	if got, _ := tpl.Shifts(kyiv); len(got) != 1 || len(got[0].Skills) != 1 || got[0].Skills[0].Skill.ID != 9 {
		t.Errorf("skills = %+v, want the template's on every shift", got)
	}
}
