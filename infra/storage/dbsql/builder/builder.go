package builder

import (
	"slices"

	"github.com/huandu/go-sqlbuilder"
)

type Builder struct {
	flavor sqlbuilder.Flavor
}

func NewBuilder(flavor sqlbuilder.Flavor) *Builder {
	return &Builder{flavor: flavor}
}

func (b *Builder) Format(format string, args ...any) sqlbuilder.Builder {
	return sqlbuilder.Build(format, args...)
}

func (b *Builder) Select(cols ...string) *sqlbuilder.SelectBuilder {
	return b.flavor.NewSelectBuilder().Select(cols...)
}

func (b *Builder) Insert(table string, args map[string]any) *sqlbuilder.InsertBuilder {
	ib := b.flavor.NewInsertBuilder().InsertInto(table)
	if len(args) > 0 {
		keys := make([]string, 0, len(args))
		for k := range args {
			keys = append(keys, k)
		}

		slices.Sort(keys)
		ks := make([]string, 0, len(args))
		vs := make([]any, 0, len(args))
		for _, k := range keys {
			ks = append(ks, k)
			vs = append(vs, args[k])
		}

		ib.Cols(ks...).Values(vs...)
	}

	return ib
}

func (b *Builder) Update(table string) *sqlbuilder.UpdateBuilder {
	return b.flavor.NewUpdateBuilder().Update(table)
}

func (b *Builder) Delete(table string) *sqlbuilder.DeleteBuilder {
	return b.flavor.NewDeleteBuilder().DeleteFrom(table)
}

func (b *Builder) CTE(tables ...*sqlbuilder.CTETableBuilder) *sqlbuilder.CTEBuilder {
	return b.flavor.NewCTEBuilder().With(tables...)
}

func (b *Builder) With(table string) *sqlbuilder.CTETableBuilder {
	return b.flavor.NewCTETableBuilder().Table(table)
}

func (b *Builder) Values(value ...any) *sqlbuilder.InsertBuilder {
	vb := b.flavor.NewInsertBuilder()

	if len(value) > 0 {
		vb.Values(value...)
	}
	return vb
}

func (b *Builder) Struct(value any) *sqlbuilder.Struct {
	return sqlbuilder.NewStruct(value)
}

func (b *Builder) RBAC(use bool, acl string, id int64, domain int64, groups []int, access uint32) *sqlbuilder.WhereClause {
	sb := b.flavor.NewSelectBuilder()
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
