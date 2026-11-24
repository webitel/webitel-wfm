package model

import (
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/webitel/webitel-go-kit/pkg/errors"
	"github.com/webitel/webitel-go-kit/pkg/etag"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
)

var ErrWorkingScheduleHeader = errors.InvalidArgument("invalid input: a working schedule needs a name, a team, a calendar and a start date not after its end date", errors.WithID("model.working_schedule.header"))

type WorkingScheduleState int32

const (
	WorkingScheduleStateUnspecified WorkingScheduleState = iota
	WorkingScheduleStateDraft
	WorkingScheduleStatePending
	WorkingScheduleStateActive
	WorkingScheduleStateArchived
)

func (s WorkingScheduleState) String() string {
	return []string{"unspecified", "draft", "pending", "active", "archived"}[s]
}

// ShiftsWritable: an archived schedule is read only.
func (s WorkingScheduleState) ShiftsWritable() bool {
	return s == WorkingScheduleStateDraft || s == WorkingScheduleStatePending || s == WorkingScheduleStateActive
}

// AgentsWritable: an archived schedule keeps its agents.
func (s WorkingScheduleState) AgentsWritable() bool {
	return s.ShiftsWritable()
}

// BlockWritable: blocking outside a shift switches in any state but archived.
func (s WorkingScheduleState) BlockWritable() bool {
	return s.ShiftsWritable()
}

// Deletable: an active schedule is worked by; its state changes first.
func (s WorkingScheduleState) Deletable() bool {
	return s == WorkingScheduleStateDraft || s == WorkingScheduleStatePending || s == WorkingScheduleStateArchived
}

type WorkingSchedule struct {
	DomainRecord

	Name  string               `db:"name"`
	State WorkingScheduleState `db:"state"`

	Team                 LookupItem    `db:"team,json"`
	Calendar             LookupItem    `db:"calendar,json"`
	StartDateAt          pgtype.Date   `db:"start_date_at,json"`
	EndDateAt            pgtype.Date   `db:"end_date_at,json"`
	StartTimeAt          int64         `db:"start_time_at"`
	EndTimeAt            int64         `db:"end_time_at"`
	ExtraSkills          []*LookupItem `db:"extra_skills,json"`
	BlockOutsideActivity bool          `db:"block_outside_activity"`
	Agents               []*LookupItem `db:"agents,json"`
	TotalAgents          int64         `db:"total_agents"`
	Ver                  int32         `db:"ver"`
}

func (w *WorkingSchedule) Validate() error {
	if w.Name == "" || w.Team.ID == 0 || w.Calendar.ID == 0 || !w.StartDateAt.Valid || !w.EndDateAt.Valid || w.StartDateAt.Time.After(w.EndDateAt.Time) {
		return ErrWorkingScheduleHeader
	}

	return nil
}

func (w *WorkingSchedule) MarshalProto() *pb.WorkingSchedule {
	skills := make([]*pb.LookupEntity, 0, len(w.ExtraSkills))
	for _, skill := range w.ExtraSkills {
		skills = append(skills, skill.MarshalProto())
	}

	agents := make([]*pb.LookupEntity, 0, len(w.Agents))
	for _, agent := range w.Agents {
		agents = append(agents, agent.MarshalProto())
	}

	out := &pb.WorkingSchedule{
		Id:                   w.ID,
		DomainId:             w.DomainID,
		CreatedBy:            w.CreatedBy.MarshalProto(),
		UpdatedBy:            w.UpdatedBy.MarshalProto(),
		Name:                 w.Name,
		State:                pb.WorkingScheduleState(w.State),
		Team:                 w.Team.MarshalProto(),
		Calendar:             w.Calendar.MarshalProto(),
		StartDateAt:          DateMillis(w.StartDateAt),
		EndDateAt:            DateMillis(w.EndDateAt),
		StartTimeAt:          w.StartTimeAt,
		EndTimeAt:            w.EndTimeAt,
		ExtraSkills:          skills,
		BlockOutsideActivity: w.BlockOutsideActivity,
		Agents:               agents,
		TotalAgents:          w.TotalAgents,
	}

	// ver starts at 1: 0 means a field mask left it out, and a token built from
	// it would fail every save with a false conflict.
	if w.Ver > 0 {
		out.Etag, _ = etag.EncodeEtag(etag.EtagWorkingSchedule, w.ID, w.Ver)
	}

	if !w.CreatedAt.Time.IsZero() {
		out.CreatedAt = w.CreatedAt.Time.UnixMilli()
	}

	if !w.UpdatedAt.Time.IsZero() {
		out.UpdatedAt = w.UpdatedAt.Time.UnixMilli()
	}

	return out
}
