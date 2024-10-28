package model

import (
	"github.com/jackc/pgx/v5/pgtype"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
)

type ForecastCalculation struct {
	DomainRecord

	Name        string   `json:"name" db:"name"`
	Description *string  `json:"description" db:"description"`
	Procedure   string   `json:"procedure" db:"procedure"`
	Args        []string `json:"args" db:"args"`
}

func (p *ForecastCalculation) MarshalProto() *pb.ForecastCalculation {
	out := &pb.ForecastCalculation{
		Id:          p.Id,
		DomainId:    p.DomainId,
		Name:        p.Name,
		Description: p.Description,
		Procedure:   p.Procedure,
		Args:        p.Args,
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
	Timestamp pgtype.Timestamp `db:"forecast_at"`
	Agents    *int64           `db:"agents"`
}

func (f *ForecastCalculationResult) MarshalProto() *pb.ExecuteForecastCalculationResponse_Forecast {
	return &pb.ExecuteForecastCalculationResponse_Forecast{
		Timestamp: f.Timestamp.Time.UnixMilli(),
		Agents:    *f.Agents,
	}
}
