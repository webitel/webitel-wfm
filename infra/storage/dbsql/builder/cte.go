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

func (c *CTEQuery) Tables() []string {
	tables := make([]string, 0, len(c.queries))
	for _, q := range c.queries {
		tables = append(tables, q.TableName())
	}

	return tables
}

func (c *CTEQuery) Builder() *sqlbuilder.CTEBuilder {
	return sqlbuilder.With(c.queries...)
}
