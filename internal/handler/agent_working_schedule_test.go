package handler

import (
	"context"
	"testing"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/service"
)

type searchRecorder struct {
	service.AgentWorkingScheduleManager

	search *model.AgentWorkingScheduleSearch
}

func (s *searchRecorder) SearchAgentsWorkingSchedule(_ context.Context, _ *model.SignedInUser, search *model.AgentWorkingScheduleSearch) ([]*model.AgentWorkingSchedule, []*model.Holiday, error) {
	s.search = search

	return nil, nil, nil
}

// The period is optional in a search; leaving it out must not panic, and must
// reach the service as an unset period rather than a zero one.
func TestSearchAgentsWorkingScheduleWithoutPeriod(t *testing.T) {
	svc := &searchRecorder{}
	ctx := grpccontext.SetUser(context.Background(), &model.SignedInUser{Id: 1, DomainId: 2})

	if _, err := NewAgentWorkingSchedule(svc).SearchAgentsWorkingSchedule(ctx, &pb.SearchAgentsWorkingScheduleRequest{WorkingScheduleId: 7}); err != nil {
		t.Fatal(err)
	}

	if d := svc.search.SearchItem.Date; d != nil && (d.From.Valid || d.To.Valid) {
		t.Errorf("period = %+v, want unset", d)
	}
}

func TestUnmarshalAgentScheduleShiftKeepsPauseID(t *testing.T) {
	shift := unmarshalAgentScheduleShift(&pb.AgentScheduleShift{
		Id:     10,
		Pauses: []*pb.AgentScheduleShiftPause{{Id: 20}},
	})

	if got := shift.Pauses[0].Id; got != 20 {
		t.Errorf("pause id = %d, want its own id 20, not the shift's", got)
	}
}
