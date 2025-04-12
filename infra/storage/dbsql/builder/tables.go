package builder

var (
	UserTable = Table{name: "directory.wbt_user", alias: "wu"}
)

var (
	CalendarTable = Table{name: "flow.calendar", alias: "cal"}
)

var (
	PauseCauseTable = Table{name: "call_center.cc_pause_cause", alias: "cpc"}
	AgentTable      = Table{name: "call_center.cc_agent", alias: "ca"}
	TeamTable       = Table{name: "call_center.cc_team", alias: "ct"}
	SkillTable      = Table{name: "call_center.cc_skill", alias: "cs"}
)

var (
	PauseTemplateTable             = Table{name: "wfm.pause_template", alias: "pt"}
	PauseTemplateCauseTable        = Table{name: "wfm.pause_template_cause", alias: "ptc"}
	ShiftTemplateTable             = Table{name: "wfm.shift_template", alias: "st"}
	WorkingConditionTable          = Table{name: "wfm.working_condition", alias: "wc"}
	AgentWorkingConditionTable     = Table{name: "wfm.agent_working_condition", alias: "awc"}
	AgentAbsenceTable              = Table{name: "wfm.agent_absence", alias: "aa"}
	ForecastCalculationTable       = Table{name: "wfm.forecast_calculation", alias: "fc"}
	WorkingScheduleTable           = Table{name: "wfm.working_schedule", alias: "ws"}
	WorkingScheduleExtraSkillTable = Table{name: "wfm.working_schedule_extra_skill", alias: "wses"}
	WorkingScheduleAgentTable      = Table{name: "wfm.working_schedule_agent", alias: "wsa"}
)

type Table struct {
	name  string
	alias string
}

func (t *Table) String() string {
	return Alias(t.name, t.alias)
}

func (t *Table) Name() string {
	return t.name
}

func (t *Table) Alias() string {
	return t.alias
}

func (t *Table) Ident(column string) string {
	return Ident(t.alias, column)
}

func (t *Table) WithAlias(alias string) Table {
	t.alias = alias

	return *t
}
