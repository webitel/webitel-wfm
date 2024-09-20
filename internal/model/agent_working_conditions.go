package model

import pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"

type AgentWorkingConditions struct {
	WorkingCondition LookupItem  `json:"working_condition" db:"working_condition,json"`
	PauseTemplate    *LookupItem `json:"pause_template" db:"pause_template,json"`
}

func (a *AgentWorkingConditions) MarshalProto() *pb.AgentWorkingConditions {
	out := &pb.AgentWorkingConditions{
		WorkingCondition: a.WorkingCondition.MarshalProto(),
		PauseTemplate:    a.PauseTemplate.MarshalProto(),
	}

	return out
}
