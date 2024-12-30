package model

import (
	"github.com/jackc/pgx/v5/pgtype"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
)

type AgentScheduleShiftPause struct {
	DomainRecord

	Start int64       `json:"start" db:"start"`
	End   int64       `json:"end" db:"end"`
	Cause *LookupItem `json:"cause" db:"cause,json"`
}

func (a *AgentScheduleShiftPause) MarshalProto() *pb.AgentScheduleShiftPause {
	return &pb.AgentScheduleShiftPause{
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

type AgentScheduleShiftSkill struct {
	Skill    LookupItem `json:"skill" db:"skill,json"`
	Capacity int64      `json:"capacity" db:"capacity"`
	Enabled  bool       `json:"enabled" db:"enabled"`
}

func (a *AgentScheduleShiftSkill) MarshalProto() *pb.AgentScheduleShiftSkill {
	return &pb.AgentScheduleShiftSkill{
		Skill:    a.Skill.MarshalProto(),
		Capacity: a.Capacity,
		Enabled:  a.Enabled,
	}
}

type AgentScheduleShift struct {
	DomainRecord

	Start  int64                      `json:"start" db:"start"`
	End    int64                      `json:"end" db:"end"`
	Pauses []*AgentScheduleShiftPause `json:"pauses" db:"pauses"`
	Skills []*AgentScheduleShiftSkill `json:"skills" db:"skills"`
}

func (a *AgentScheduleShift) MarshalProto() *pb.AgentScheduleShift {
	pauses := make([]*pb.AgentScheduleShiftPause, 0, len(a.Pauses))
	for _, pause := range a.Pauses {
		pauses = append(pauses, pause.MarshalProto())
	}

	skills := make([]*pb.AgentScheduleShiftSkill, 0, len(a.Skills))
	for _, skill := range a.Skills {
		skills = append(skills, skill.MarshalProto())
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
		Skills:    skills,
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
	Date    pgtype.Date         `json:"date" db:"date,json"`
	Locked  bool                `json:"locked" db:"locked,json"`
	Absence *AgentAbsenceType   `json:"absence" db:"absence"`
	Shift   *AgentScheduleShift `json:"shift" db:"shift,json"`
}

func (a *AgentSchedule) MarshalProto() *pb.AgentSchedule {
	schedule := &pb.AgentSchedule{
		Date:   a.Date.Time.Unix(),
		Locked: a.Locked,
	}

	if schedule.Locked {
		return schedule
	}

	if a.Absence != nil {
		schedule.Type = &pb.AgentSchedule_Absence{
			Absence: pb.AgentAbsenceType(*a.Absence),
		}

		return schedule
	}

	schedule.Type = &pb.AgentSchedule_Shift{
		Shift: a.Shift.MarshalProto(),
	}

	return schedule
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

type CreateAgentsWorkingScheduleShifts struct {
	WorkingScheduleID int64
	Date              FilterBetween                 `json:"date" db:"date,json"`
	Agents            []*LookupItem                 `json:"agents" db:"agents,json"`
	Shifts            map[int64]*AgentScheduleShift `json:"shifts" db:"shifts,json"`
}
