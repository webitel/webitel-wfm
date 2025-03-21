//
// Code generated by go-jet DO NOT EDIT.
//
// WARNING: Changes to this file may cause incorrect behavior
// and will be lost if the code is regenerated
//

package table

import (
	"github.com/go-jet/jet/v2/postgres"
)

var Calendar = newCalendarTable("flow", "calendar", "")

type calendarTable struct {
	postgres.Table

	// Columns
	ID          postgres.ColumnInteger
	StartAt     postgres.ColumnInteger
	EndAt       postgres.ColumnInteger
	Name        postgres.ColumnString
	DomainID    postgres.ColumnInteger
	Description postgres.ColumnString
	TimezoneID  postgres.ColumnInteger
	CreatedAt   postgres.ColumnInteger
	CreatedBy   postgres.ColumnInteger
	UpdatedAt   postgres.ColumnInteger
	UpdatedBy   postgres.ColumnInteger
	Excepts     postgres.ColumnString
	Accepts     postgres.ColumnString

	AllColumns     postgres.ColumnList
	MutableColumns postgres.ColumnList
	DefaultColumns postgres.ColumnList
}

type CalendarTable struct {
	calendarTable

	EXCLUDED calendarTable
}

// AS creates new CalendarTable with assigned alias
func (a CalendarTable) AS(alias string) *CalendarTable {
	return newCalendarTable(a.SchemaName(), a.TableName(), alias)
}

// Schema creates new CalendarTable with assigned schema name
func (a CalendarTable) FromSchema(schemaName string) *CalendarTable {
	return newCalendarTable(schemaName, a.TableName(), a.Alias())
}

// WithPrefix creates new CalendarTable with assigned table prefix
func (a CalendarTable) WithPrefix(prefix string) *CalendarTable {
	return newCalendarTable(a.SchemaName(), prefix+a.TableName(), a.TableName())
}

// WithSuffix creates new CalendarTable with assigned table suffix
func (a CalendarTable) WithSuffix(suffix string) *CalendarTable {
	return newCalendarTable(a.SchemaName(), a.TableName()+suffix, a.TableName())
}

func newCalendarTable(schemaName, tableName, alias string) *CalendarTable {
	return &CalendarTable{
		calendarTable: newCalendarTableImpl(schemaName, tableName, alias),
		EXCLUDED:      newCalendarTableImpl("", "excluded", ""),
	}
}

func newCalendarTableImpl(schemaName, tableName, alias string) calendarTable {
	var (
		IDColumn          = postgres.IntegerColumn("id")
		StartAtColumn     = postgres.IntegerColumn("start_at")
		EndAtColumn       = postgres.IntegerColumn("end_at")
		NameColumn        = postgres.StringColumn("name")
		DomainIDColumn    = postgres.IntegerColumn("domain_id")
		DescriptionColumn = postgres.StringColumn("description")
		TimezoneIDColumn  = postgres.IntegerColumn("timezone_id")
		CreatedAtColumn   = postgres.IntegerColumn("created_at")
		CreatedByColumn   = postgres.IntegerColumn("created_by")
		UpdatedAtColumn   = postgres.IntegerColumn("updated_at")
		UpdatedByColumn   = postgres.IntegerColumn("updated_by")
		ExceptsColumn     = postgres.StringColumn("excepts")
		AcceptsColumn     = postgres.StringColumn("accepts")
		allColumns        = postgres.ColumnList{IDColumn, StartAtColumn, EndAtColumn, NameColumn, DomainIDColumn, DescriptionColumn, TimezoneIDColumn, CreatedAtColumn, CreatedByColumn, UpdatedAtColumn, UpdatedByColumn, ExceptsColumn, AcceptsColumn}
		mutableColumns    = postgres.ColumnList{StartAtColumn, EndAtColumn, NameColumn, DomainIDColumn, DescriptionColumn, TimezoneIDColumn, CreatedAtColumn, CreatedByColumn, UpdatedAtColumn, UpdatedByColumn, ExceptsColumn, AcceptsColumn}
		defaultColumns    = postgres.ColumnList{IDColumn}
	)

	return calendarTable{
		Table: postgres.NewTable(schemaName, tableName, alias, allColumns...),

		//Columns
		ID:          IDColumn,
		StartAt:     StartAtColumn,
		EndAt:       EndAtColumn,
		Name:        NameColumn,
		DomainID:    DomainIDColumn,
		Description: DescriptionColumn,
		TimezoneID:  TimezoneIDColumn,
		CreatedAt:   CreatedAtColumn,
		CreatedBy:   CreatedByColumn,
		UpdatedAt:   UpdatedAtColumn,
		UpdatedBy:   UpdatedByColumn,
		Excepts:     ExceptsColumn,
		Accepts:     AcceptsColumn,

		AllColumns:     allColumns,
		MutableColumns: mutableColumns,
		DefaultColumns: defaultColumns,
	}
}
