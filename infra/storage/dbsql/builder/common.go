package builder

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

type Expression interface {
	String() string
}

type expression struct {
	Left, Op, Right string
}

func (e *expression) String() string {
	return fmt.Sprintf("%s %s %s", e.Left, e.Op, e.Right)
}

// Equal returns a SQL equality expression (e.g., "left = right").
func Equal(left, right string) *expression {
	return &expression{Left: left, Op: "=", Right: right}
}

type OrderDirection int

func (o OrderDirection) String() string {
	return []string{"ASC", "DESC"}[o]
}

const (
	OrderDirectionASC OrderDirection = iota
	OrderDirectionDESC
)

func Ident(left, right string) string {
	return fmt.Sprintf("%s.%s", left, right)
}

func Alias(left, right string) string {
	return fmt.Sprintf("%s AS %s", left, right)
}

func OrderBy(left string, direction OrderDirection) string {
	return fmt.Sprintf("%s %s", left, direction)
}

func Coalesce(cols ...string) string {
	return fmt.Sprintf("COALESCE(%s)", strings.Join(cols, ", "))
}

type JSONBuildObjectFields map[string]any

func (j *JSONBuildObjectFields) More(columns JSONBuildObjectFields) {
	for key, field := range columns {
		(*j)[key] = field
	}
}

func Lookup(table Table, cols ...string) JSONBuildObjectFields {
	m := make(map[string]any, len(cols))
	for _, v := range cols {
		m[v] = table.Ident(v)
	}

	return m
}

// UserLookup returns json_build_object as part of JSONBuildObject.
// The resulting SQL will be:
//
//	json_build_object('id', table.Alias().id, 'name', COALESCE(table.Alias().name, table.Alias().username))
func UserLookup(table Table) JSONBuildObjectFields {
	return JSONBuildObjectFields{
		"id":   table.Ident("id"),
		"name": Coalesce(table.Ident("name"), table.Ident("username")),
	}
}

// JSONBuildObject generates a SQL json_build_object expression.
func JSONBuildObject(fields JSONBuildObjectFields) string {
	keys := slices.Sorted(maps.Keys(fields))
	parts := make([]string, 0, len(fields))
	for _, key := range keys {
		switch v := fields[key].(type) {
		case map[string]any, JSONBuildObjectFields:
			parts = append(parts, fmt.Sprintf("'%s', %s", key, JSONBuildObject(v.(JSONBuildObjectFields))))
		case string:
			parts = append(parts, fmt.Sprintf("'%s', %s", key, v))
		}
	}

	return fmt.Sprintf("json_build_object(%s)", strings.Join(parts, ", "))
}

// ConvertArgs converts a slice of any type to a slice of any.
func ConvertArgs[T any](input []T) []any {
	result := make([]any, 0, len(input))
	for _, v := range input {
		result = append(result, v)
	}

	return result
}
