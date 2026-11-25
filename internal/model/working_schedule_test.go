package model

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestStateRules(t *testing.T) {
	// shifts and agents writable, deletable
	want := map[WorkingScheduleState][2]bool{
		WorkingScheduleStateUnspecified: {false, false},
		WorkingScheduleStateDraft:       {true, true},
		WorkingScheduleStatePending:     {true, true},
		WorkingScheduleStateActive:      {true, false},
		WorkingScheduleStateArchived:    {false, true},
	}

	for state, w := range want {
		if got := [2]bool{state.ShiftsWritable(), state.Deletable()}; got != w || state.AgentsWritable() != w[0] || state.BlockWritable() != w[0] {
			t.Errorf("%s: writable, deletable = %v, agents writable = %v, want %v", state, got, state.AgentsWritable(), w)
		}
	}
}

func TestWorkingScheduleValidate(t *testing.T) {
	day := func(d int) pgtype.Date { return DateFromMillis(int64(d) * 86400000) }

	for name, c := range map[string]struct {
		in *WorkingSchedule
		ok bool
	}{
		"one day":      {in: &WorkingSchedule{Name: "s", Team: LookupItem{ID: 1}, Calendar: LookupItem{ID: 1}, StartDateAt: day(3), EndDateAt: day(3)}, ok: true},
		"no team":      {in: &WorkingSchedule{Name: "s", Calendar: LookupItem{ID: 1}, StartDateAt: day(3), EndDateAt: day(4)}},
		"no calendar":  {in: &WorkingSchedule{Name: "s", Team: LookupItem{ID: 1}, StartDateAt: day(3), EndDateAt: day(4)}},
		"no end":       {in: &WorkingSchedule{Name: "s", Team: LookupItem{ID: 1}, Calendar: LookupItem{ID: 1}, StartDateAt: day(3)}},
		"end before":   {in: &WorkingSchedule{Name: "s", Team: LookupItem{ID: 1}, Calendar: LookupItem{ID: 1}, StartDateAt: day(4), EndDateAt: day(3)}},
		"no name":      {in: &WorkingSchedule{Team: LookupItem{ID: 1}, Calendar: LookupItem{ID: 1}, StartDateAt: day(3), EndDateAt: day(3)}},
		"no start day": {in: &WorkingSchedule{Name: "s", Team: LookupItem{ID: 1}, Calendar: LookupItem{ID: 1}, EndDateAt: day(3)}},
	} {
		if err := c.in.Validate(); (err == nil) != c.ok {
			t.Errorf("%s: %v", name, err)
		}
	}
}
