package model

import pb "github.com/webitel/webitel-wfm/gen/go/api"

type ShiftTemplate struct {
	DomainRecord

	Name        string  `json:"name" db:"name"`
	Description *string `json:"description" db:"description"`
}

func (s *ShiftTemplate) MarshalProto() *pb.ShiftTemplate {
	out := &pb.ShiftTemplate{
		Id:          s.Id,
		DomainId:    s.DomainId,
		Name:        s.Name,
		Description: s.Description,
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
	DomainRecord

	Start int32 `json:"start" db:"start_min"`
	End   int32 `json:"end" db:"end_min"`
}

func (s *ShiftTemplateTime) MarshalProto() *pb.ShiftTemplateTime {
	out := &pb.ShiftTemplateTime{
		Id:        s.Id,
		DomainId:  s.DomainId,
		Start:     s.Start,
		End:       s.End,
		CreatedBy: s.CreatedBy.MarshalProto(),
		UpdatedBy: s.UpdatedBy.MarshalProto(),
	}

	if !s.CreatedAt.Time.IsZero() {
		out.CreatedAt = s.CreatedAt.Time.UnixMilli()
	}

	if !s.UpdatedAt.Time.IsZero() {
		out.UpdatedAt = s.UpdatedAt.Time.UnixMilli()
	}

	return out
}
