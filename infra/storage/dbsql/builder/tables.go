package builder

var (
	UserTable = Table{name: "directory.wbt_user", alias: "wu"}
)

var (
	PauseCauseTable = Table{name: "call_center.cc_pause_cause", alias: "cpc"}
)

var (
	PauseTemplateTable      = Table{name: "wfm.pause_template", alias: "pt"}
	PauseTemplateCauseTable = Table{name: "wfm.pause_template_cause", alias: "ptc"}
	ShiftTemplateTable      = Table{name: "wfm.shift_template", alias: "st"}
	WorkingConditionTable   = Table{name: "wfm.working_condition", alias: "wc"}
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
