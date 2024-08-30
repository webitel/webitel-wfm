package model

import pb "github.com/webitel/webitel-wfm/gen/go/api"

type ForecastCalculation struct {
	DomainRecord

	Name        string  `json:"name" db:"name"`
	Description *string `json:"description" db:"description"`
	Query       string  `json:"query" db:"query"`
}

func (p *ForecastCalculation) MarshalProto() *pb.ForecastCalculation {
	out := &pb.ForecastCalculation{
		Id:          p.Id,
		DomainId:    p.DomainId,
		Name:        p.Name,
		Description: p.Description,
		Query:       p.Query,
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
