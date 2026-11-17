package handler

import (
	"context"
	"testing"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/internal/service"
)

type agentsService struct {
	service.WorkingScheduleManager

	removed int64
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
