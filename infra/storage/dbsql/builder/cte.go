package builder

import "github.com/huandu/go-sqlbuilder"

type CTEQuery struct {
	flavor  sqlbuilder.Flavor
	queries []*sqlbuilder.CTEQueryBuilder
}

func NewCTEQuery(flavor sqlbuilder.Flavor, queries ...*sqlbuilder.CTEQueryBuilder) *CTEQuery {
	return &CTEQuery{
		flavor:  flavor,
		queries: queries,
	}
}

func (c *CTEQuery) With(table *sqlbuilder.CTEQueryBuilder) *CTEQuery {
	c.queries = append(c.queries, table)

	return c
}

func (c *CTEQuery) Builder() *sqlbuilder.CTEBuilder {
	return c.flavor.NewCTEBuilder().With(c.queries...)
}
