package handler

import "github.com/google/wire"

var Set = wire.NewSet(NewPauseTemplate, NewShiftTemplate, NewWorkingCondition, NewAgentWorkingConditions,
	NewAgentAbsence, NewForecastCalculation, NewWorkingSchedule, NewAgentWorkingSchedule,
)

type Handlers struct {
	PauseTemplate          *PauseTemplate
	ShiftTemplate          *ShiftTemplate
	WorkingCondition       *WorkingCondition
	AgentWorkingConditions *AgentWorkingConditions
	AgentAbsence           *AgentAbsence
	ForecastCalculation    *ForecastCalculation
	WorkingSchedule        *WorkingSchedule
	AgentWorkingSchedule   *AgentWorkingSchedule
}
