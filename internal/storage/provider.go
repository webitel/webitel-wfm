package storage

import "github.com/google/wire"

var Set = wire.NewSet(NewPauseTemplate, wire.Bind(new(PauseTemplateManager), new(*PauseTemplate)),
	NewShiftTemplate, wire.Bind(new(ShiftTemplateManager), new(*ShiftTemplate)),
	NewWorkingCondition, wire.Bind(new(WorkingConditionManager), new(*WorkingCondition)),
	NewAgentWorkingConditions, wire.Bind(new(AgentWorkingConditionsManager), new(*AgentWorkingConditions)),
	NewAgentAbsence, wire.Bind(new(AgentAbsenceManager), new(*AgentAbsence)),
	NewForecastCalculation, wire.Bind(new(ForecastCalculationManager), new(*ForecastCalculation)),
	NewWorkingSchedule, wire.Bind(new(WorkingScheduleManager), new(*WorkingSchedule)),
	NewAgentWorkingSchedule, wire.Bind(new(AgentWorkingScheduleManager), new(*AgentWorkingSchedule)),
)
