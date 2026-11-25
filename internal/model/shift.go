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
	ErrShiftSkillDuplicate = errors.InvalidArgument("invalid input: a skill should appear once in a shift", errors.WithID("model.shift.skill_duplicate"))
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

	skills := make(map[int64]bool, len(s.Skills))
	for _, k := range s.Skills {
		if skills[k.Skill.ID] {
			return s.invalid(ErrShiftSkillDuplicate)
		}

		skills[k.Skill.ID] = true
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

// WeekTemplate is a weekly template expanded into shifts for several agents over a range of
// dates. Hours are wall-clock minutes from midnight in the calendar's zone; a range ending
// before it starts crosses midnight and belongs to the day it starts on.
type WeekTemplate struct {
	Agents   []int64
	From, To time.Time // dates at 00:00 UTC, both included
	Days     map[time.Weekday][]ShiftTemplateTime
	Pauses   []ShiftTemplatePause // of every day, each in the range that contains it
	Skills   []*ShiftSkill
}

type ShiftTemplatePause struct {
	Start    int32 // minutes from midnight
	Duration int32 // minutes
	Cause    *LookupItem
}

// Shifts expands the template in zone, agent by agent for every day of the range.
func (w *WeekTemplate) Shifts(zone *time.Location) ([]*Shift, error) {
	var out []*Shift

	for day := w.From; !day.After(w.To); day = day.AddDate(0, 0, 1) {
		ranges := w.Days[day.Weekday()]
		if len(ranges) == 0 {
			continue
		}

		y, m, d := day.Date()
		at := func(minutes int32) time.Time { return time.Date(y, m, d, 0, int(minutes), 0, 0, zone) }

		shifts := make([]Shift, len(ranges))
		for i, r := range ranges {
			shifts[i] = Shift{Start: at(r.Start), End: at(r.end()), Skills: w.Skills}
		}

		for _, p := range w.Pauses {
			start := p.Start

			i := slices.IndexFunc(ranges, func(r ShiftTemplateTime) bool { return r.holds(start, p.Duration) })
			if i < 0 {
				// A pause after midnight belongs to a night range.
				start += 1440
				i = slices.IndexFunc(ranges, func(r ShiftTemplateTime) bool { return r.holds(start, p.Duration) })
			}

			if i < 0 {
				return nil, errors.Wrap(ErrShiftPauseOutside, errors.WithValue("date", day), errors.WithValue("pause_start", p.Start))
			}

			shifts[i].Pauses = append(shifts[i].Pauses, &ShiftPause{Start: at(start), End: at(start + p.Duration), Cause: p.Cause})
		}

		for _, agent := range w.Agents {
			for _, s := range shifts {
				s.AgentID = agent
				out = append(out, &s)
			}
		}
	}

	return out, nil
}

// end is the end in minutes from the start's midnight: past 1440 when the range crosses it.
func (t *ShiftTemplateTime) end() int32 {
	if t.End <= t.Start {
		return t.End + 1440
	}

	return t.End
}

func (t *ShiftTemplateTime) holds(start, duration int32) bool {
	return t.Start <= start && start+duration <= t.end()
}
