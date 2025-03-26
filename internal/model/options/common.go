package options

import (
	"slices"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql/builder"
)

type (
	orderBy       map[string]builder.OrderDirection
	OrderByOption interface {
		OrderBy() orderBy
		OrderByField(name string) (string, builder.OrderDirection, bool)
		WithOrderBy(field string, order builder.OrderDirection)
	}
)

var _ OrderByOption = (*orderBy)(nil)

func (o *orderBy) OrderBy() orderBy {
	return *o
}

func (o *orderBy) OrderByField(name string) (string, builder.OrderDirection, bool) {
	if v, ok := (*o)[name]; ok {
		return name, v, ok
	}

	return "", 0, false
}

func (o *orderBy) WithOrderBy(field string, order builder.OrderDirection) {
	if o == nil || *o == nil {
		*o = make(map[string]builder.OrderDirection) // Initialize the map if it's nil
	}

	(*o)[field] = order
}

type (
	fields       []string
	FieldsOption interface {
		Fields() fields
		WithField(field string)
	}
)

var _ FieldsOption = (*fields)(nil)

func (f *fields) Fields() fields {
	return *f
}

func (f *fields) WithField(field string) {
	if f == nil || *f == nil {
		*f = make([]string, 0)
	}

	if !slices.Contains(*f, field) {
		*f = append(*f, field)
	}
}

type (
	derived        map[string]*Derived
	DerivedOptions interface {
		Derived() derived
		DerivedByName(name string) *Derived
		WithDerived(name string, derived *Derived)
	}
)

var _ DerivedOptions = (*derived)(nil)

func (d *derived) Derived() derived {
	if d == nil || *d == nil {
		return make(derived) // Prevent nil map access
	}

	return *d
}

func (d *derived) DerivedByName(name string) *Derived {
	if d == nil || *d == nil {
		return &Derived{} // Prevent nil map access
	}

	if v, ok := (*d)[name]; ok {
		return v
	}

	return &Derived{}
}

func (d *derived) WithDerived(name string, derived *Derived) {
	if d == nil || *d == nil {
		*d = make(map[string]*Derived) // Initialize the map if it's nil
	}

	(*d)[name] = derived
}
