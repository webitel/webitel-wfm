package handler

import (
	"context"
	"slices"
	"testing"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/internal/service"
)

type agentsService struct {
	service.WorkingScheduleManager

	added   []int64
	removed int64
}

func (s *agentsService) UpdateWorkingScheduleAddAgents(_ context.Context, _ *options.Read, agentIDs []int64) ([]*model.LookupItem, error) {
	s.added = agentIDs

	return nil, nil
}

func (s *agentsService) UpdateWorkingScheduleRemoveAgent(_ context.Context, _ *options.Read, agentID int64) (int64, error) {
	s.removed = agentID

	return agentID, nil
}

func TestRemoveAgentIsServed(t *testing.T) {
	ctx := grpccontext.SetUser(context.Background(), &model.SignedInUser{ID: 1, DomainID: 2})
	svc := &agentsService{}

	var srv pb.WorkingScheduleServiceServer = NewWorkingSchedule(svc)

	out, err := srv.UpdateWorkingScheduleRemoveAgent(ctx, &pb.UpdateWorkingScheduleRemoveAgentRequest{Id: 3, AgentId: 7})
	if err != nil {
		t.Fatal(err)
	}

	if svc.removed != 7 || out.GetId() != 7 {
		t.Errorf("removed %d, answered %d; want 7", svc.removed, out.GetId())
	}
}

func TestAddAgentsSendsEachAgentOnce(t *testing.T) {
	ctx := grpccontext.SetUser(context.Background(), &model.SignedInUser{ID: 1, DomainID: 2})
	svc := &agentsService{}
	agents := []*pb.LookupEntity{{Id: 7}, {Id: 3}, {Id: 7}}

	if _, err := NewWorkingSchedule(svc).UpdateWorkingScheduleAddAgents(ctx, &pb.UpdateWorkingScheduleAddAgentsRequest{Id: 3, Agents: agents}); err != nil {
		t.Fatal(err)
	}

	if want := []int64{3, 7}; !slices.Equal(svc.added, want) {
		t.Errorf("service got %v, want %v", svc.added, want)
	}
}
