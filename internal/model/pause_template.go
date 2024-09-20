package model

import pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"

type PauseTemplate struct {
	DomainRecord

	Name        string               `json:"name" db:"name"`
	Description *string              `json:"description" db:"description"`
	Causes      []PauseTemplateCause `json:"causes" db:"causes,json"`
}

func (p *PauseTemplate) MarshalProto() *pb.PauseTemplate {
	causes := make([]*pb.PauseTemplateCause, 0, len(p.Causes))
	for _, c := range p.Causes {
		causes = append(causes, c.MarshalProto())
	}

	out := &pb.PauseTemplate{
		Id:          p.Id,
		DomainId:    p.DomainId,
		Name:        p.Name,
		Description: p.Description,
		Causes:      causes,
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
	Duration int64       `json:"duration" db:"duration"`
	Cause    *LookupItem `json:"cause" db:"cause,json"`
}

func (p *PauseTemplateCause) MarshalProto() *pb.PauseTemplateCause {
	return &pb.PauseTemplateCause{
		Duration: p.Duration,
		Cause:    p.Cause.MarshalProto(),
	}
}
