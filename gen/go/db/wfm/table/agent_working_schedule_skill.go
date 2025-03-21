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

var AgentWorkingScheduleSkill = newAgentWorkingScheduleSkillTable("wfm", "agent_working_schedule_skill", "")

type agentWorkingScheduleSkillTable struct {
	postgres.Table

	// Columns
	ID                     postgres.ColumnInteger
	DomainID               postgres.ColumnInteger
	AgentWorkingScheduleID postgres.ColumnInteger
	SkillID                postgres.ColumnInteger
	Capacity               postgres.ColumnInteger

	AllColumns     postgres.ColumnList
	MutableColumns postgres.ColumnList
	DefaultColumns postgres.ColumnList
}

type AgentWorkingScheduleSkillTable struct {
	agentWorkingScheduleSkillTable

	EXCLUDED agentWorkingScheduleSkillTable
}

// AS creates new AgentWorkingScheduleSkillTable with assigned alias
func (a AgentWorkingScheduleSkillTable) AS(alias string) *AgentWorkingScheduleSkillTable {
	return newAgentWorkingScheduleSkillTable(a.SchemaName(), a.TableName(), alias)
}

// Schema creates new AgentWorkingScheduleSkillTable with assigned schema name
func (a AgentWorkingScheduleSkillTable) FromSchema(schemaName string) *AgentWorkingScheduleSkillTable {
	return newAgentWorkingScheduleSkillTable(schemaName, a.TableName(), a.Alias())
}

// WithPrefix creates new AgentWorkingScheduleSkillTable with assigned table prefix
func (a AgentWorkingScheduleSkillTable) WithPrefix(prefix string) *AgentWorkingScheduleSkillTable {
	return newAgentWorkingScheduleSkillTable(a.SchemaName(), prefix+a.TableName(), a.TableName())
}

// WithSuffix creates new AgentWorkingScheduleSkillTable with assigned table suffix
func (a AgentWorkingScheduleSkillTable) WithSuffix(suffix string) *AgentWorkingScheduleSkillTable {
	return newAgentWorkingScheduleSkillTable(a.SchemaName(), a.TableName()+suffix, a.TableName())
}

func newAgentWorkingScheduleSkillTable(schemaName, tableName, alias string) *AgentWorkingScheduleSkillTable {
	return &AgentWorkingScheduleSkillTable{
		agentWorkingScheduleSkillTable: newAgentWorkingScheduleSkillTableImpl(schemaName, tableName, alias),
		EXCLUDED:                       newAgentWorkingScheduleSkillTableImpl("", "excluded", ""),
	}
}

func newAgentWorkingScheduleSkillTableImpl(schemaName, tableName, alias string) agentWorkingScheduleSkillTable {
	var (
		IDColumn                     = postgres.IntegerColumn("id")
		DomainIDColumn               = postgres.IntegerColumn("domain_id")
		AgentWorkingScheduleIDColumn = postgres.IntegerColumn("agent_working_schedule_id")
		SkillIDColumn                = postgres.IntegerColumn("skill_id")
		CapacityColumn               = postgres.IntegerColumn("capacity")
		allColumns                   = postgres.ColumnList{IDColumn, DomainIDColumn, AgentWorkingScheduleIDColumn, SkillIDColumn, CapacityColumn}
		mutableColumns               = postgres.ColumnList{DomainIDColumn, AgentWorkingScheduleIDColumn, SkillIDColumn, CapacityColumn}
		defaultColumns               = postgres.ColumnList{IDColumn}
	)

	return agentWorkingScheduleSkillTable{
		Table: postgres.NewTable(schemaName, tableName, alias, allColumns...),

		//Columns
		ID:                     IDColumn,
		DomainID:               DomainIDColumn,
		AgentWorkingScheduleID: AgentWorkingScheduleIDColumn,
		SkillID:                SkillIDColumn,
		Capacity:               CapacityColumn,

		AllColumns:     allColumns,
		MutableColumns: mutableColumns,
		DefaultColumns: defaultColumns,
	}
}
