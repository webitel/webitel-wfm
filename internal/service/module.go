package service

import "go.uber.org/fx"

var Module = fx.Module("service",
	fx.Provide(
		fx.Annotate(NewPauseTemplate, fx.As(new(PauseTemplateManager))),
		fx.Annotate(NewShiftTemplate, fx.As(new(ShiftTemplateManager))),
		fx.Annotate(NewWorkingCondition, fx.As(new(WorkingConditionManager))),
		fx.Annotate(NewAgentWorkingConditions, fx.As(new(AgentWorkingConditionsManager))),
		fx.Annotate(NewAgentAbsence, fx.As(new(AgentAbsenceManager))),
		fx.Annotate(NewForecastCalculation, fx.As(new(ForecastCalculationManager))),
		fx.Annotate(NewWorkingSchedule, fx.As(new(WorkingScheduleManager))),
		fx.Annotate(NewAgentWorkingSchedule, fx.As(new(AgentWorkingScheduleManager))),
	),
)
