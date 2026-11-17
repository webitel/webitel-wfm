package model

import "testing"

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
