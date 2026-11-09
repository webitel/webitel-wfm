package model

import "time"

// Shift is a shift to write: the agent works [Start, End), with the pauses inside it
// and the skills switched for it. The storage sets its working date, the day Start
// falls on in the schedule calendar's zone.
type Shift struct {
	AgentID int64
	Start   time.Time
	End     time.Time
	Pauses  []*ShiftPause
	Skills  []*ShiftSkill
}

type ShiftPause struct {
	Start time.Time
	End   time.Time
	Cause *LookupItem
}

// ShiftSkill switches one of the agent's skills on or off for the shift; a nil Capacity keeps the agent's own.
type ShiftSkill struct {
	Skill    LookupItem
	Capacity *int64
	Enabled  bool
}
