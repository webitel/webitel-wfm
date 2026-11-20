package model

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestShiftsWritable(t *testing.T) {
	want := map[WorkingScheduleState]bool{
		WorkingScheduleStateUnspecified: false,
		WorkingScheduleStateDraft:       true,
		WorkingScheduleStatePending:     true,
		WorkingScheduleStateActive:      true,
		WorkingScheduleStateArchived:    false,
	}

	for state, writable := range want {
		if got := state.ShiftsWritable(); got != writable {
			t.Errorf("%s: writable = %v, want %v", state, got, writable)
		}
	}
}

func TestWorkingScheduleValidate(t *testing.T) {
	day := func(d int) pgtype.Date { return DateFromMillis(int64(d) * 86400000) }

	for name, c := range map[string]struct {
		in *WorkingSchedule
		ok bool
	}{
		"one day":      {in: &WorkingSchedule{Team: LookupItem{ID: 1}, Calendar: LookupItem{ID: 1}, StartDateAt: day(3), EndDateAt: day(3)}, ok: true},
		"no team":      {in: &WorkingSchedule{Calendar: LookupItem{ID: 1}, StartDateAt: day(3), EndDateAt: day(4)}},
		"no calendar":  {in: &WorkingSchedule{Team: LookupItem{ID: 1}, StartDateAt: day(3), EndDateAt: day(4)}},
		"no end":       {in: &WorkingSchedule{Team: LookupItem{ID: 1}, Calendar: LookupItem{ID: 1}, StartDateAt: day(3)}},
		"end before":   {in: &WorkingSchedule{Team: LookupItem{ID: 1}, Calendar: LookupItem{ID: 1}, StartDateAt: day(4), EndDateAt: day(3)}},
		"no start day": {in: &WorkingSchedule{Team: LookupItem{ID: 1}, Calendar: LookupItem{ID: 1}, EndDateAt: day(3)}},
	} {
		if err := c.in.Validate(); (err == nil) != c.ok {
			t.Errorf("%s: %v", name, err)
		}
	}
}
