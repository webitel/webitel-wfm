package model

import pb "github.com/webitel/webitel-wfm/gen/go/api"

type PauseTemplate struct {
	DomainRecord

	Name        string  `json:"name" db:"name"`
	Description *string `json:"description" db:"description"`
}

func (p *PauseTemplate) MarshalProto() *pb.PauseTemplate {
	out := &pb.PauseTemplate{
		Id:          p.Id,
		DomainId:    p.DomainId,
		Name:        p.Name,
		Description: p.Description,
		CreatedBy:   p.CreatedBy.MarshalProto(),
		UpdatedBy:   p.UpdatedBy.MarshalProto(),
	}

	if !p.CreatedAt.Time.IsZero() {
		out.CreatedAt = p.CreatedAt.Time.UnixMilli()
	}

	if !p.UpdatedAt.Time.IsZero() {
		out.UpdatedAt = p.UpdatedAt.Time.UnixMilli()
	}

	return out
}

type PauseTemplateCause struct {
	DomainRecord

	Duration int64      `json:"duration" db:"duration"`
	Cause    LookupItem `json:"cause" db:"cause,json"`
}

func (p *PauseTemplateCause) MarshalProto() *pb.PauseTemplateCause {
	out := &pb.PauseTemplateCause{
		Id:        p.Id,
		DomainId:  p.DomainId,
		Duration:  p.Duration,
		Cause:     p.Cause.MarshalProto(),
		CreatedBy: p.CreatedBy.MarshalProto(),
		UpdatedBy: p.UpdatedBy.MarshalProto(),
	}

	if !p.CreatedAt.Time.IsZero() {
		out.CreatedAt = p.CreatedAt.Time.UnixMilli()
	}

	if !p.UpdatedAt.Time.IsZero() {
		out.UpdatedAt = p.UpdatedAt.Time.UnixMilli()
	}

	return out
}
