package model

import (
	"github.com/jackc/pgx/v5/pgtype"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
)

type Holiday struct {
	Date pgtype.Date
	Name string
}

func (h *Holiday) MarshalProto() *pb.Holiday {
	return &pb.Holiday{
		Date: h.Date.Time.Unix(),
		Name: h.Name,
	}
}
