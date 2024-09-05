package model

import (
	"fmt"

	pb "github.com/webitel/webitel-wfm/gen/go/api"
)

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

type ForecastCalculationResult struct {
	Name   string
	Type   string
	Values []any
}

func (f *ForecastCalculationResult) MarshalProto() *pb.ExecuteForecastCalculationResponse_Field {
	val := make([]string, 0, len(f.Values))
	for _, v := range f.Values {
		val = append(val, fmt.Sprintf("%v", v))
	}

	return &pb.ExecuteForecastCalculationResponse_Field{
		Name:   f.Name,
		Type:   f.Type,
		Values: val,
	}
}
