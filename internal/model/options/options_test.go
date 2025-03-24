package options

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithID(t *testing.T) {
	type expected struct {
		single   int64
		multiple []int64
	}

	tests := map[string]struct {
		id       []int64
		expected expected
	}{
		"single id": {
			id:       []int64{1},
			expected: expected{single: 1, multiple: []int64{1}},
		},
		"duplicated id value": {
			id:       []int64{1, 1, 1},
			expected: expected{single: 1, multiple: []int64{1}},
		},
		"multiple id": {
			id:       []int64{1, 2, 3},
			expected: expected{single: 3, multiple: []int64{1, 2, 3}},
		},
	}

	options := []struct {
		options any
		err     error
	}{
		{
			options: &Read{},
			err:     nil,
		},
		{
			options: &Search{},
			err:     nil,
		},
	}

	for _, o := range options {
		t.Run(fmt.Sprintf("%T", o.options), func(t *testing.T) {
			for scenario, tt := range tests {
				opts := DeepCopy(o.options)
				t.Run(scenario, func(t *testing.T) {
					for _, id := range tt.id {
						err := WithID(id)(opts)
						if o.err != nil {
							require.ErrorIs(t, err, o.err)
						} else {
							require.NoError(t, err)
						}
					}

					if v, ok := opts.(interface{ ID() int64 }); ok {
						t.Log("options implements ID() int64 interface")

						assert.Equal(t, tt.expected.single, v.ID())
					}

					if v, ok := opts.(interface{ IDs() []int64 }); ok {
						t.Log("options implements IDs() int64 interface")

						assert.Equal(t, tt.expected.multiple, v.IDs())
					}
				})
			}
		})
	}
}

func TestWithIDs(t *testing.T) {
	tests := map[string]struct {
		ids      []int64
		expected []int64
	}{
		"single id": {
			ids:      []int64{1},
			expected: []int64{1},
		},
		"duplicated id value": {
			ids:      []int64{1, 1, 1},
			expected: []int64{1},
		},
		"multiple id": {
			ids:      []int64{1, 2, 3},
			expected: []int64{1, 2, 3},
		},
	}

	options := []struct {
		options any
		err     error
	}{
		{
			options: &Read{},
			err:     ErrInsufficientRequestCapabilities,
		},
		{
			options: &Search{},
			err:     nil,
		},
	}

	for _, o := range options {
		t.Run(fmt.Sprintf("%T", o.options), func(t *testing.T) {
			for scenario, tt := range tests {
				opts := DeepCopy(o.options)
				t.Run(scenario, func(t *testing.T) {
					err := WithIDs(tt.ids...)(opts)
					if o.err != nil {
						require.ErrorIs(t, err, o.err)
					} else {
						require.NoError(t, err)
					}

					if v, ok := opts.(interface{ IDs() []int64 }); ok {
						t.Log("options implements IDs() int64 interface")

						assert.Equal(t, tt.expected, v.IDs())
					}
				})
			}
		})
	}
}

func TestWithFields(t *testing.T) {
	type expected struct {
		fields  fields
		derived derived
	}

	tests := map[string]struct {
		fields   []string
		expected expected
	}{
		"fields": {
			fields: []string{"id", "name"},
			expected: expected{
				fields: fields{"id", "name"},
			},
		},
		"duplicated fields": {
			fields: []string{"id", "name", "name"},
			expected: expected{
				fields: fields{"id", "name"},
			},
		},
		"derived fields": {
			fields: []string{
				"id",
				"name",
				"created_by.id",
				"created_by.name",
			},
			expected: expected{
				fields: fields{"id", "name", "created_by"},
				derived: derived{
					"created_by": {
						fields: fields{"id", "name"},
					},
				},
			},
		},
		"duplicated derived fields": {
			fields: []string{
				"id",
				"name",
				"name",
				"created_by.id",
				"created_by.name",
				"created_by.name",
			},
			expected: expected{
				fields: fields{"id", "name", "created_by"},
				derived: derived{
					"created_by": {
						fields: fields{"id", "name"},
					},
				},
			},
		},
		"nested derived fields": {
			fields: []string{
				"id",
				"name",
				"created_by.id",
				"created_by.name",
				"created_by.manager.id",
				"created_by.manager.role",
			},
			expected: expected{
				fields: fields{"id", "name", "created_by"},
				derived: derived{
					"created_by": {
						fields: fields{"id", "name", "manager"},
						derived: derived{
							"manager": {
								fields: fields{"id", "role"},
							},
						},
					},
				},
			},
		},
	}

	options := []struct {
		options any
		err     error
	}{
		{
			options: &Read{},
			err:     nil,
		},
		{
			options: &Search{},
			err:     nil,
		},
	}

	for _, o := range options {
		t.Run(fmt.Sprintf("%T", o.options), func(t *testing.T) {
			for scenario, tt := range tests {
				opts := DeepCopy(o.options)
				t.Run(scenario, func(t *testing.T) {
					err := WithFields(tt.fields)(opts)
					if o.err != nil {
						require.ErrorIs(t, err, o.err)
					} else {
						require.NoError(t, err)
					}

					if v, ok := opts.(FieldsOption); ok {
						t.Log("options implements FieldOption interface")

						assert.Equal(t, tt.expected.fields, v.Fields())
					}

					if v, ok := opts.(DerivedOptions); ok {
						t.Log("options implements DerivedOptions interface")

						assertDerivedEqual(t, tt.expected.derived, v.Derived())
					}
				})
			}
		})
	}
}

func TestWithOrder(t *testing.T) {
	type expected struct {
		orderBy orderBy
		derived map[string]*Derived
	}

	tests := map[string]struct {
		orderBy  []string
		expected expected
	}{
		"order by": {
			orderBy: []string{"+name"},
			expected: expected{
				orderBy: orderBy{
					"name": OrderDirectionASC,
				},
			},
		},
		"multiple order by": {
			orderBy: []string{"-name", "+created_by", "created_at"},
			expected: expected{
				orderBy: orderBy{
					"name":       OrderDirectionDESC,
					"created_by": OrderDirectionASC,
					"created_at": OrderDirectionASC,
				},
			},
		},
		"derived order by": {
			orderBy: []string{"id", "name.id"},
			expected: expected{
				orderBy: orderBy{
					"id": OrderDirectionASC,
				},
				derived: map[string]*Derived{
					"name": {
						orderBy: orderBy{
							"id": OrderDirectionASC,
						},
					},
				},
			},
		},
		"nested derived order by": {
			orderBy: []string{"-id", "name.id", "+name.common", "-created_by.manager.id"},
			expected: expected{
				orderBy: orderBy{
					"id": OrderDirectionDESC,
				},
				derived: map[string]*Derived{
					"name": {
						orderBy: orderBy{
							"id":     OrderDirectionASC,
							"common": OrderDirectionASC,
						},
					},
					"created_by": {
						derived: map[string]*Derived{
							"manager": {
								orderBy: orderBy{
									"id": OrderDirectionDESC,
								},
							},
						},
					},
				},
			},
		},
	}

	options := []struct {
		options any
		err     error
	}{
		{
			options: &Read{},
			err:     ErrInsufficientRequestCapabilities,
		},
		{
			options: &Search{},
			err:     nil,
		},
	}

	for _, o := range options {
		t.Run(fmt.Sprintf("%T", o.options), func(t *testing.T) {
			for scenario, tt := range tests {
				opts := DeepCopy(o.options)
				t.Run(scenario, func(t *testing.T) {
					err := WithOrder(tt.orderBy...)(opts)
					if o.err != nil {
						require.ErrorIs(t, err, o.err)

						return
					} else {
						require.NoError(t, err)
					}

					if v, ok := opts.(OrderByOption); ok {
						assert.Equal(t, tt.expected.orderBy, v.OrderBy())
					}

					if v, ok := opts.(DerivedOptions); ok {
						assertDerivedEqual(t, tt.expected.derived, v.Derived())
					}
				})
			}
		})
	}
}

func assertDerivedEqual(t *testing.T, expected, actual map[string]*Derived) {
	require.Len(t, actual, len(expected), "derived maps have different lengths")

	for key, expectedDerived := range expected {
		actualDerived, exists := actual[key]
		require.True(t, exists, "expected derived field %s not found", key)

		// Ensure fields match ignoring order
		assert.ElementsMatch(t, expectedDerived.Fields(), actualDerived.Fields(), "mismatch in fields for derived key: %s", key)
		assert.Equal(t, expectedDerived.OrderBy(), actualDerived.OrderBy(), "mismatch in orderBy for derived key: %s", key)

		// Recur for deeper derived structures
		assertDerivedEqual(t, expectedDerived.Derived(), actualDerived.Derived())
	}
}

func DeepCopy[T any](src T) T {
	return deepCopy(reflect.ValueOf(src)).Interface().(T)
}

func deepCopy(v reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}

		copy := reflect.New(v.Elem().Type())
		copy.Elem().Set(deepCopy(v.Elem()))

		return copy
	case reflect.Struct:
		copy := reflect.New(v.Type()).Elem()
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				copy.Field(i).Set(deepCopy(v.Field(i)))
			}
		}

		return copy
	case reflect.Slice:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}

		copy := reflect.MakeSlice(v.Type(), v.Len(), v.Cap())
		for i := 0; i < v.Len(); i++ {
			copy.Index(i).Set(deepCopy(v.Index(i)))
		}

		return copy
	case reflect.Map:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}

		copy := reflect.MakeMap(v.Type())
		for _, key := range v.MapKeys() {
			copy.SetMapIndex(key, deepCopy(v.MapIndex(key)))
		}

		return copy
	default:
		return v
	}
}
