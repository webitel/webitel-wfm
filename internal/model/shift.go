package model

import "time"

// Shift is [Start, End); the storage sets its working date.
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

// ShiftSkill with a nil Capacity keeps the agent's own.
type ShiftSkill struct {
	Skill    LookupItem
	Capacity *int64
	Enabled  bool
}
