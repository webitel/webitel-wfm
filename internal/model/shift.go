package model

import (
	"cmp"
	"slices"
	"time"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

var (
	ErrShiftDuration       = errors.InvalidArgument("invalid input: a shift should end after it starts, within 24 hours", errors.WithID("model.shift.duration"))
	ErrShiftPauseOutside   = errors.InvalidArgument("invalid input: a pause should end after it starts, inside its shift", errors.WithID("model.shift.pause_outside"))
	ErrShiftPauseOverlap   = errors.InvalidArgument("invalid input: the pauses of a shift should not overlap", errors.WithID("model.shift.pause_overlap"))
	ErrShiftRequestOverlap = errors.InvalidArgument("invalid input: the shifts of an agent should not overlap", errors.WithID("model.shift.request_overlap"))
)

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

// Validate checks the shift on its own. Sorted by start, intervals overlap only if
// two neighbors do.
func (s *Shift) Validate() error {
	if !s.Start.Before(s.End) || s.End.Sub(s.Start) > 24*time.Hour {
		return s.invalid(ErrShiftDuration)
	}

	pauses := slices.SortedFunc(slices.Values(s.Pauses), func(a, b *ShiftPause) int { return a.Start.Compare(b.Start) })
	for i, pause := range pauses {
		if !pause.Start.Before(pause.End) || pause.Start.Before(s.Start) || pause.End.After(s.End) {
			return s.invalid(ErrShiftPauseOutside)
		}

		if i > 0 && pause.Start.Before(pauses[i-1].End) {
			return s.invalid(ErrShiftPauseOverlap)
		}
	}

	return nil
}

// ValidateShifts validates each shift and that the shifts of an agent do not overlap.
func ValidateShifts(in []*Shift) error {
	for _, shift := range in {
		if err := shift.Validate(); err != nil {
			return err
		}
	}

	byAgent := slices.SortedFunc(slices.Values(in), func(a, b *Shift) int {
		return cmp.Or(cmp.Compare(a.AgentID, b.AgentID), a.Start.Compare(b.Start))
	})
	for i := 1; i < len(byAgent); i++ {
		if byAgent[i].AgentID == byAgent[i-1].AgentID && byAgent[i].Start.Before(byAgent[i-1].End) {
			return byAgent[i].invalid(ErrShiftRequestOverlap)
		}
	}

	return nil
}

func (s *Shift) invalid(err error) error {
	return errors.Wrap(err, errors.WithValue("agent_id", s.AgentID), errors.WithValue("start_at", s.Start))
}
