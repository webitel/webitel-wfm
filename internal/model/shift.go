package model

import "time"

// ShiftSource says who wrote a shift, as wfm.agent_working_schedule.source stores it.
type ShiftSource int16

const (
	ShiftSourceUnspecified ShiftSource = iota
	ShiftSourceManual
	ShiftSourcePlanner
)

// Shift is a shift to write: the agent works [Start, End), with the pauses inside
// it and the skills switched for it. Its working date is the day Start falls on
// in the schedule calendar's time zone; the storage computes it.
type Shift struct {
	AgentID int64
	Start   time.Time
	End     time.Time
	Pauses  []*ShiftPause
	Skills  []*ShiftSkill
}

// ShiftPause is [Start, End) inside its shift.
type ShiftPause struct {
	Start time.Time
	End   time.Time
	Cause *LookupItem
}

// ShiftSkill switches one of the agent's skills on or off for the shift; a nil
// Capacity keeps the agent's own.
type ShiftSkill struct {
	Skill    LookupItem
	Capacity *int64
	Enabled  bool
}
