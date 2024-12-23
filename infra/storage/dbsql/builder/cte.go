package builder

import "github.com/huandu/go-sqlbuilder"

type CTEQuery struct {
	queries []*sqlbuilder.CTEQueryBuilder
}

func NewCTEQuery(queries ...*sqlbuilder.CTEQueryBuilder) *CTEQuery {
	return &CTEQuery{
		queries: queries,
	}
}

func (c *CTEQuery) With(table *sqlbuilder.CTEQueryBuilder) *CTEQuery {
	c.queries = append(c.queries, table)

	return c
}

func (c *CTEQuery) Builder() *sqlbuilder.CTEBuilder {
	sqlbuilder.DefaultFlavor.NewCTEQueryBuilder()
	return sqlbuilder.With(c.queries...)
}
