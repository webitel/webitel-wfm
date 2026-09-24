package handler

import (
	"go.uber.org/fx"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	grpcsrv "github.com/webitel/webitel-wfm/infra/server/grpc"
)

var Module = fx.Module("handler",
	fx.Provide(
		NewPauseTemplate,
		NewShiftTemplate,
		NewWorkingCondition,
		NewAgentWorkingConditions,
		NewAgentAbsence,
		NewForecastCalculation,
		NewWorkingSchedule,
		NewAgentWorkingSchedule,
	),
	fx.Invoke(RegisterPauseTemplateServer),
	fx.Invoke(RegisterShiftTemplateServer),
	fx.Invoke(RegisterWorkingConditionServer),
	fx.Invoke(RegisterAgentWorkingConditionsServer),
	fx.Invoke(RegisterAgentAbsenceServer),
	fx.Invoke(RegisterForecastCalculationServer),
	fx.Invoke(RegisterWorkingScheduleServer),
	fx.Invoke(RegisterAgentWorkingScheduleServer),
)

func RegisterPauseTemplateServer(srv *grpcsrv.Server, h *PauseTemplate) {
	pb.RegisterPauseTemplateServiceServer(srv.Server, h)
}

func RegisterShiftTemplateServer(srv *grpcsrv.Server, h *ShiftTemplate) {
	pb.RegisterShiftTemplateServiceServer(srv.Server, h)
}

func RegisterWorkingConditionServer(srv *grpcsrv.Server, h *WorkingCondition) {
	pb.RegisterWorkingConditionServiceServer(srv.Server, h)
}

func RegisterAgentWorkingConditionsServer(srv *grpcsrv.Server, h *AgentWorkingConditions) {
	pb.RegisterAgentWorkingConditionsServiceServer(srv.Server, h)
}

func RegisterAgentAbsenceServer(srv *grpcsrv.Server, h *AgentAbsence) {
	pb.RegisterAgentAbsenceServiceServer(srv.Server, h)
}

func RegisterForecastCalculationServer(srv *grpcsrv.Server, h *ForecastCalculation) {
	pb.RegisterForecastCalculationServiceServer(srv.Server, h)
}

func RegisterWorkingScheduleServer(srv *grpcsrv.Server, h *WorkingSchedule) {
	pb.RegisterWorkingScheduleServiceServer(srv.Server, h)
}

func RegisterAgentWorkingScheduleServer(srv *grpcsrv.Server, h *AgentWorkingSchedule) {
	pb.RegisterAgentWorkingScheduleServiceServer(srv.Server, h)
}
