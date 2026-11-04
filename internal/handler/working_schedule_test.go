package handler

import (
	"strings"
	"testing"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/internal/model"
)

// A schedule state is numbered alike in the database, the model and the
// contract (Core design §1.3), so it keeps its name both ways across the API.
func TestWorkingScheduleStateKeepsItsName(t *testing.T) {
	for _, state := range []model.WorkingScheduleState{
		model.WorkingScheduleStateDraft,
		model.WorkingScheduleStatePending,
		model.WorkingScheduleStateActive,
		model.WorkingScheduleStateArchived,
	} {
		name := "WORKING_SCHEDULE_STATE_" + strings.ToUpper(state.String())

		if got := (&model.WorkingSchedule{State: state}).MarshalProto().GetState().String(); got != name {
			t.Errorf("model %s goes out as %s, want %s", state, got, name)
		}

		in := unmarshalWorkingScheduleProto(&pb.WorkingSchedule{
			State:    pb.WorkingScheduleState(pb.WorkingScheduleState_value[name]),
			Team:     &pb.LookupEntity{},
			Calendar: &pb.LookupEntity{},
		})
		if in.State != state {
			t.Errorf("%s comes in as model %s, want %s", name, in.State, state)
		}
	}
}
