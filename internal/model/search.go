package model

import (
	"fmt"
	"sort"
	"strings"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql/builder"
)

type SearchItem struct {
	Id int64 `json:"id" db:"id"`

	Page   int32   `json:"page,omitempty" db:"page"`
	Size   int32   `json:"size,omitempty" db:"size"`
	Search *string `json:"q,omitempty" db:"q"`

	Sort   *string  `json:"sort,omitempty" db:"sort"`
	Fields []string `json:"fields,omitempty" db:"fields"`
}

func (s *SearchItem) SortFields() {
	sort.Strings(s.Fields)
}

func (s *SearchItem) Limit() int32 {
	var limit int32

	limit = 10
	if s.Size >= 1 {
		limit = s.Size + 1
	}

	return limit + 1
}

func (s *SearchItem) Offset() int32 {
	var page int32

	page = 1
	if s.Page > 0 {
		page = s.Page
	}

	return s.Size * (page - 1)
}

func (s *SearchItem) OrderBy(table string) string {
	order := "updated_at DESC"
	if s.Sort != nil {
		o, field := orderBy(*s.Sort)
		order = fmt.Sprintf(`CASE WHEN NOT call_center.cc_is_lookup(%[1]s, %[2]s) THEN %[4]s END %[3]s, CASE WHEN call_center.cc_is_lookup(%[1]s, %[2]s) THEN CAST((CAST(%[2]s AS text)) AS json) ->> 'name' END %[3]s`,
			quoteLiteral(table), quoteLiteral(field), o, field)
	}

	return order
}

// Where formats WHERE clauses:
//   - if Id != 0, then set's `id = Id`
//   - if Search != nil, then set's `search = Search`
func (s *SearchItem) Where(searchField string) *builder.WhereClause {
	if searchField == "" {
		searchField = "name"
	}

	wb := builder.Where()
	if s.Id != 0 {
		wb.AddWhereExpr(wb.Args, wb.Equal("id", s.Id))
	}

	if s.Search != nil {
		search := strings.Replace(*s.Search, "*", "%", -1)
		wb.AddWhereExpr(wb.Args, wb.ILike(searchField, search+"%"))
	}

	return &wb
}

func ListResult[C any](s int32, items []C) (bool, []C) {
	if int32(len(items)) == s {
		return true, items[0 : len(items)-1]
	}

	return false, items
}

func orderBy(s string) (sort string, field string) {
	if s[0] == '+' || s[0] == 32 {
		return "ASC", s[1:]
	}

	if s[0] == '-' {
		return "DESC", s[1:]
	}

	return "", s
}

// quoteLiteral quotes a 'literal' (e.g., a parameter, often used to pass literal
// to DDL and other statements that do not accept parameters) to be used as part
// of an SQL statement.
// For example,
//
//	exp_date := pq.QuoteLiteral("2023-01-05 15:00:00Z")
//	err := db.Exec(fmt.Sprintf("CREATE ROLE my_user VALID UNTIL %s", exp_date))
//
// Any single quotes in name will be escaped.
// Any backslashes (i.e. "\") will be
// replaced by two backslashes (i.e. "\\") and the C-style escape identifier
// that PostgreSQL provides ('E') will be prepended to the string.
func quoteLiteral(literal string) string {
	// This follows the PostgreSQL internal algorithm for handling quoted literals
	// from libpq, which can be found in the "PQEscapeStringInternal" function,
	// which is found in the libpq/fe-exec.c source file:
	// https://git.postgresql.org/gitweb/?p=postgresql.git;a=blob;f=src/interfaces/libpq/fe-exec.c
	//
	// substitute any single-quotes (') with two single-quotes ('')
	literal = strings.Replace(literal, `'`, `''`, -1)

	// determine if the string has any backslashes (\) in it.
	// if it does, replace any backslashes (\) with two backslashes (\\)
	// then, we need to wrap the entire string with a PostgreSQL
	// C-style escape. Per how "PQEscapeStringInternal" handles this case, we
	// also add a space before the "E"
	if strings.Contains(literal, `\`) {
		literal = strings.Replace(literal, `\`, `\\`, -1)
		literal = ` E'` + literal + `'`
	} else {
		// otherwise, we can just wrap the literal with a pair of single quotes
		literal = `'` + literal + `'`
	}

	return literal
}
