package model

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
)

type AgentScheduleShiftPause struct {
	DomainRecord

	StartAt time.Time   `json:"start_at"`
	EndAt   time.Time   `json:"end_at"`
	Cause   *LookupItem `json:"cause"`
}

func (a *AgentScheduleShiftPause) MarshalProto() *pb.AgentScheduleShiftPause {
	return &pb.AgentScheduleShiftPause{
		Id:        a.ID,
		DomainId:  a.DomainID,
		CreatedAt: a.CreatedAt.Time.UnixMilli(),
		CreatedBy: a.CreatedBy.MarshalProto(),
		UpdatedAt: a.UpdatedAt.Time.UnixMilli(),
		UpdatedBy: a.UpdatedBy.MarshalProto(),
		StartAt:   a.StartAt.UnixMilli(),
		EndAt:     a.EndAt.UnixMilli(),
		Cause:     a.Cause.MarshalProto(),
	}
}

// AgentScheduleShiftSkill keeps the stored capacity, which the contract leaves out until WFM-114.
type AgentScheduleShiftSkill struct {
	Skill    LookupItem `json:"skill"`
	Capacity *int64     `json:"capacity"`
	Enabled  bool       `json:"enabled"`
}

func (a *AgentScheduleShiftSkill) MarshalProto() *pb.AgentScheduleShiftSkill {
	return &pb.AgentScheduleShiftSkill{Skill: a.Skill.MarshalProto(), Enabled: a.Enabled}
}

// AgentScheduleShift is [StartAt, EndAt), listed under its working date.
type AgentScheduleShift struct {
	DomainRecord

	StartAt      time.Time                  `json:"start_at"`
	EndAt        time.Time                  `json:"end_at"`
	WorkingDate  pgtype.Date                `json:"working_date"`
	Ver          int32                      `json:"ver"`
	Source       int16                      `json:"source"`
	PlannerRunID *int64                     `json:"planner_run_id"`
	Pauses       []*AgentScheduleShiftPause `json:"pauses"`
	Skills       []*AgentScheduleShiftSkill `json:"skills"`
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

	// The etag waits for its go-kit type with the edit of a shift (WFM-99).
	out := &pb.AgentScheduleShift{
		Id:          a.ID,
		DomainId:    a.DomainID,
		CreatedAt:   a.CreatedAt.Time.UnixMilli(),
		CreatedBy:   a.CreatedBy.MarshalProto(),
		UpdatedAt:   a.UpdatedAt.Time.UnixMilli(),
		UpdatedBy:   a.UpdatedBy.MarshalProto(),
		StartAt:     a.StartAt.UnixMilli(),
		EndAt:       a.EndAt.UnixMilli(),
		WorkingDate: DateMillis(a.WorkingDate),
		Source:      pb.ShiftSource(a.Source),
		Pauses:      pauses,
		Skills:      skills,
	}

	if a.PlannerRunID != nil {
		out.PlannerRunId = *a.PlannerRunID
	}

	return out
}

// AgentSchedule is a calendar cell: one working date of an agent.
type AgentSchedule struct {
	Date    pgtype.Date           `json:"date"`
	Locked  bool                  `json:"locked"`
	Absence *AgentAbsenceType     `json:"absence"`
	Shifts  []*AgentScheduleShift `json:"shifts"`
}

func (a *AgentSchedule) MarshalProto() *pb.AgentSchedule {
	shifts := make([]*pb.AgentScheduleShift, 0, len(a.Shifts))
	for _, shift := range a.Shifts {
		shifts = append(shifts, shift.MarshalProto())
	}

	out := &pb.AgentSchedule{Date: DateMillis(a.Date), Locked: a.Locked, Shifts: shifts}
	if a.Absence != nil {
		out.Absence = pb.AbsenceType(*a.Absence)
	}

	return out
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

// AgentWorkingScheduleCalendar is the calendar page of a schedule read in one request:
// the agents' cells, the holidays of the period, and the calendar's zone and week.
type AgentWorkingScheduleCalendar struct {
	Items    []*AgentWorkingSchedule
	Holidays []*Holiday
	Calendar *ScheduleCalendar
}

type AgentWorkingScheduleSearch struct {
	SearchItem SearchItem

	WorkingScheduleID int64

	AgentIDs      []int64
	SupervisorIDs []int64
	TeamIDs       []int64
	SkillIDs      []int64
}
