//
// Code generated by go-jet DO NOT EDIT.
//
// WARNING: Changes to this file may cause incorrect behavior
// and will be lost if the code is regenerated
//

package model

import (
	"time"
)

type CcSkill struct {
	ID          int32 `sql:"primary_key"`
	Name        string
	DomainID    int64
	Description string
	CreatedAt   *time.Time
	CreatedBy   *int64
	UpdatedAt   *time.Time
	UpdatedBy   *int64
}
