package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONBuildObject(t *testing.T) {
	tests := map[string]struct {
		input  JSONBuildObjectFields
		expect string
	}{
		"simple fields": {
			input:  JSONBuildObjectFields{"id": UserTable.Ident("id"), "name": UserTable.Ident("name")},
			expect: "json_build_object('id', wu.id, 'name', wu.name)",
		},
		"nested json": {
			input: JSONBuildObjectFields{
				"id":   AgentTable.Ident("id"),
				"meta": JSONBuildObjectFields{"age": UserTable.Ident("age"), "active": UserTable.Ident("active")},
			},
			expect: "json_build_object('id', ca.id, 'meta', json_build_object('active', wu.active, 'age', wu.age))",
		},
		"nested lookup": {
			input: JSONBuildObjectFields{
				"id":   AgentTable.Ident("id"),
				"meta": Lookup(AgentTable, "id", "name"),
			},
			expect: "json_build_object('id', ca.id, 'meta', json_build_object('id', ca.id, 'name', ca.name))",
		},
		"nested user lookup": {
			input: JSONBuildObjectFields{
				"id":   AgentTable.Ident("id"),
				"meta": UserLookup(UserTable),
			},
			expect: "json_build_object('id', ca.id, 'meta', json_build_object('id', wu.id, 'name', COALESCE(wu.name, wu.username)))",
		},
	}

	for scenario, tt := range tests {
		t.Run(scenario, func(t *testing.T) {
			result := JSONBuildObject(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}
