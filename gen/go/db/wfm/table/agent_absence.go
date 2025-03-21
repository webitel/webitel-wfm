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

var AgentAbsence = newAgentAbsenceTable("wfm", "agent_absence", "")

type agentAbsenceTable struct {
	postgres.Table

	// Columns
	ID            postgres.ColumnInteger
	DomainID      postgres.ColumnInteger
	CreatedAt     postgres.ColumnTimestampz
	CreatedBy     postgres.ColumnInteger
	UpdatedAt     postgres.ColumnTimestampz
	UpdatedBy     postgres.ColumnInteger
	AbsentAt      postgres.ColumnDate
	AgentID       postgres.ColumnInteger
	AbsenceTypeID postgres.ColumnInteger

	AllColumns     postgres.ColumnList
	MutableColumns postgres.ColumnList
	DefaultColumns postgres.ColumnList
}

type AgentAbsenceTable struct {
	agentAbsenceTable

	EXCLUDED agentAbsenceTable
}

// AS creates new AgentAbsenceTable with assigned alias
func (a AgentAbsenceTable) AS(alias string) *AgentAbsenceTable {
	return newAgentAbsenceTable(a.SchemaName(), a.TableName(), alias)
}

// Schema creates new AgentAbsenceTable with assigned schema name
func (a AgentAbsenceTable) FromSchema(schemaName string) *AgentAbsenceTable {
	return newAgentAbsenceTable(schemaName, a.TableName(), a.Alias())
}

// WithPrefix creates new AgentAbsenceTable with assigned table prefix
func (a AgentAbsenceTable) WithPrefix(prefix string) *AgentAbsenceTable {
	return newAgentAbsenceTable(a.SchemaName(), prefix+a.TableName(), a.TableName())
}

// WithSuffix creates new AgentAbsenceTable with assigned table suffix
func (a AgentAbsenceTable) WithSuffix(suffix string) *AgentAbsenceTable {
	return newAgentAbsenceTable(a.SchemaName(), a.TableName()+suffix, a.TableName())
}

func newAgentAbsenceTable(schemaName, tableName, alias string) *AgentAbsenceTable {
	return &AgentAbsenceTable{
		agentAbsenceTable: newAgentAbsenceTableImpl(schemaName, tableName, alias),
		EXCLUDED:          newAgentAbsenceTableImpl("", "excluded", ""),
	}
}

func newAgentAbsenceTableImpl(schemaName, tableName, alias string) agentAbsenceTable {
	var (
		IDColumn            = postgres.IntegerColumn("id")
		DomainIDColumn      = postgres.IntegerColumn("domain_id")
		CreatedAtColumn     = postgres.TimestampzColumn("created_at")
		CreatedByColumn     = postgres.IntegerColumn("created_by")
		UpdatedAtColumn     = postgres.TimestampzColumn("updated_at")
		UpdatedByColumn     = postgres.IntegerColumn("updated_by")
		AbsentAtColumn      = postgres.DateColumn("absent_at")
		AgentIDColumn       = postgres.IntegerColumn("agent_id")
		AbsenceTypeIDColumn = postgres.IntegerColumn("absence_type_id")
		allColumns          = postgres.ColumnList{IDColumn, DomainIDColumn, CreatedAtColumn, CreatedByColumn, UpdatedAtColumn, UpdatedByColumn, AbsentAtColumn, AgentIDColumn, AbsenceTypeIDColumn}
		mutableColumns      = postgres.ColumnList{DomainIDColumn, CreatedAtColumn, CreatedByColumn, UpdatedAtColumn, UpdatedByColumn, AbsentAtColumn, AgentIDColumn, AbsenceTypeIDColumn}
		defaultColumns      = postgres.ColumnList{IDColumn, CreatedAtColumn, UpdatedAtColumn, AbsentAtColumn}
	)

	return agentAbsenceTable{
		Table: postgres.NewTable(schemaName, tableName, alias, allColumns...),

		//Columns
		ID:            IDColumn,
		DomainID:      DomainIDColumn,
		CreatedAt:     CreatedAtColumn,
		CreatedBy:     CreatedByColumn,
		UpdatedAt:     UpdatedAtColumn,
		UpdatedBy:     UpdatedByColumn,
		AbsentAt:      AbsentAtColumn,
		AgentID:       AgentIDColumn,
		AbsenceTypeID: AbsenceTypeIDColumn,

		AllColumns:     allColumns,
		MutableColumns: mutableColumns,
		DefaultColumns: defaultColumns,
	}
}
