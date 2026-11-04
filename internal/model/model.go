package model

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
)

type DomainRecord struct {
	ID        int64            `json:"id" db:"id"`
	DomainID  int64            `json:"domain_id" db:"domain_id"`
	CreatedAt pgtype.Timestamp `json:"created_at" db:"created_at,json"`
	CreatedBy LookupItem       `json:"created_by" db:"created_by,json"`
	UpdatedAt pgtype.Timestamp `json:"updated_at" db:"updated_at,json"`
	UpdatedBy LookupItem       `json:"updated_by" db:"updated_by,json"`
}

type LookupItem struct {
	ID   int64   `json:"id" db:"id"`
	Name *string `json:"name" db:"name"`
}

func (l *LookupItem) SafeID() *int64 {
	if l == nil || l.ID == 0 {
		return nil
	}

	return &l.ID
}

func (l *LookupItem) Value() (driver.Value, error) {
	return json.Marshal(l)
}

func (l *LookupItem) Scan(src any) error {
	switch v := src.(type) {
	case []byte:
		if err := json.Unmarshal(v, l); err != nil {
			return err
		}

		if l.ID == 0 {
			name := "[deleted]"
			l.Name = &name
		}

		return nil
	case string:
		if err := json.Unmarshal([]byte(v), l); err != nil {
			return err
		}

		if l.ID == 0 {
			name := "[deleted]"
			l.Name = &name
		}

		return nil
	}

	return nil
}

func (l *LookupItem) IsZero() bool {
	return l == &LookupItem{}
}

func (l *LookupItem) MarshalProto() *pb.LookupEntity {
	if l == nil {
		return nil
	}

	return &pb.LookupEntity{
		Id:   l.ID,
		Name: l.Name,
	}
}

type Column struct {
	Name   string
	Type   string
	Values []any
}
