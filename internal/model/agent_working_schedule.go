package model

import (
	"github.com/jackc/pgx/v5/pgtype"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
)

type AgentScheduleShiftType int32

const (
	AgentScheduleShiftTypeUnspecified AgentScheduleShiftType = iota
	AgentScheduleShiftTypeShift
	AgentScheduleShiftTypePause
)

func (s AgentScheduleShiftType) String() string {
	return []string{"unspecified", "shift", "pause"}[s]
}

type AgentScheduleShift struct {
	DomainRecord

	Type  AgentScheduleShiftType `json:"type" db:"type"`
	Start int64                  `json:"start" db:"start"`
	End   int64                  `json:"end" db:"end"`
}

func (a *AgentScheduleShift) MarshalProto() *pb.AgentScheduleShift {
	return &pb.AgentScheduleShift{
		Id:        a.Id,
		DomainId:  a.DomainId,
		CreatedAt: a.CreatedAt.Time.UnixMilli(),
		CreatedBy: a.CreatedBy.MarshalProto(),
		UpdatedAt: a.UpdatedAt.Time.UnixMilli(),
		UpdatedBy: a.UpdatedBy.MarshalProto(),
		Type:      pb.AgentScheduleShiftType(a.Type),
		Start:     a.Start,
		End:       a.End,
	}
}

type AgentScheduleType int32

const (
	AgentScheduleTypeUnspecified AgentScheduleType = iota
	AgentScheduleTypeAbsent
	AgentScheduleTypeLocked
	AgentScheduleTypeShift
)

func (s AgentScheduleType) String() string {
	return []string{"unspecified", "absent", "locked", "shift"}[s]
}

type AgentSchedule struct {
	Date    pgtype.Date           `json:"date" db:"date,json"`
	Type    *AgentScheduleType    `json:"type" db:"type"`
	Absence *AgentAbsenceType     `json:"absence" db:"absence"`
	Shifts  []*AgentScheduleShift `json:"shifts" db:"shifts,json"`
}

func (a *AgentSchedule) MarshalProto() *pb.AgentSchedule {
	t := pb.AgentScheduleType_AGENT_SCHEDULE_TYPE_UNSPECIFIED
	if a.Type != nil {
		t = pb.AgentScheduleType(*a.Type)
	}

	absence := pb.AgentAbsenceType_AGENT_ABSENCE_TYPE_UNSPECIFIED
	if a.Absence != nil {
		absence = pb.AgentAbsenceType(*a.Absence)
	}

	shifts := make([]*pb.AgentScheduleShift, 0, len(a.Shifts))
	for _, shift := range a.Shifts {
		shifts = append(shifts, shift.MarshalProto())
	}

	return &pb.AgentSchedule{
		Date:    a.Date.Time.Unix(),
		Type:    t,
		Absence: absence,
		Shifts:  shifts,
	}
}

type AgentWorkingSchedule struct {
	Agent    LookupItem       `json:"agent" db:"agent,json"`
	Schedule []*AgentSchedule `json:"schedule,omitempty" db:"schedule,json"`
}

func (a *AgentWorkingSchedule) MarshalProto() *pb.AgentWorkingSchedule {
	schedules := make([]*pb.AgentSchedule, 0, len(a.Schedule))
	for _, schedule := range a.Schedule {
		schedules = append(schedules, schedule.MarshalProto())
	}

	return &pb.AgentWorkingSchedule{
		Agent:    a.Agent.MarshalProto(),
		Schedule: schedules,
	}
}
