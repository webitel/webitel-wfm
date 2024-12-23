package builder

import (
	"slices"

	"github.com/huandu/go-sqlbuilder"
)

func init() {
	sqlbuilder.DefaultFlavor = sqlbuilder.PostgreSQL
}

func Format(format string, args ...any) sqlbuilder.Builder {
	return sqlbuilder.Build(format, args...)
}

func Select(cols ...string) *sqlbuilder.SelectBuilder {
	return sqlbuilder.NewSelectBuilder().Select(cols...)
}

func Insert(table string, args []map[string]any) *sqlbuilder.InsertBuilder {
	ib := sqlbuilder.NewInsertBuilder().InsertInto(table)
	for _, arg := range args {
		if len(arg) > 0 {
			keys := make([]string, 0, len(args))
			for k := range arg {
				keys = append(keys, k)
			}

			slices.Sort(keys)

			ks := make([]string, 0, len(arg))
			vs := make([]any, 0, len(arg))
			for _, k := range keys {
				ks = append(ks, k)
				vs = append(vs, arg[k])
			}

			ib.Cols(ks...).Values(vs...)
		}
	}

	return ib
}

// Update creates UpdateBuilder using a specified table and list of arguments.
// Args representing as SET "field = value" in UPDATE.
//
//	Update("test", map[string]any{"foo": "bar"})
//	// Resulting to: UPDATE test SET foo = "bar"
func Update(table string, args map[string]any) *sqlbuilder.UpdateBuilder {
	ub := sqlbuilder.NewUpdateBuilder().Update(table)
	if len(args) > 0 {
		keys := make([]string, 0, len(args))
		for k := range args {
			keys = append(keys, k)
		}

		slices.Sort(keys)
		vs := make([]string, 0, len(args))
		for _, k := range keys {
			vs = append(vs, ub.Assign(k, args[k]))
		}

		ub.Set(vs...)
	}

	return ub
}

func Delete(table string) *sqlbuilder.DeleteBuilder {
	return sqlbuilder.NewDeleteBuilder().DeleteFrom(table)
}

func CTE(tables ...*sqlbuilder.CTEQueryBuilder) *CTEQuery {
	return NewCTEQuery(tables...)
}

func With(table string) *sqlbuilder.CTEQueryBuilder {
	return sqlbuilder.DefaultFlavor.NewCTEQueryBuilder().Table(table)
}

func Values(value ...any) *sqlbuilder.InsertBuilder {
	vb := sqlbuilder.NewInsertBuilder()
	if len(value) > 0 {
		vb.Values(value...)
	}

	return vb
}

func RBAC(use bool, acl string, id int64, domain int64, groups []int, access uint32) *sqlbuilder.WhereClause {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select(sb.As(sb.Var("1"), "rbac")).
		From(acl).
		Where(sb.Equal("dc", domain),
			sb.Any("subject", "=", groups),
			"access & "+sb.Var(access)+" = "+sb.Var(access))

	if id != 0 {
		sb.Where(sb.Equal("object", id))
	} else {
		sb.Where("object = id")
	}

	wb := sqlbuilder.NewWhereClause()
	cond := sqlbuilder.NewCond()
	wb.AddWhereExpr(cond.Args, cond.Or(
		cond.Var(use)+" IS FALSE",
		cond.Exists(sb),
	))

	return wb
}

type WhereClause struct {
	sqlbuilder.WhereClause
	sqlbuilder.Cond
}

func Where() WhereClause {
	return WhereClause{
		WhereClause: sqlbuilder.WhereClause{},
		Cond: sqlbuilder.Cond{
			Args: &sqlbuilder.Args{},
		},
	}
}
