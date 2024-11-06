package model

import pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"

type WorkingSchedule struct {
	DomainRecord

	Name  string `db:"name"`
	State int32  `db:"state"`

	Team                 LookupItem    `db:"team"`
	Calendar             LookupItem    `db:"calendar"`
	StartDateAt          int64         `db:"start_date_at"`
	EndDateAt            int64         `db:"end_date_at"`
	StartTimeAt          int64         `db:"start_time_at"`
	EndTimeAt            int64         `db:"end_time_at"`
	ExtraSkills          []*LookupItem `db:"extra_skills"`
	BlockOutsideActivity bool          `db:"block_outside_activity"`
	Agents               []*LookupItem `db:"agents"`
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
		Id:                   w.Id,
		DomainId:             w.DomainId,
		CreatedBy:            w.CreatedBy.MarshalProto(),
		UpdatedBy:            w.UpdatedBy.MarshalProto(),
		Name:                 w.Name,
		State:                pb.WorkingScheduleState(w.State),
		Team:                 w.Team.MarshalProto(),
		Calendar:             w.Calendar.MarshalProto(),
		StartDateAt:          w.StartDateAt,
		EndDateAt:            w.EndDateAt,
		StartTimeAt:          w.StartTimeAt,
		EndTimeAt:            w.EndTimeAt,
		ExtraSkills:          skills,
		BlockOutsideActivity: w.BlockOutsideActivity,
		Agents:               agents,
	}

	if !w.CreatedAt.Time.IsZero() {
		out.CreatedAt = w.CreatedAt.Time.UnixMilli()
	}

	if !w.UpdatedAt.Time.IsZero() {
		out.UpdatedAt = w.UpdatedAt.Time.UnixMilli()
	}

	return out
}
