package handler

import (
	"context"

	"github.com/webitel/webitel-go-kit/pkg/errors"
	"github.com/webitel/webitel-go-kit/pkg/etag"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/internal/service"
	"github.com/webitel/webitel-wfm/pkg/timeutils"
)

// ErrWorkingScheduleEtag: a missing, malformed or foreign etag carries no usable
// version, so it is a bad request, not the version conflict.
var ErrWorkingScheduleEtag = errors.InvalidArgument("invalid input: item.etag of this working schedule is required", errors.WithID("handler.working_schedule.etag"))

type WorkingSchedule struct {
	pb.UnimplementedWorkingScheduleServiceServer

	service service.WorkingScheduleManager
}

func NewWorkingSchedule(service service.WorkingScheduleManager) *WorkingSchedule {
	return &WorkingSchedule{
		service: service,
	}
}

func (w *WorkingSchedule) CreateWorkingSchedule(ctx context.Context, req *pb.CreateWorkingScheduleRequest) (*pb.CreateWorkingScheduleResponse, error) {
	s := grpccontext.FromContext(ctx)

	out, err := w.service.CreateWorkingSchedule(ctx, s.SignedInUser, unmarshalWorkingScheduleProto(req.GetItem()))
	if err != nil {
		return nil, err
	}

	return &pb.CreateWorkingScheduleResponse{Item: out.MarshalProto()}, nil
}

func (w *WorkingSchedule) ReadWorkingSchedule(ctx context.Context, req *pb.ReadWorkingScheduleRequest) (*pb.ReadWorkingScheduleResponse, error) {
	read, err := options.NewRead(ctx, options.WithID(req.GetId()), options.WithFields(req.GetFields()))
	if err != nil {
		return nil, err
	}

	out, err := w.service.ReadWorkingSchedule(ctx, read)
	if err != nil {
		return nil, err
	}

	return &pb.ReadWorkingScheduleResponse{Item: out.MarshalProto()}, nil
}

func (w *WorkingSchedule) ReadWorkingScheduleForecast(ctx context.Context, req *pb.ReadWorkingScheduleForecastRequest) (*pb.ReadWorkingScheduleForecastResponse, error) {
	read, err := options.NewRead(ctx, options.WithID(req.GetId()))
	if err != nil {
		return nil, err
	}

	date := &model.FilterBetween{}
	if v := req.GetDate(); v != nil {
		date = &model.FilterBetween{
			From: model.TimestampFromMillis(v.GetFrom()),
			To:   model.TimestampFromMillis(v.GetTo()),
		}
	}

	items, err := w.service.ReadWorkingScheduleForecast(ctx, read, date)
	if err != nil {
		return nil, err
	}

	out := make(map[int64]*pb.WorkingScheduleForecast)

	for _, i := range items {
		day := timeutils.Date(i.Timestamp.Time).UnixMilli()
		if _, ok := out[day]; !ok {
			out[day] = &pb.WorkingScheduleForecast{Forecast: make([]*pb.WorkingScheduleForecast_Forecast, 0)}
		}

		out[day].Forecast = append(out[day].Forecast, &pb.WorkingScheduleForecast_Forecast{
			Hour:   int64(i.Timestamp.Time.Hour()),
			Agents: *i.Agents,
		})
	}

	return &pb.ReadWorkingScheduleForecastResponse{Items: out}, nil
}

func (w *WorkingSchedule) SearchWorkingSchedule(ctx context.Context, req *pb.SearchWorkingScheduleRequest) (*pb.SearchWorkingScheduleResponse, error) {
	opts := []options.Option{
		options.WithPagination(req.GetPage(), req.GetSize()),
		options.WithSearch(req.GetQ()),
		options.WithFields(req.GetFields()),
		options.WithOrder(req.GetSort()),
	}

	search, err := options.NewSearch(ctx, opts...)
	if err != nil {
		return nil, err
	}

	items, next, err := w.service.SearchWorkingSchedule(ctx, search)
	if err != nil {
		return nil, err
	}

	return &pb.SearchWorkingScheduleResponse{Items: marshalWorkingScheduleBulkProto(items), Next: next}, nil
}

func (w *WorkingSchedule) UpdateWorkingSchedule(ctx context.Context, req *pb.UpdateWorkingScheduleRequest) (*pb.UpdateWorkingScheduleResponse, error) {
	s := grpccontext.FromContext(ctx)
	in := unmarshalWorkingScheduleProto(req.GetItem())

	// Required here, not in the contract: the message is shared with create.
	tag, err := etag.ExpectEtag(etag.EtagWorkingSchedule, req.GetItem().GetEtag())
	if err != nil || tag.GetOid() != in.ID {
		return nil, errors.Wrap(ErrWorkingScheduleEtag, errors.WithCause(err))
	}

	in.Ver = tag.GetVer()

	out, err := w.service.UpdateWorkingSchedule(ctx, s.SignedInUser, in)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateWorkingScheduleResponse{Item: out.MarshalProto()}, nil
}

func (w *WorkingSchedule) DeleteWorkingSchedule(ctx context.Context, req *pb.DeleteWorkingScheduleRequest) (*pb.DeleteWorkingScheduleResponse, error) {
	read, err := options.NewRead(ctx, options.WithID(req.GetId()))
	if err != nil {
		return nil, err
	}

	id, err := w.service.DeleteWorkingSchedule(ctx, read)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteWorkingScheduleResponse{Id: id}, nil
}

func (w *WorkingSchedule) UpdateWorkingScheduleAddAgents(ctx context.Context, req *pb.UpdateWorkingScheduleAddAgentsRequest) (*pb.UpdateWorkingScheduleAddAgentsResponse, error) {
	read, err := options.NewRead(ctx, options.WithID(req.GetId()))
	if err != nil {
		return nil, err
	}

	agents := make([]int64, 0, len(req.GetAgents()))
	for _, agent := range req.GetAgents() {
		agents = append(agents, agent.GetId())
	}

	items, err := w.service.UpdateWorkingScheduleAddAgents(ctx, read, agents)
	if err != nil {
		return nil, err
	}

	out := make([]*pb.LookupEntity, 0, len(items))
	for _, item := range items {
		out = append(out, item.MarshalProto())
	}

	return &pb.UpdateWorkingScheduleAddAgentsResponse{Agents: out}, nil
}

func (w *WorkingSchedule) UpdateWorkingScheduleRemoveAgents(ctx context.Context, req *pb.UpdateWorkingScheduleRemoveAgentRequest) (*pb.UpdateWorkingScheduleRemoveAgentResponse, error) {
	read, err := options.NewRead(ctx, options.WithID(req.GetId()))
	if err != nil {
		return nil, err
	}

	out, err := w.service.UpdateWorkingScheduleRemoveAgent(ctx, read, req.GetAgentId())
	if err != nil {
		return nil, err
	}

	return &pb.UpdateWorkingScheduleRemoveAgentResponse{Id: out}, nil
}

func unmarshalWorkingScheduleProto(in *pb.WorkingSchedule) *model.WorkingSchedule {
	skills := make([]*model.LookupItem, 0, len(in.GetExtraSkills()))
	for _, skill := range in.GetExtraSkills() {
		skills = append(skills, &model.LookupItem{ID: skill.GetId()})
	}

	agents := make([]*model.LookupItem, 0, len(in.GetAgents()))
	for _, agent := range in.GetAgents() {
		agents = append(agents, &model.LookupItem{ID: agent.GetId()})
	}

	return &model.WorkingSchedule{
		DomainRecord:         model.DomainRecord{ID: in.GetId()},
		Name:                 in.GetName(),
		State:                model.WorkingScheduleState(in.GetState().Number()),
		Team:                 model.LookupItem{ID: in.GetTeam().GetId()},
		Calendar:             model.LookupItem{ID: in.GetCalendar().GetId()},
		StartDateAt:          model.DateFromMillis(in.GetStartDateAt()),
		EndDateAt:            model.DateFromMillis(in.GetEndDateAt()),
		StartTimeAt:          in.GetStartTimeAt(),
		EndTimeAt:            in.GetEndTimeAt(),
		ExtraSkills:          skills,
		BlockOutsideActivity: in.GetBlockOutsideActivity(),
		Agents:               agents,
	}
}

func marshalWorkingScheduleBulkProto(in []*model.WorkingSchedule) []*pb.WorkingSchedule {
	out := make([]*pb.WorkingSchedule, 0, len(in))
	for _, i := range in {
		out = append(out, i.MarshalProto())
	}

	return out
}
