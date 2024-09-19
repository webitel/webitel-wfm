package model

import pb "github.com/webitel/webitel-wfm/gen/go/api"

type ShiftTemplate struct {
	DomainRecord

	Name        string              `json:"name" db:"name"`
	Description *string             `json:"description" db:"description"`
	Times       []ShiftTemplateTime `json:"times" db:"times,json"`
}

func (s *ShiftTemplate) MarshalProto() *pb.ShiftTemplate {
	times := make([]*pb.ShiftTemplateTime, 0, len(s.Times))
	for _, t := range s.Times {
		times = append(times, t.MarshalProto())
	}

	out := &pb.ShiftTemplate{
		Id:          s.Id,
		DomainId:    s.DomainId,
		Name:        s.Name,
		Description: s.Description,
		Times:       times,
		CreatedBy:   s.CreatedBy.MarshalProto(),
		UpdatedBy:   s.UpdatedBy.MarshalProto(),
	}

	if !s.CreatedAt.Time.IsZero() {
		out.CreatedAt = s.CreatedAt.Time.UnixMilli()
	}

	if !s.UpdatedAt.Time.IsZero() {
		out.UpdatedAt = s.UpdatedAt.Time.UnixMilli()
	}

	return out
}

type ShiftTemplateTime struct {
	Start int32 `json:"start" db:"start_min"`
	End   int32 `json:"end" db:"end_min"`
}

func (s *ShiftTemplateTime) MarshalProto() *pb.ShiftTemplateTime {
	return &pb.ShiftTemplateTime{
		Start: s.Start,
		End:   s.End,
	}
}
