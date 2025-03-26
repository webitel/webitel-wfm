package builder

import (
	"fmt"
	"strings"
)

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

func JSONBuildObject(table Table, column ...string) string {
	jsonFields := make([]string, 0, len(column))
	for _, field := range column {
		jsonFields = append(jsonFields, fmt.Sprintf("'%s', %s.%s", field, table.Alias(), field))
	}

	return fmt.Sprintf("json_build_object(%s)", strings.Join(jsonFields, ", "))
}

// ConvertArgs converts a slice of any type to a slice of any
func ConvertArgs[T any](input []T) []any {
	result := make([]any, 0, len(input))
	for _, v := range input {
		result = append(result, v)
	}

	return result
}
