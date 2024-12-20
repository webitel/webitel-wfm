package model

import (
	"github.com/jackc/pgx/v5/pgtype"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
)

type AgentSchedulePause struct {
	DomainRecord

	Start int64       `json:"start" db:"start"`
	End   int64       `json:"end" db:"end"`
	Cause *LookupItem `json:"cause" db:"cause,json"`
}

func (a *AgentSchedulePause) MarshalProto() *pb.AgentSchedulePause {
	return &pb.AgentSchedulePause{
		Id:        a.Id,
		DomainId:  a.DomainId,
		CreatedAt: a.CreatedAt.Time.UnixMilli(),
		CreatedBy: a.CreatedBy.MarshalProto(),
		UpdatedAt: a.UpdatedAt.Time.UnixMilli(),
		UpdatedBy: a.UpdatedBy.MarshalProto(),
		Start:     a.Start,
		End:       a.End,
		Cause:     a.Cause.MarshalProto(),
	}
}

type AgentScheduleShift struct {
	DomainRecord

	Start  int64                 `json:"start" db:"start"`
	End    int64                 `json:"end" db:"end"`
	Pauses []*AgentSchedulePause `json:"pauses" db:"pauses"`
}

func (a *AgentScheduleShift) MarshalProto() *pb.AgentScheduleShift {
	pauses := make([]*pb.AgentSchedulePause, 0, len(a.Pauses))
	for _, pause := range a.Pauses {
		pauses = append(pauses, pause.MarshalProto())
	}

	return &pb.AgentScheduleShift{
		Id:        a.Id,
		DomainId:  a.DomainId,
		CreatedAt: a.CreatedAt.Time.UnixMilli(),
		CreatedBy: a.CreatedBy.MarshalProto(),
		UpdatedAt: a.UpdatedAt.Time.UnixMilli(),
		UpdatedBy: a.UpdatedBy.MarshalProto(),
		Start:     a.Start,
		End:       a.End,
		Pauses:    pauses,
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

type AgentWorkingScheduleSearch struct {
	SearchItem SearchItem

	WorkingScheduleId int64

	Ids           []int64
	AgentIds      []int64
	SupervisorIds []int64
	TeamIds       []int64
	SkillIds      []int64
}
