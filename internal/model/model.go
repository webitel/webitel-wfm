package model

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"

	pb "github.com/webitel/webitel-wfm/gen/go/api"
)

type DomainRecord struct {
	Id        int64            `json:"id" db:"id"`
	DomainId  int64            `json:"domain_id" db:"domain_id"`
	CreatedAt pgtype.Timestamp `json:"created_at" db:"created_at,json"`
	CreatedBy LookupItem       `json:"created_by" db:"created_by,json"`
	UpdatedAt pgtype.Timestamp `json:"updated_at" db:"updated_at,json"`
	UpdatedBy LookupItem       `json:"updated_by" db:"updated_by,json"`
}

type LookupItem struct {
	Id   int64   `json:"id" db:"id"`
	Name *string `json:"name" db:"name"`
}

func (l *LookupItem) Value() (driver.Value, error) {
	return json.Marshal(l)
}

func (l *LookupItem) IsZero() bool {
	return l == &LookupItem{}
}

func (l *LookupItem) MarshalProto() *pb.LookupEntity {
	if l == nil {
		return nil
	}

	return &pb.LookupEntity{
		Id:   l.Id,
		Name: l.Name,
	}
}

type Column struct {
	Name   string
	Type   string
	Values []any
}
