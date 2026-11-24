package model

import "testing"

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
		if got := [2]bool{state.ShiftsWritable(), state.Deletable()}; got != w || state.AgentsWritable() != w[0] {
			t.Errorf("%s: writable, deletable = %v, agents writable = %v, want %v", state, got, state.AgentsWritable(), w)
		}
	}
}
