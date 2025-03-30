package model

import (
	"github.com/jackc/pgx/v5/pgtype"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
)

type AgentAbsenceType int32

const (
	AgentAbsenceTypeUnspecified AgentAbsenceType = iota
	AgentAbsenceTypeDayOff
	AgentAbsenceTypeVacation
	AgentAbsenceTypeSickDay
)

func (s AgentAbsenceType) String() string {
	return []string{"unspecified", "dayoff", "vacation", "sickday"}[s]
}

type Absence struct {
	DomainRecord

	AbsentAt    pgtype.Date      `json:"absent_at" db:"absent_at,json"`
	AbsenceType AgentAbsenceType `json:"absence_type_id" db:"absence_type_id"`
}

func (a *Absence) MarshalProto() *pb.Absence {
	return &pb.Absence{
		Id:        a.Id,
		DomainId:  a.DomainId,
		TypeId:    pb.AbsenceType(a.AbsenceType),
		AbsentAt:  a.AbsentAt.Time.UnixMilli(),
		CreatedAt: a.CreatedAt.Time.UnixMilli(),
		CreatedBy: a.CreatedBy.MarshalProto(),
		UpdatedAt: a.UpdatedAt.Time.UnixMilli(),
		UpdatedBy: a.UpdatedBy.MarshalProto(),
	}
}

type AgentAbsences struct {
	Agent   LookupItem `json:"agent" db:"agent,json"`
	Absence []*Absence `json:"absence" db:"absence,json"`
}

func (a *AgentAbsences) MarshalProto() *pb.AgentAbsences {
	absences := make([]*pb.Absence, 0, len(a.Absence))
	for _, absence := range a.Absence {
		absences = append(absences, absence.MarshalProto())
	}

	return &pb.AgentAbsences{
		Agent:    a.Agent.MarshalProto(),
		Absences: absences,
	}
}
