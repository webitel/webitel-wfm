package model

import (
	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
)

type ForecastCalculation struct {
	DomainRecord

	Name        string  `json:"name" db:"name"`
	Description *string `json:"description" db:"description"`
	Procedure   string  `json:"procedure" db:"procedure"`
}

func (p *ForecastCalculation) MarshalProto() *pb.ForecastCalculation {
	out := &pb.ForecastCalculation{
		Id:          p.Id,
		DomainId:    p.DomainId,
		Name:        p.Name,
		Description: p.Description,
		Procedure:   p.Procedure,
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
	Timestamp int64
	Agents    int64
}

func (f *ForecastCalculationResult) MarshalProto() *pb.ExecuteForecastCalculationResponse_Forecast {
	return &pb.ExecuteForecastCalculationResponse_Forecast{
		Timestamp: f.Timestamp,
		Agents:    f.Agents,
	}
}

type ForecastCalculationExecution struct {
	ForecastFrom int64
	ForecastTo   int64
}
