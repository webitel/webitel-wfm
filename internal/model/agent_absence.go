package model

import (
	"github.com/jackc/pgx/v5/pgtype"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/builder"
)

type Absence struct {
	DomainRecord

	AbsentAt      pgtype.Date `json:"absent_at" db:"absent_at,json"`
	AbsenceTypeId int64       `json:"absence_type_id" db:"absence_type_id"`
}

func (a *Absence) MarshalProto() *pb.Absence {
	return &pb.Absence{
		Id:        a.Id,
		DomainId:  a.DomainId,
		TypeId:    pb.AgentAbsenceType(a.AbsenceTypeId),
		AbsentAt:  a.AbsentAt.Time.UnixMilli(),
		CreatedAt: a.CreatedAt.Time.UnixMilli(),
		CreatedBy: a.CreatedBy.MarshalProto(),
		UpdatedAt: a.UpdatedAt.Time.UnixMilli(),
		UpdatedBy: a.UpdatedBy.MarshalProto(),
	}
}

type AgentAbsence struct {
	Agent   LookupItem `json:"agent" db:"agent,json"`
	Absence Absence    `json:"absence" db:"absence"`
}

func (a *AgentAbsence) MarshalProto() *pb.AgentAbsence {
	return &pb.AgentAbsence{
		Agent:   a.Agent.MarshalProto(),
		Absence: a.Absence.MarshalProto(),
	}
}

type AgentAbsences struct {
	Agent   LookupItem `json:"agent" db:"agent,json"`
	Absence []Absence  `json:"absence" db:"absence,json"`
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

type AgentAbsenceBulk struct {
	AbsenceTypeId int64 `json:"absence_type_id" db:"absence_type_id"`
	AbsentAtFrom  int64
	AbsentAtTo    int64
}

type AgentAbsenceSearch struct {
	SearchItem SearchItem

	AbsentAtFrom pgtype.Timestamp
	AbsentAtTo   pgtype.Timestamp

	Ids           []int64
	AgentIds      []int64
	SupervisorIds []int64
	TeamIds       []int64
	SkillIds      []int64
}

func (a *AgentAbsenceSearch) Where(search string) *builder.WhereClause {
	wb := a.SearchItem.Where(search)
	if a.AbsentAtTo.Valid {
		wb.AddWhereExpr(wb.Args, wb.LessEqualThan("absent_at", a.AbsentAtTo))
	}

	if a.AbsentAtFrom.Valid {
		wb.AddWhereExpr(wb.Args, wb.GreaterEqualThan("absent_at", a.AbsentAtFrom))
	}

	if len(a.AgentIds) > 0 {
		wb.AddWhereExpr(wb.Args, wb.Any("(agent ->> 'id')::bigint", "=", a.AgentIds))
	}

	if len(a.Ids) > 0 {
		wb.AddWhereExpr(wb.Args, wb.Any("id", "=", a.Ids))
	}

	return wb
}
