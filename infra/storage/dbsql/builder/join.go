package builder

import (
	"strings"

	"github.com/huandu/go-sqlbuilder"
)

func LeftJoin(table Table, expr ...Expression) (sqlbuilder.JoinOption, string, string) {
	expressions := make([]string, 0, len(expr))
	for _, e := range expr {
		expressions = append(expressions, e.String())
	}

	return sqlbuilder.LeftJoin, table.String(), strings.Join(expressions, " AND ")
}

type JoinRegistry struct {
	tables map[Table]struct{}
}

func NewJoinRegistry(tables ...Table) *JoinRegistry {
	r := &JoinRegistry{
		tables: make(map[Table]struct{}),
	}

	for _, table := range tables {
		r.Register(table)
	}

	return r
}

func (j *JoinRegistry) Register(table Table) {
	j.tables[table] = struct{}{}
}

func (j *JoinRegistry) Has(table Table) bool {
	_, ok := j.tables[table]

	return ok
}
